package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

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
}

type ExpectedDir struct {
	Name            string
	NumOfRawFiles   int
	NumOfSumFiles   int
	AsicsHkPresent  bool
	ExternHkPresent bool
}

func TestRefresh(t *testing.T) {
	if err := RefreshDbContents(testdb, "testdata"); err != nil {
		t.Fatalf("Error running RefreshDbContents: %s", err)
	}

	var count int
	testdb.Model(&Acquisition{}).Count(&count)
	if count != 4 {
		t.Fatalf("Wrong number of acquisitions: %d", count)
	}

	expecteddirs := []ExpectedDir{
		{Name: "2018-04-06_14.20.35__testbackups", NumOfRawFiles: 1, NumOfSumFiles: 1, AsicsHkPresent: true},
		{Name: "2018-05-22_13.33.56__mytest", NumOfRawFiles: 0, NumOfSumFiles: 0, ExternHkPresent: true},
		{Name: "2018-05-22_13.38.15__test_backhome", NumOfRawFiles: 0, NumOfSumFiles: 0, ExternHkPresent: true},
		{Name: "2018-05-22_15.22.22__test_withGPS", NumOfRawFiles: 0, NumOfSumFiles: 0, ExternHkPresent: true},
	}
	for _, dir := range expecteddirs {
		var acq Acquisition
		res := testdb.Where("directoryname = ?", dir.Name).First(&acq)
		if res.RecordNotFound() {
			t.Fatalf("Acquisition \"%s\" not found in the database", dir.Name)
		}

		if res := testdb.Model(&acq).Related(&acq.RawFiles).Error; res != nil {
			t.Fatalf("Error for acquisition %d (%s): %s", acq.ID, acq.Directoryname, res)
		}
		if len(acq.RawFiles) != dir.NumOfRawFiles {
			t.Fatalf("Wrong number of raw files for \"%s\", it is %d but it should be %d (%v)",
				dir.Name, len(acq.RawFiles), dir.NumOfRawFiles, acq.RawFiles)
		}

		if res := testdb.Model(&acq).Related(&acq.SumFiles).Error; res != nil {
			t.Fatalf("Error for acquisition %d (%s): %s", acq.ID, acq.Directoryname, res)
		}
		if len(acq.SumFiles) != dir.NumOfSumFiles {
			t.Fatalf("Wrong number of science files for \"%s\", it is %d but it should be %d",
				dir.Name, len(acq.SumFiles), dir.NumOfSumFiles)
		}

		if dir.AsicsHkPresent && acq.AsicHkFileName == "" {
			t.Fatalf("ASIC HK file for \"%s\" not found", dir.Name)
		}

		if dir.ExternHkPresent && acq.ExternHkFileName == "" {
			t.Fatalf("External HK file for \"%s\" not found", dir.Name)
		}

	}
}

func touch(filename string) error {
	newFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("Unable to create file \"%s\"", filename)
	}
	newFile.Close()
	return nil
}

func TestErrorsOnRefresh(t *testing.T) {
	repopath := filepath.Join("testdata", "testerrors")
	hkpath := filepath.Join(repopath, "2018-06-08_13.28.00__duplicate", "Hks")
	spuriousHkFilename1 := filepath.Join(hkpath, "hk-extern-2018.06.01.000000.fits")
	spuriousHkFilename2 := filepath.Join(hkpath, "hk-extern-2018.06.02.000000.fits")

	_ = os.RemoveAll(repopath)
	os.MkdirAll(hkpath, 0777)
	defer os.RemoveAll(repopath)

	if err := touch(spuriousHkFilename1); err != nil {
		t.Fatalf("Unable to create file \"%s\"", spuriousHkFilename1)
	}

	if err := touch(spuriousHkFilename2); err != nil {
		t.Fatalf("Unable to create file \"%s\"", spuriousHkFilename2)
	}

	if err := RefreshDbContents(testdb, repopath); err == nil {
		t.Fatal("RefreshDbContents did not signal the presence of more than an HK file")
	}
}

func TestMain(m *testing.M) {
	testdb, _ = gorm.Open("sqlite3", "file::memory:?mode=memory&cache=shared")
	defer testdb.Close()

	InitDb(testdb, &Configuration{})
	os.Exit(m.Run())
}
