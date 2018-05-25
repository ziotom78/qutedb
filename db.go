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
	"database/sql"

	log "github.com/sirupsen/logrus"
)

// createDbTables creates all the tables in the SQLite3 database. It takes care
// of not raising errors if the tables are already present.
func createDbTables(db *sql.DB) {
	sqlStmt := `
CREATE TABLE IF NOT EXISTS users (
	id PRIMARY KEY,
	name TEXT NOT NULL,
	email TEXT,
	created_at TEXT DEFAULT CURRENT_TIMESTAMP 
)

CREATE TABLE IF NOT EXISTS sessions (
	id PRIMARY KEY,
	uuid NUMBER NOT NULL,
	user_id NUMBER NOT NULL,
	created_at TEXT DEFAULT CURRENT_TIMESTAMP
)`

	if _, err := db.Exec(sqlStmt); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Unable to create the tables in the database")
	}
}
