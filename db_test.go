package qutedb

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

const refUserEmail = "foo@bar.com"
const refPassword = "dummy"

var testdb *gorm.DB

func TestUser(t *testing.T) {
	user, err := CreateUser(testdb, refUserEmail, refPassword, false)
	if err != nil {
		t.Fatalf("Unexpected error while creating a user: %s", err)
	}
	if user.Email != refUserEmail {
		t.Errorf("Mismatch in the name of the user (%s != %s)", user.Email, refUserEmail)
	}

	if user.Superuser {
		t.Error("Mismatch in the superuser flag")
	}

	if string(user.HashedPassword) == refPassword {
		t.Error("The password has been saved in clear text")
	}

	if _, valid, err := CheckUserPassword(testdb, refUserEmail, refPassword); !valid || err != nil {
		t.Error("The password hash algorithm is not working")
	}

	if _, valid, _ := CheckUserPassword(testdb, refUserEmail, "thisisblatantlywrong"); valid {
		t.Error("The password hash algorithm is accepting wrong passwords")
	}

	if foundUser, err := QueryUserByID(testdb, user.ID); foundUser == nil || err != nil || foundUser.ID != user.ID {
		if err != nil {
			t.Errorf("Unexpected error while querying a user: %s", err)
		} else {
			if foundUser.ID != user.ID {
				t.Errorf("QueryUserByID returns an user with the wrong ID: %v != %v",
					foundUser.ID, user.ID)
			} else {
				t.Error("QueryUserByID does not work")
			}
		}
	}

	userList, err := QueryAllUsers(testdb)
	if err != nil {
		t.Errorf("Unexpected error in QueryAllUsers: %v", err)
	}
	if len(userList) != 1 {
		t.Errorf("Expected one user in the database, found %d", len(userList))
	}
	if userList[0].Email != refUserEmail {
		t.Errorf("QueryAllUsers returned the wrong user: \"%s\" instead of \"%s\"",
			userList[0].Email, refUserEmail)
	}

	if foundUser, err := QueryUserByEmail(testdb, refUserEmail); foundUser == nil || err != nil {
		if err != nil {
			t.Errorf("Unexpected error while querying a user: %s", err)
		} else {
			t.Error("QueryUserByEmail does not work")
		}
	}

	if foundUser, err := QueryUserByEmail(testdb, strings.ToUpper(refUserEmail)); foundUser == nil || err != nil {
		if err != nil {
			t.Errorf("Unexpected error while querying a user: %s", err)
		} else {
			t.Error("QueryUserByEmail does not ignore case differences as it should")
		}
	}

	if err := DeleteUser(testdb, user); err != nil {
		t.Errorf("Unexpected error while deleting an user: %s", err)
	}

	if foundUser, err := QueryUserByEmail(testdb, refUserEmail); foundUser != nil || err != nil {
		if err != nil {
			t.Errorf("Unexpected error while querying a deleted user: %s", err)
		} else {
			t.Errorf("The following user should no longer exist: %v", *foundUser)
		}
	}
}

func TestSession(t *testing.T) {
	user, err := CreateUser(testdb, refUserEmail, refPassword, false)
	if err != nil {
		t.Fatalf("Unexpected error while creating a user: %s", err)
	}

	var session *Session
	session, err = CreateSession(testdb, user)
	if err != nil {
		t.Fatalf("Unexpected error while creating a session: %s", err)
	}
	if session == nil {
		t.Fatal("Unable to create a session")
	}
	if session.UserID != user.ID {
		t.Fatalf("Mismatch in the session/user IDs: %d != %d", session.UserID, user.ID)
	}

	var newSession *Session
	newSession, err = QuerySessionByUUID(testdb, session.UUID)
	if newSession == nil {
		t.Fatalf("Unable to retrieve an existing session from the DB")
	}
	if err != nil {
		t.Fatalf("Unexpected error while querying an existing session: %v", err)
	}

	err = DeleteSession(testdb, session.UUID)
	if err != nil {
		t.Fatalf("Unexpected error while deleting an existing session: %v", err)
	}

	newSession, err = QuerySessionByUUID(testdb, session.UUID)
	if newSession != nil {
		t.Fatalf("I've just retrieved a non-existing session!")
	} else if err != nil {
		t.Fatalf("I wasn't expecting an error here: %v", err)
	}
}

type ExpectedDir struct {
	Name            string
	DirName         string
	AcquisitionTime string
	NumOfRawFiles   int
	NumOfSumFiles   int
	AsicsHkPresent  bool
	InternHkPresent bool
	ExternHkPresent bool
	MmrHkPresent    bool
	MgcHkPresent    bool
	CalibPresent    bool
}

func TestRefresh(t *testing.T) {
	if err := RefreshDbContents(testdb, "testdata"); err != nil {
		t.Fatalf("Error running RefreshDbContents: %s", err)
	}

	var count int
	testdb.Model(&Acquisition{}).Count(&count)
	if count != 6 {
		t.Fatalf("Wrong number of acquisitions: %d", count)
	}

	expecteddirs := []ExpectedDir{
		{
			Name:            "testbackups",
			DirName:         "2018-04-06_14.20.35__testbackups",
			AcquisitionTime: TimeToCanonicalStr(time.Date(2018, 4, 6, 14, 20, 35, 0, time.UTC)),
			NumOfRawFiles:   1,
			NumOfSumFiles:   1,
			AsicsHkPresent:  true,
			CalibPresent:    false,
		},
		{
			Name:            "mytest",
			DirName:         "2018-05-22_13.33.56__mytest",
			AcquisitionTime: TimeToCanonicalStr(time.Date(2018, 5, 22, 13, 33, 56, 0, time.UTC)),
			NumOfRawFiles:   0,
			NumOfSumFiles:   0,
			ExternHkPresent: true,
			CalibPresent:    false,
		},
		{
			Name:            "test_backhome",
			DirName:         "2018-05-22_13.38.15__test_backhome",
			AcquisitionTime: TimeToCanonicalStr(time.Date(2018, 5, 22, 13, 38, 15, 0, time.UTC)),
			NumOfRawFiles:   0,
			NumOfSumFiles:   0,
			ExternHkPresent: true,
			CalibPresent:    false,
		},
		{
			Name:            "test_withGPS",
			DirName:         "2018-05-22_15.22.22__test_withGPS",
			AcquisitionTime: TimeToCanonicalStr(time.Date(2018, 5, 22, 15, 22, 22, 0, time.UTC)),
			NumOfRawFiles:   0,
			NumOfSumFiles:   0,
			ExternHkPresent: true,
			CalibPresent:    false,
		},
		{
			Name:            "RF_switch_cont_13_34",
			DirName:         "2019-05-07_18.11.29__RF_switch_cont_13_34",
			AcquisitionTime: TimeToCanonicalStr(time.Date(2019, 5, 7, 18, 11, 29, 0, time.UTC)),
			NumOfRawFiles:   2,
			NumOfSumFiles:   2,
			InternHkPresent: true,
			ExternHkPresent: true,
			MmrHkPresent:    true,
			MgcHkPresent:    true,
			CalibPresent:    false,
		},
		{
			Name:            "Test-CalibrationSource-Timeconstant",
			DirName:         "2022-04-05_15.54.04__Test-CalibrationSource-Timeconstant",
			AcquisitionTime: TimeToCanonicalStr(time.Date(2022, 4, 5, 15, 54, 4, 0, time.UTC)),
			NumOfRawFiles:   2,
			NumOfSumFiles:   2,
			InternHkPresent: false,
			ExternHkPresent: false,
			MmrHkPresent:    false,
			MgcHkPresent:    false,
			CalibPresent:    true,
		},
	}
	for _, dir := range expecteddirs {
		var acq Acquisition
		res := testdb.Where("directoryname = ?", dir.DirName).First(&acq)
		if res.RecordNotFound() {
			t.Fatalf("Acquisition \"%s\" not found in the database", dir.Name)
		}

		if acq.Name != dir.Name {
			t.Fatalf("Acquisition name mismatch: \"%s\" != \"%s\"", acq.Name, dir.Name)
		}

		if acq.AcquisitionTime != dir.AcquisitionTime {
			t.Fatalf("Creation time mismatch: \"%v\" != \"%v\"", acq.AcquisitionTime, dir.AcquisitionTime)
		}

		if res := testdb.Model(&acq).Related(&acq.RawFiles).Error; res != nil {
			t.Fatalf("Error for acquisition %d (%s): %s", acq.ID, acq.Directoryname, res)
		}
		if len(acq.RawFiles) != dir.NumOfRawFiles {
			t.Fatalf("Wrong number of raw files for \"%s\", it is %d but it should be %d (%v)",
				dir.Name, len(acq.RawFiles), dir.NumOfRawFiles, acq.RawFiles)
		}
		if len(acq.RawFiles) > 0 {
			if acq.RawFiles[0].AsicNumber != 1 {
				t.Fatalf("Wrong ASIC number (%d) for raw file %s",
					acq.RawFiles[0].AsicNumber,
					acq.RawFiles[0].FileName)
			}
		}

		if res := testdb.Model(&acq).Related(&acq.SumFiles).Error; res != nil {
			t.Fatalf("Error for acquisition %d (%s): %s", acq.ID, acq.Directoryname, res)
		}
		if len(acq.SumFiles) != dir.NumOfSumFiles {
			t.Fatalf("Wrong number of science files for \"%s\", it is %d but it should be %d",
				dir.Name, len(acq.SumFiles), dir.NumOfSumFiles)
		}
		if len(acq.SumFiles) > 0 {
			if acq.SumFiles[0].AsicNumber != 1 {
				t.Fatalf("Wrong ASIC number (%d) for science file %s",
					acq.SumFiles[0].AsicNumber,
					acq.SumFiles[0].FileName)
			}
		}

		if dir.AsicsHkPresent && acq.AsicHkFileName == "" {
			t.Fatalf("ASIC HK file for \"%s\" not found", dir.Name)
		}

		if dir.InternHkPresent && acq.InternHkFileName == "" {
			t.Fatalf("Internal HK file for \"%s\" not found", dir.Name)
		}

		if dir.ExternHkPresent && acq.ExternHkFileName == "" {
			t.Fatalf("External HK file for \"%s\" not found", dir.Name)
		}

		if dir.MmrHkPresent && acq.MmrHkFileName == "" {
			t.Fatalf("MMR HK file for \"%s\" not found", dir.Name)
		}

		if dir.MgcHkPresent && acq.MgcHkFileName == "" {
			t.Fatalf("MGC HK file for \"%s\" not found", dir.Name)
		}

		if dir.CalibPresent && (acq.CalDataFileName == "" || acq.CalConfFileName == "") {
			t.Fatalf("Calibration HK files for \"%s\" not found", dir.Name)
		}
	}
}

func touch(name string) error {
	return ioutil.WriteFile(name, nil, 0644)
}

var app *App

func TestMain(m *testing.M) {
	testdb, _ = gorm.Open("sqlite3", "file::memory:?mode=memory&cache=shared")
	defer testdb.Close()

	InitDb(testdb, &Configuration{})
	app = &App{
		config: nil,
		db:     testdb,
	}
	os.Exit(m.Run())
}
