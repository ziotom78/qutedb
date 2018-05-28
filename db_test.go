package main

import (
	"os"
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

func TestMain(m *testing.M) {
	testdb, _ = gorm.Open("sqlite3", "file::memory:?mode=memory&cache=shared")
	defer testdb.Close()

	InitDb(testdb, Configuration{})
	os.Exit(m.Run())
}
