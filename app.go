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

package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"

	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"

	"github.com/gorilla/securecookie"
)

// An App is a structure which encapsulate the whole state of the application
type App struct {
	config        *Configuration
	db            *gorm.DB
	cookieEncoder *securecookie.SecureCookie
}

// NewApp creates a new application and performs a number of initializations.
func NewApp() *App {
	config := createConfiguration()

	// Before calling configureLogging, we need to initialize the output file in
	// "main", so that the file gets closed automatically when the program
	// closes
	switch config.LogOutput {
	case "-":
		log.SetOutput(os.Stderr)
	case "--":
		log.SetOutput(os.Stdout)
	default:
		logfile, err := os.Create(config.LogOutput)
		if err != nil {
			panic(fmt.Errorf("unable to create log file \"%s\": %s", config.LogOutput, err))
		}
		defer func() { _ = logfile.Close() }()

		log.SetOutput(logfile)
	}

	// Now we can configure the logger
	configureLogging(config)

	log.WithFields(log.Fields{
		"GOOS":      runtime.GOOS,
		"GOARCH":    runtime.GOARCH,
		"COMPILER":  runtime.Compiler,
		"log_level": log.GetLevel(),
	}).Info("Starting the application")

	log.WithFields(log.Fields{
		"configfile": config.ConfigurationFileName,
	}).Info("Configuration has been read")

	hashKey := config.CookieHashKey
	blockKey := config.CookieBlockKey

	return &App{
		config:        config,
		db:            nil,
		cookieEncoder: securecookie.New(hashKey, blockKey),
	}
}

// CreateDefaultUser creates a superuser with a standard password, if
// no user exists.
func (app *App) CreateDefaultUser() error {
	var user User
	result := app.db.First(&user)
	if !result.RecordNotFound() {
		return nil
	}

	_, err := CreateUser(app.db, "admin@localhost", "changeme", true)
	return err
}

// Run opens the database and starts the main loop.
func (app *App) Run() {
	log.WithFields(log.Fields{
		"database_file": app.config.DatabaseFile,
	}).Info("Going to establish a connection to database")
	db, err := gorm.Open("sqlite3", app.config.DatabaseFile)
	if err != nil {
		log.WithFields(log.Fields{
			"database_file": app.config.DatabaseFile,
			"error":         err,
		}).Fatalf("Unable to open database")
	}
	defer db.Close()

	if err := InitDb(db, app.config); err != nil {
		panic(fmt.Sprintf("Unable to initialize the database: %s", err))
	}
	app.db = db

	// If no user exists, create a default superuser
	if err := app.CreateDefaultUser(); err != nil {
		log.Fatalf("Unable to create default user")
	}

	// Refresh the contents of the database
	log.WithFields(log.Fields{
		"repository": app.config.RepositoryPath,
	}).Info("Refreshing the database")
	if err := RefreshDbContents(db, app.config.RepositoryPath); err != nil {
		panic(fmt.Sprintf("Unable to refresh the database: %s", err))
	}

	log.WithFields(log.Fields{
		"server":      app.config.ServerName,
		"port_number": app.config.PortNumber,
	}).Info("Main loop is going to start now")

	app.serve()
}

// Error contains information about an HTTP error
type Error struct {
	err  error
	msg  string
	code int
}

func (e Error) Error() string {
	code := e.code
	if code == 0 {
		code = http.StatusInternalServerError
	}
	return fmt.Sprintf("%s: %v (code=%d)", e.msg, e.err, code)
}
