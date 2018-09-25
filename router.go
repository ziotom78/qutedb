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
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/mux"
)

// logMiddleware is a middleware function used by the router. It logs a number
// of information about the HTTP request that is being processed.
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(log.Fields{
			"request_uri": r.RequestURI,
			"remote_addr": r.RemoteAddr,
			"host":        r.Host,
			"method":      r.Method,
		}).Info("Routing a request")
		next.ServeHTTP(w, r)
	})
}

// generateHTML assembles a number of HTML files in the "templates" directory
func generateHTML(w http.ResponseWriter, data interface{}, fn ...string) error {
	var files []string
	for _, file := range fn {
		files = append(files, fmt.Sprintf("templates/%s.html", file))
	}

	templates := template.Must(template.ParseFiles(files...))

	return templates.ExecuteTemplate(w, "layout", data)
}

func (app *App) session(w http.ResponseWriter, r *http.Request) (*Session, error) {
	cookie, err := r.Cookie("_cookie")
	if err != nil {
		return nil, err
	}

	var value string
	if err = app.cookieEncoder.Decode("_cookie", cookie.Value, &value); err != nil {
		return nil, err
	}

	return QuerySessionByUUID(app.db, value)
}

func (app *App) homeHandler(w http.ResponseWriter, r *http.Request) error {
	session, _ := app.session(w, r)
	if session == nil {
		return generateHTML(w, []string{}, "layout", "public.navbar", "index")
	} else {
		return generateHTML(w, []string{}, "layout", "private.navbar", "index")
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) error {
	return generateHTML(w, []string{}, "layout", "public.navbar", "login")
}

func (app *App) logoutHandler(w http.ResponseWriter, r *http.Request) error {
	session, _ := app.session(w, r)

	// Do not bother checking for error messages here, as the user is
	// logging out
	_ = DeleteSession(app.db, session.UUID)

	// We don't bother deleting the cookie: as long as it is invalid,
	// keeping it in requests is the same as not having it anymore.
	http.Redirect(w, r, "/", 302)
	return nil
}

func (app *App) authenticateHandler(w http.ResponseWriter, r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}

	user, err := QueryUserByEmail(app.db, r.PostFormValue("email"))
	if err != nil {
		return err
	}

	if user == nil {
		log.WithFields(log.Fields{
			"email": r.PostFormValue("email"),
		}).Info("User not found")

		http.Redirect(w, r, "/login", 302)
		return nil
	}

	_, goodPasswd, err := CheckUserPassword(app.db, user.Email, r.PostFormValue("password"))
	if err != nil {
		return err
	}
	if !goodPasswd {
		http.Redirect(w, r, "/login", 302)
		return nil
	}

	session, err := CreateSession(app.db, user)
	if err != nil {
		return err
	}

	// Encode the cookie to prevent tampering
	encoded, err := app.cookieEncoder.Encode("_cookie", session.UUID)
	if err != nil {
		return err
	}

	cookie := http.Cookie{
		Name:  "_cookie",
		Value: encoded,

		// true means no scripts, HTTP/HTTPS requests only are
		// allowed. This prevents cross-site scripting (XSS) attacks
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)
	http.Redirect(w, r, "/", 302)

	return nil
}

func (app *App) acquisitionListHandler(w http.ResponseWriter, r *http.Request) error {
	var acq []Acquisition
	if app == nil {
		panic("app cannot be nil")
	}
	if err := app.db.Find(&acq).Error; err != nil {
		return Error{err: err, msg: "Unable to query the database"}
	}

	data, err := json.Marshal(acq)
	if err != nil {
		return Error{err: err, msg: "Unable to encode the list of acquisitions"}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
	return nil
}

func (app *App) acquisitionHandler(w http.ResponseWriter, r *http.Request) error {
	if app == nil {
		panic("app cannot be nil")
	}

	vars := mux.Vars(r)
	var acq Acquisition
	if err := app.db.Where("acquisition_time = ?", vars["acq_id"]).First(&acq).Error; err != nil {
		return Error{err: err, msg: fmt.Sprintf("Unable to query the database for acquisition with ID %s",
			vars["acq_id"])}
	}

	data, err := json.Marshal(acq)
	if err != nil {
		return Error{err: err, msg: "Unable to encode the acquisition"}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
	return nil
}

func (app *App) rawListHandler(w http.ResponseWriter, r *http.Request) error {
	if app == nil {
		panic("app cannot be nil")
	}

	vars := mux.Vars(r)
	var rawFiles []RawDataFile
	if err := app.db.Joins("JOIN acquisitions ON raw_data_files.acquisition_id = acquisitions.id").
		Where("acquisitions.acquisition_time = ?", vars["acq_id"]).Find(&rawFiles).Error; err != nil {
		return Error{
			err: err,
			msg: fmt.Sprintf("Unable to query the database for raw files belonging to ID %s", vars["acq_id"]),
		}
	}

	data, err := json.Marshal(rawFiles)
	if err != nil {
		return Error{err: err, msg: "Unable to encode the list of raw files"}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
	return nil
}

func (app *App) rawFileHandler(w http.ResponseWriter, r *http.Request) error {
	if app == nil {
		panic("app cannot be nil")
	}

	vars := mux.Vars(r)
	asicNumber, _ := strconv.Atoi(vars["asic_num"])
	var rawFiles []RawDataFile
	if err := app.db.Joins("JOIN acquisitions ON raw_data_files.acquisition_id = acquisitions.id").
		Where("acquisitions.acquisition_time = ? AND asic_number = ?",
			vars["acq_id"], asicNumber).Find(&rawFiles).Error; err != nil {
		return Error{
			err: err,
			msg: fmt.Sprintf("Unable to query the database for raw file (ASIC %d) belonging to ID %s",
				asicNumber, vars["acq_id"],
			),
		}
	}

	fitsfile, err := os.Open(rawFiles[0].FileName)
	if err != nil {
		return Error{err: err, msg: "Unable to retrieve the FITS file"}
	}
	defer fitsfile.Close()

	if _, err := io.Copy(w, fitsfile); err != nil {
		return Error{err: err, msg: "Unable to send the FITS file"}
	}

	w.Header().Set("Content-Type", "application/fits")
	return nil
}

func (app *App) sumListHandler(w http.ResponseWriter, r *http.Request) error {
	if app == nil {
		panic("app cannot be nil")
	}

	vars := mux.Vars(r)
	var sumFiles []SumDataFile
	if err := app.db.Joins("JOIN acquisitions ON sum_data_files.acquisition_id = acquisitions.id").
		Where("acquisitions.acquisition_time = ?", vars["acq_id"]).Find(&sumFiles).Error; err != nil {
		return Error{err: err, msg: fmt.Sprintf("Unable to query the database for science files belonging to ID %s", vars["acq_id"])}
	}

	data, err := json.Marshal(sumFiles)
	if err != nil {
		return Error{err: err, msg: "Unable to encode the list of science files"}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
	return nil
}

func (app *App) sumFileHandler(w http.ResponseWriter, r *http.Request) error {
	if app == nil {
		panic("app cannot be nil")
	}

	vars := mux.Vars(r)
	asicNumber, _ := strconv.Atoi(vars["asic_num"])
	var sumFiles []SumDataFile
	if err := app.db.Joins("JOIN acquisitions ON sum_data_files.acquisition_id = acquisitions.id").
		Where("acquisitions.acquisition_time = ? AND asic_number = ?",
			vars["acq_id"], asicNumber).Find(&sumFiles).Error; err != nil {
		return Error{
			err: err,
			msg: fmt.Sprintf("Unable to query the database for science file (ASIC %d) belonging to ID %s",
				asicNumber, vars["acq_id"],
			),
		}
	}

	fitsfile, err := os.Open(sumFiles[0].FileName)
	if err != nil {
		return Error{err: err, msg: "Unable to retrieve the FITS file"}
	}
	defer fitsfile.Close()

	if _, err := io.Copy(w, fitsfile); err != nil {
		return Error{err: err, msg: "Unable to send the FITS file"}
	}

	w.Header().Set("Content-Type", "application/fits")
	return nil
}

func (app *App) asicHkHandler(w http.ResponseWriter, r *http.Request) error {
	if app == nil {
		panic("app cannot be nil")
	}

	vars := mux.Vars(r)
	var acq Acquisition
	if err := app.db.Where("acquisition_time = ?", vars["acq_id"]).First(&acq).Error; err != nil {
		return Error{err: err, msg: fmt.Sprintf("Unable to query the database for acquisition with ID %s",
			vars["acq_id"])}
	}

	fitsfile, err := os.Open(acq.AsicHkFileName)
	if err != nil {
		return Error{err: err, msg: fmt.Sprintf("Unable to retrieve the FITS file %q", acq.AsicHkFileName)}
	}
	defer fitsfile.Close()

	if _, err := io.Copy(w, fitsfile); err != nil {
		return Error{err: err, msg: "Unable to send the FITS file"}
	}

	w.Header().Set("Content-Type", "application/fits")
	return nil
}

func (app *App) externHkHandler(w http.ResponseWriter, r *http.Request) error {
	if app == nil {
		panic("app cannot be nil")
	}

	vars := mux.Vars(r)
	var acq Acquisition
	if err := app.db.Where("acquisition_time = ?", vars["acq_id"]).First(&acq).Error; err != nil {
		return Error{err: err, msg: fmt.Sprintf("Unable to query the database for acquisition with ID %s",
			vars["acq_id"])}
	}

	fitsfile, err := os.Open(acq.ExternHkFileName)
	if err != nil {
		return Error{err: err, msg: fmt.Sprintf("Unable to retrieve the FITS file %q", acq.ExternHkFileName)}
	}
	defer fitsfile.Close()

	if _, err := io.Copy(w, fitsfile); err != nil {
		return Error{err: err, msg: "Unable to send the FITS file"}
	}

	w.Header().Set("Content-Type", "application/fits")
	return nil
}

func (app *App) serve() {
	router := mux.NewRouter()

	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/",
		http.FileServer(http.Dir(app.config.StaticPath))))

	router.Use(logMiddleware)
	app.initRouter(router)

	address := fmt.Sprintf("%s:%d",
		app.config.ServerName,
		app.config.PortNumber)
	srv := &http.Server{
		Handler:      router,
		Addr:         address,
		WriteTimeout: time.Duration(app.config.WriteTimeout * int64(time.Second)),
		ReadTimeout:  time.Duration(app.config.ReadTimeout * int64(time.Second)),
	}

	log.Fatal(srv.ListenAndServe())
}

func (app *App) handleErrWrap(f func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := f(w, r)
		if err != nil {
			code := http.StatusInternalServerError
			msg := err.Error()
			if e, ok := err.(Error); ok {
				code = e.code
				msg = e.msg
			}
			http.Error(w, err.Error(), code)
			log.WithFields(log.Fields{
				"handler": r.URL.Path,
				"error":   msg,
			}).Error("error executing handler")
			return
		}
	}
}

func (app *App) initRouter(router *mux.Router) {
	router.HandleFunc("/", app.handleErrWrap(app.homeHandler))
	router.HandleFunc("/login", app.handleErrWrap(loginHandler))
	router.HandleFunc("/logout", app.handleErrWrap(app.logoutHandler))
	router.HandleFunc("/authenticate", app.handleErrWrap(app.authenticateHandler))
	router.HandleFunc("/api/v1/acquisitions", app.handleErrWrap(app.acquisitionListHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[0-9]+}", app.handleErrWrap(app.acquisitionHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[0-9]+}/rawdata", app.handleErrWrap(app.rawListHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[0-9]+}/rawdata/{asic_num:[0-9]+}", app.handleErrWrap(app.rawFileHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[0-9]+}/sumdata", app.handleErrWrap(app.sumListHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[0-9]+}/sumdata/{asic_num:[0-9]+}", app.handleErrWrap(app.sumFileHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[0-9]+}/asichk", app.handleErrWrap(app.asicHkHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[0-9]+}/externhk", app.handleErrWrap(app.externHkHandler)).Methods("GET")
}
