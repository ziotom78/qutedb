/*
The MIT License

Copyright (c) 2018 Maurizio Tomasi

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package qutedb

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/elithrar/simple-scrypt"
	"github.com/gobuffalo/uuid"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3" // We need SQLite3 in order for GORM to use it
	log "github.com/sirupsen/logrus"
)

// An User is somebody which can have (read-only) access to the data
type User struct {
	gorm.Model
	Email          string `gorm:"unique_index"`
	HashedPassword []byte
	Superuser      bool
}

// A Session records who is currently allowed to access the site. This only
// happens if a user has successfully logged in.
type Session struct {
	gorm.Model
	UUID   string `gorm:"size:36;unique_index"`
	UserID uint
}

// A RawDataFile represents the file containing raw data acquired with one ASIC
type RawDataFile struct {
	ID            int    `json:"id" gorm:"primary_key"`
	FileName      string `json:"file_name"`
	AsicNumber    int    `json:"asic_number"`
	AcquisitionID int    `json:"-"`
}

// A SumDataFile represents the file containing science data acquired with one ASIC
type SumDataFile struct {
	ID            int    `json:"id" gorm:"primary_key"`
	FileName      string `json:"file_name"`
	AsicNumber    int    `json:"asic_number"`
	AcquisitionID int    `json:"-"`
}

// An Acquisition represents a set of files within a folder in the repository
type Acquisition struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	Name          string `json:"name"`
	Directoryname string `json:"directory_name" gorm:"unique_index"`
	// We encode the acquisition time as a string in order to have
	// full control on the formatting, which is always "YYYY-MM-DDThh:mm:ss"
	AcquisitionTime    string        `json:"acquisition_time"`
	RawFiles           []RawDataFile `json:"-"`
	SumFiles           []SumDataFile `json:"-"`
	AsicHkFileName     string        `json:"-"`
	InternHkFileName   string        `json:"-"`
	ExternHkFileName   string        `json:"-"`
	CryostatHkFileName string        `json:"-"`
}

// TimeToCanonicalStr converts a standard date/time into
// a string which is formatted according to the template "YYYYMMSShhmmss".
// This is the format used in Acquisition.AcquisitionTime
func TimeToCanonicalStr(t time.Time) string {
	return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}

// InitDb creates all the tables in the database. It takes care of not raising
// errors if the tables are already present. You should call this function only
// once during the lifetime of the program, as it resets all open sessions.
func InitDb(db *gorm.DB, config *Configuration) error {
	db.AutoMigrate(
		&User{},
		&Session{},
		&RawDataFile{},
		&SumDataFile{},
		&Acquisition{},
	)

	// Clear all existing sessions in the database. Ignore any error
	db.Delete(&Session{})

	return db.Error
}

// HkDirName returns the name of the test directory containing the housekeeping files
func HkDirName(folder string) string {
	return path.Join(folder, "Hks")
}

// RawDirName returns the name of the test directory containing raw files
func RawDirName(folder string) string {
	return path.Join(folder, "Raws")
}

// SumDirName returns the name of the test directory containing raw files
func SumDirName(folder string) string {
	return path.Join(folder, "Sums")
}

// findMultipleFiles returns a sorted list of files matching "mask" in the
// specified "path"
func findMultipleFiles(path string, mask string) ([]string, error) {
	filenames, err := filepath.Glob(filepath.Join(path, mask))
	if len(filenames) == 0 {
		return []string{}, err
	}

	// Sort the files in lexicographic order
	sort.Strings(filenames)

	result := []string{}
	for _, curname := range filenames {
		fi, err := os.Stat(curname)
		if err != nil || !fi.Mode().IsRegular() {
			continue
		}

		result = append(result, curname)
	}

	return result, nil
}

// findOneMatchingFile looks for the files matching "mask" (a file name pattern
// built using POSIX wildcards). If no matches are found, it returns "". If one
// match is found, it returns the name of the file. If more tha one match is
// found, it returns an error. matching files were found.
func findOneMatchingFile(path string, mask string) (string, error) {
	filenames, err := findMultipleFiles(path, mask)
	if err != nil || len(filenames) == 0 {
		return "", err
	}

	if len(filenames) > 1 {
		return "", fmt.Errorf("Found more than one file (%d) matching the mask \"%s\"",
			len(filenames), mask)
	}

	return filenames[0], nil
}

// refreshFolder scans a folder containing *one* acquisition and updates the
// database accordingly. This function does not check whether "folderPath" is
// really within the repository or not.
func refreshFolder(db *gorm.DB, folderPath string) error {
	dirname := filepath.Base(folderPath)
	acquisitionTime, err := time.Parse("2006-01-02_15.04.05", dirname[:19])
	if err != nil {
		panic(fmt.Sprintf("Wrong time in string \"%s\"", dirname))
	}

	newacq := Acquisition{
		Name:            dirname[21:],
		Directoryname:   dirname,
		AcquisitionTime: TimeToCanonicalStr(acquisitionTime),
	}

	// Check if the folder is already present in the db
	result := db.Where("directoryname = ?", newacq.Directoryname).First(&Acquisition{})
	if !result.RecordNotFound() {
		return nil
	}

	// Check for the presence of housekeeping files
	hkDir := HkDirName(folderPath)
	filename, err := findOneMatchingFile(hkDir, "conf-asics-????.??.??.??????.fits")
	if err != nil {
		return err
	}
	if filename != "" {
		newacq.AsicHkFileName = filename
	}

	filename, err = findOneMatchingFile(hkDir, "hk-intern-????.??.??.??????.fits")
	if err != nil {
		return err
	}
	if filename != "" {
		newacq.InternHkFileName = filename
	}

	filename, err = findOneMatchingFile(hkDir, "hk-extern-????.??.??.??????.fits")
	if err != nil {
		return err
	}
	if filename != "" {
		newacq.ExternHkFileName = filename
	}

	// TODO: Cryostat thermometers will need to be considered at this point,
	// once the mask for their files is finalized

	asicRe := regexp.MustCompile("asic([0-9]+)")
	if rawFiles, err := findMultipleFiles(
		RawDirName(folderPath),
		"raw-asic*-????.??.??.??????.fits",
	); err == nil && len(rawFiles) > 0 {
		for _, filename := range rawFiles {
			matches := asicRe.FindStringSubmatch(filepath.Base(filename))
			asicNum, err := strconv.Atoi(matches[1])
			if err != nil {
				panic(fmt.Sprintf("Unexpected error in Atoi(\"%s\"): %s", matches[1], err))
			}
			newacq.RawFiles = append(
				newacq.RawFiles,
				RawDataFile{
					FileName:   filename,
					AsicNumber: asicNum,
				},
			)
		}
	}

	if sumFiles, err := findMultipleFiles(
		SumDirName(folderPath),
		"science-asic*-????.??.??.??????.fits",
	); err == nil && len(sumFiles) > 0 {
		for _, filename := range sumFiles {
			matches := asicRe.FindStringSubmatch(filepath.Base(filename))
			asicNum, err := strconv.Atoi(matches[1])
			if err != nil {
				panic(fmt.Sprintf("Unexpected error in Atoi(\"%s\"): %s",
					matches[1], err))
			}
			newacq.SumFiles = append(
				newacq.SumFiles,
				SumDataFile{
					FileName:   filename,
					AsicNumber: asicNum,
				},
			)
		}
	}

	log.WithFields(log.Fields{
		"new_acquisition": newacq,
	}).Info("Going to create new acquisition")

	if db.Create(&newacq).Error != nil {
		return fmt.Errorf("Error while creating a new acquisition for \"%s\": %s",
			folderPath, db.Error)
	}

	return nil
}

// RefreshDbContents scans the repository for any file that is missing from the
// database, and create an entry for each of them.
func RefreshDbContents(db *gorm.DB, repositoryPath string) error {
	return filepath.Walk(repositoryPath, func(
		path string,
		info os.FileInfo,
		err error,
	) error {
		if err != nil {
			return err
		}

		log.WithFields(log.Fields{
			"folder_name": path,
		}).Debug("Walking into directory")

		if !info.IsDir() {
			return nil
		}

		matched, err := filepath.Match("????-??-??_??.??.??__*", filepath.Base(path))
		if err != nil {
			return err
		}

		if matched {
			log.WithFields(log.Fields{
				"folder_name": path,
			}).Info("Processing folder")

			if err := refreshFolder(db, path); err != nil {
				return err
			}

			// This directory has been processed, so don't walk into it
			return filepath.SkipDir
		}

		return nil
	})
}

// CreateUser creates a new "User" object and initializes it with the hash of
// the password and the other parameters as well. The new object is saved in the
// database.
func CreateUser(
	db *gorm.DB,
	email string,
	password string,
	superuser bool,
) (*User, error) {

	hash, err := scrypt.GenerateFromPassword([]byte(password), scrypt.DefaultParams)
	if err != nil {
		return nil, err
	}

	user := User{
		Email:          strings.ToLower(email),
		HashedPassword: hash,
		Superuser:      superuser,
	}
	if err := db.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// DeleteUser removes an user from the database
func DeleteUser(db *gorm.DB, user *User) error {
	// Use Unscoped to avoid soft deletions
	return db.Unscoped().Delete(user).Error
}

// UpdateUserPassword changes the password associated with a user
func UpdateUserPassword(db *gorm.DB, user *User, newPassword string) error {
	hash, err := scrypt.GenerateFromPassword([]byte(newPassword), scrypt.DefaultParams)
	log.WithFields(log.Fields{
		"user":     user.Email,
		"new_hash": string(hash),
	}).Info("New password hash")
	if err != nil {
		return err
	}

	return db.Model(user).Update("hashed_password", hash).Error
}

// QueryUserByID searches in the database for an user with the
// matching user ID and returns a pointer to a User structure. If the
// user is not found, the pointer is nil. The "error" variable is set
// to something else than nil only if a real error is occurred.
func QueryUserByID(db *gorm.DB, userID uint) (*User, error) {
	var user User
	result := db.Where("ID = ?", userID).First(&user)

	if result.RecordNotFound() {
		return nil, nil
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return &user, nil
}

// QueryAllUsers returns a list of all the users defined in the database
func QueryAllUsers(db *gorm.DB) ([]User, error) {
	var users []User
	db.Order("Email").Find(&users)
	return users, nil
}

// QueryUserByEmail searches in the database for an user with the matching email
// and returns a poitner to a User structure. If the user is not found, the
// pointer is nil. The "error" variable is set to something else than nil only
// if a real error is occurred.
func QueryUserByEmail(db *gorm.DB, email string) (*User, error) {
	var user User
	result := db.Where("Email = ?", strings.ToLower(email)).First(&user)

	if result.RecordNotFound() {
		return nil, nil
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return &user, nil
}

// CheckUserPassword checks that an user with the specified email and password is
// in the database. If found, return the tuple (userID, true, nil). If the user
// does not exist, or if the passwords don't match, return (0, false, nil). In
// the event of an error, the last element in the returned tuple identifies the
// error.
func CheckUserPassword(db *gorm.DB, email string, password string) (uint, bool, error) {
	var user User
	if db.Where("Email = ?", email).First(&user).RecordNotFound() {
		return 0, false, nil
	} else if db.Error != nil {
		return 0, false, db.Error
	}

	if scrypt.CompareHashAndPassword(user.HashedPassword, []byte(password)) != nil {
		return 0, false, nil
	}

	return user.ID, true, nil
}

// CreateSession inserts a new "Session" object in the database. The object is
// uniquely identified by its UUID.
func CreateSession(db *gorm.DB, user *User) (*Session, error) {
	// If the user already has an active session, return it
	var existingSessions []Session
	if err := db.Model(user).Related(&existingSessions).Error; err != nil {
		return nil, err
	}

	if len(existingSessions) > 0 {
		return &existingSessions[0], nil
	}

	newUUID, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	newSession := Session{UUID: newUUID.String(), UserID: user.ID}
	if err := db.Create(&newSession).Error; err != nil {
		return nil, err
	}

	return &newSession, nil
}

// DeleteSession finds a session with a matching UUID in the database and
// deletes it. If no session is found, returns silently without signaling
// any error.
func DeleteSession(db *gorm.DB, UUID string) error {
	return db.Delete(Session{}, "UUID = ?", UUID).Error
}

// QuerySessionByUUID searches for an active session and returns a Session
// object if found.
func QuerySessionByUUID(db *gorm.DB, UUID string) (*Session, error) {
	var session Session
	result := db.Where("UUID = ?", UUID).First(&session)

	if result.RecordNotFound() {
		return nil, nil
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return &session, nil
}

// QueryAcquisition returns an Aquisition object with all its fields
// properly filled
func QueryAcquisition(db *gorm.DB, acqtime string) (*Acquisition, error) {
	var acq Acquisition
	if err := db.
		Where("acquisition_time = ?", acqtime).First(&acq).Error; err != nil {
		return &acq, Error{
			err: err,
			msg: fmt.Sprintf("Unable to query the database for acquisition with ID %s",
				acqtime),
		}
	}

	if err := db.
		Joins("JOIN acquisitions ON raw_data_files.acquisition_id = acquisitions.id").
		Where("acquisitions.id = ?", acq.ID).
		Find(&acq.RawFiles).Error; err != nil {
		return &acq, Error{
			err: err,
			msg: fmt.Sprintf("Unable to query for raw files belonging to ID %s",
				acqtime),
		}
	}

	if err := db.
		Joins("JOIN acquisitions ON sum_data_files.acquisition_id = acquisitions.id").
		Where("acquisitions.id = ?", acq.ID).
		Find(&acq.SumFiles).Error; err != nil {
		return &acq, Error{
			err: err,
			msg: fmt.Sprintf("Unable to query for science files belonging to ID %s",
				acqtime),
		}
	}

	return &acq, nil
}
