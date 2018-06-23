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

	w.Header().Set("Content-Type", "application/json")
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

	w.Header().Set("Content-Type", "application/json")
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

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func rawFileHandler(w http.ResponseWriter, r *http.Request) {
	if app == nil {
		panic("app cannot be nil")
	}

	vars := mux.Vars(r)
	acquisitionID, _ := strconv.Atoi(vars["acq_id"])
	asicNumber, _ := strconv.Atoi(vars["asic_num"])
	var rawFiles []RawDataFile
	if app.db.Where("acquisition_id = ? AND asic_number = ?",
		acquisitionID, asicNumber).Find(&rawFiles).Error != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unable to query the database for raw file (ASIC %d) belonging to ID %d: %s",
			asicNumber, acquisitionID, app.db.Error)
		return
	}

	fitsfile, err := os.Open(rawFiles[0].FileName)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unable to retrieve the FITS file: %s", err)
		return
	}
	defer fitsfile.Close()

	if _, err := io.Copy(w, fitsfile); err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unable to send the FITS file: %s", err)
	}

	w.Header().Set("Content-Type", "application/fits")
}

func sumListHandler(w http.ResponseWriter, r *http.Request) {
	if app == nil {
		panic("app cannot be nil")
	}

	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])
	var sumFiles []SumDataFile
	if app.db.Where("acquisition_id = ?", id).Find(&sumFiles).Error != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unable to query the database for science files belonging to ID %d: %s", id, app.db.Error)
		return
	}

	data, err := json.Marshal(sumFiles)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unable to encode the list of science files, reason: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func sumFileHandler(w http.ResponseWriter, r *http.Request) {
	if app == nil {
		panic("app cannot be nil")
	}

	vars := mux.Vars(r)
	acquisitionID, _ := strconv.Atoi(vars["acq_id"])
	asicNumber, _ := strconv.Atoi(vars["asic_num"])
	var sumFiles []SumDataFile
	if app.db.Where("acquisition_id = ? AND asic_number = ?",
		acquisitionID, asicNumber).Find(&sumFiles).Error != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unable to query the database for science file (ASIC %d) belonging to ID %d: %s",
			asicNumber, acquisitionID, app.db.Error)
		return
	}

	fitsfile, err := os.Open(sumFiles[0].FileName)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unable to retrieve the FITS file: %s", err)
		return
	}
	defer fitsfile.Close()

	if _, err := io.Copy(w, fitsfile); err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unable to send the FITS file: %s", err)
	}

	w.Header().Set("Content-Type", "application/fits")
}

func asicHkHandler(w http.ResponseWriter, r *http.Request) {
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

	fitsfile, err := os.Open(acq.AsicHkFileName)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unable to retrieve the FITS file \"%s\": %s", acq.AsicHkFileName, err)
		return
	}
	defer fitsfile.Close()

	if _, err := io.Copy(w, fitsfile); err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Unable to send the FITS file: %s", err)
	}

	w.Header().Set("Content-Type", "application/fits")
}

func initRouter(router *mux.Router) {
	router.HandleFunc("/", homeHandler)
	router.HandleFunc("/authenticate", authenticateHandler)
	router.HandleFunc("/api/v1/acquisitions", acquisitionListHandler).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{id:[0-9]+}", acquisitionHandler).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{id:[0-9]+}/rawdata", rawListHandler).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[0-9]+}/rawdata/{asic_num:[0-9]+}", rawFileHandler).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{id:[0-9]+}/sumdata", sumListHandler).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[0-9]+}/sumdata/{asic_num:[0-9]+}", sumFileHandler).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{id:[0-9]+}/asichk", asicHkHandler).Methods("GET")
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
