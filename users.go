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

// This file includes the endpoints for HTTP requests related to user
// management

package main

import "net/http"

func (app *App) modifyUserHandler(w http.ResponseWriter, r *http.Request) error {
	session, _ := app.session(w, r)
	if session == nil {
		panic("No session available while accessing a reserved page")
	}

	user, _ := QueryUserByID(app.db, session.UserID)
	if user == nil {
		panic("No user associated with an existing session")
	}

	return generateHTML(w, user, "layout", "private.navbar", "usermod")
}

func (app *App) userListHandler(w http.ResponseWriter, r *http.Request) error {
	userList, err := QueryAllUsers(app.db)
	if err != nil {
		return Error{err: err, msg: "Unable to retrieve the list of users"}
	}

	return generateHTML(w, userList, "layout", "private.navbar", "userlist")
}

func (app *App) createUser(w http.ResponseWriter, r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}

	if r.PostFormValue("password") != r.PostFormValue("confirm-password") {
		return Error{err: err, msg: "Passwords do not match"}
	}

	// Check if an user with the given email already exists in the database
	user, err := QueryUserByEmail(app.db, r.PostFormValue("email"))
	if err != nil {
		return err
	}
	if user != nil {
		return Error{err: err, msg: "Invalid user"}
	}

	user, err = CreateUser(
		app.db,
		r.PostFormValue("email"),
		r.PostFormValue("password"),
		r.PostFormValue("superuser") != "",
	)
	if err != nil {
		return err
	}

	http.Redirect(w, r, "/userlist", 302)
	return nil
}

func (app *App) createUserHandler(w http.ResponseWriter, r *http.Request) error {
	return generateHTML(w, nil, "layout", "private.navbar", "createuser")
}
