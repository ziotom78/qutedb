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
	"net/http"
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

func generateHTML(w http.ResponseWriter, data interface{}, fn ...string) {
	var files []string
	for _, file := range fn {
		files = append(files, fmt.Sprintf("template/%s.html", file))
	}

	templates := template.Must(template.ParseFiles(files...))
	templates.ExecuteTemplate(w, "layout", data)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	//_, err := session(w, r)
	if 1 != 0 {
		generateHTML(w, []string{}, "layout", "public.navbar", "index")
	} else {
		generateHTML(w, []string{}, "layout", "private.navbar", "index")
	}
}

func authenticateHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	//user, _ := data.UserByUsername
}

func acquisitionListHandler(w http.ResponseWriter, r *http.Request) {
	var acq []Acquisition
	if app == nil {
		panic("app cannot be nil")
	}
	if app.db.Find(&acq).Error != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unable to query the database: %s", app.db.Error)
		return
	}

	data, err := json.Marshal(acq)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unable to encode the list of acquisitions, reason: %s", err)
		return
	}

	w.Write(data)
}

func acquisitionHandler(w http.ResponseWriter, r *http.Request) {
	if app == nil {
		panic("app cannot be nil")
	}

	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])
	var acq Acquisition
	if app.db.Where("ID = ?", id).First(&acq).Error != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unable to query the database for acquisition with ID %d: %s", id, app.db.Error)
		return
	}

	data, err := json.Marshal(acq)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unable to encode the acquisition, reason: %s", err)
		return
	}

	w.Write(data)
}

func rawListHandler(w http.ResponseWriter, r *http.Request) {
	if app == nil {
		panic("app cannot be nil")
	}

	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])
	var rawFiles []RawDataFile
	if app.db.Where("acquisition_id = ?", id).Find(&rawFiles).Error != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unable to query the database for raw files belonging to ID %d: %s", id, app.db.Error)
		return
	}

	data, err := json.Marshal(rawFiles)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unable to encode the list of raw files, reason: %s", err)
		return
	}

	w.Write(data)
}

func initRouter(router *mux.Router) {
	router.HandleFunc("/", homeHandler)
	router.HandleFunc("/authenticate", authenticateHandler)
	router.HandleFunc("/api/v1/acquisitions", acquisitionListHandler).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{id:[0-9]+}", acquisitionHandler).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{id:[0-9]+}/rawdata", rawListHandler).Methods("GET")
}

func mainEventLoop(app *App) {
	router := mux.NewRouter()

	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/",
		http.FileServer(http.Dir(app.config.StaticPath))))

	router.Use(logMiddleware)
	initRouter(router)

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
