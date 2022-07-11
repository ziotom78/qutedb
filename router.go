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
	"archive/zip"
	"compress/flate"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
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

// HomeData contains the data passed to the "index.html" template
type HomeData struct {
	User            User
	AcquisitionList []Acquisition
}

func (app *App) homeHandler(w http.ResponseWriter, r *http.Request) error {
	session, _ := app.session(w, r)
	if session == nil {
		return generateHTML(w, nil, "layout", "public.navbar", "index")
	}

	user, err := QueryUserByID(app.db, session.UserID)
	if err != nil {
		return Error{
			err:  err,
			msg:  fmt.Sprintf("Unable to find user with ID %d", session.UserID),
			code: http.StatusInternalServerError,
		}
	}

	var acqList []Acquisition
	if err := app.db.Order("acquisition_time desc").Find(&acqList).Error; err != nil {
		return Error{
			err:  err,
			msg:  "Unable to retrieve list of acquisitions",
			code: http.StatusInternalServerError,
		}
	}
	log.WithFields(log.Fields{
		"num_of_acquisitions": len(acqList),
	}).Info("List of acquisitions going to be sent to index.html")

	return generateHTML(w, HomeData{
		User:            *user,
		AcquisitionList: acqList,
	}, "layout", "private.navbar", "index")
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
	if err := app.db.Order("acquisition_time desc").Find(&acq).Error; err != nil {
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

	log.WithFields(log.Fields{
		"accept": r.Header.Get("Accept"),
	}).Info("acquisitionHandler")

	vars := mux.Vars(r)
	acq, err := QueryAcquisition(app.db, vars["acq_id"])
	if err != nil {
		return Error{
			err: err,
			msg: fmt.Sprintf("Unable to query the database for acquisition with ID %s",
				vars["acq_id"]),
		}
	}

	// If the requester wants an HTML page, satisfy it!
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		return generateHTML(w, acq, "layout", "private.navbar", "acquisition")
	}

	// Otherwise, just return a JSON record
	data, err := json.Marshal(acq)
	if err != nil {
		return Error{err: err, msg: "Unable to encode the acquisition"}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)

	return nil
}

func addFileToArchive(nameInArchive string, filename string, comment string, ziparchive *zip.Writer) error {
	fileInfo, err := os.Lstat(filename)
	if err != nil {
		return err
	}

	fileHeader, err := zip.FileInfoHeader(fileInfo)
	if err != nil {
		return err
	}
	fileHeader.Method = zip.Deflate
	fileHeader.Name = nameInArchive
	fileHeader.Comment = comment

	f, err := ziparchive.CreateHeader(fileHeader)
	if err != nil {
		return Error{
			err: err,
			msg: fmt.Sprintf("Unable to add file \"%s\" to ZIP archive", nameInArchive),
		}
	}

	datafile, err := os.Open(filename)
	if err != nil {
		return Error{
			err: err,
			msg: fmt.Sprintf("Unable to retrieve FITS file \"%s\"", filename),
		}
	}
	defer datafile.Close()

	if _, err := io.Copy(f, datafile); err != nil {
		return Error{
			err: err,
			msg: fmt.Sprintf("Unable to compress file \"%s\"", filename),
		}
	}

	return nil
}

func addFilesToArchive(filelist []string, dirname string, comment string, ziparchive *zip.Writer) error {
	for _, filename := range filelist {
		if err := addFileToArchive(
			dirname+"/"+path.Base(filename),
			filename,
			comment,
			ziparchive,
		); err != nil {
			return err
		}
	}

	return nil
}

func (app *App) acquisitionBundleHandler(w http.ResponseWriter, r *http.Request) error {
	if app == nil {
		panic("app cannot be nil")
	}

	log.WithFields(log.Fields{
		"accept": r.Header.Get("Accept"),
	}).Info("acquisitionBundleHandler")

	vars := mux.Vars(r)
	acq, err := QueryAcquisition(app.db, vars["acq_id"])
	if err != nil {
		return Error{
			err: err,
			msg: fmt.Sprintf("Unable to query the database for acquisition with ID %s",
				vars["acq_id"]),
		}
	}

	zipFile, err := os.CreateTemp("", "qutedb")
	if err != nil {
		return Error{
			err: err,
			msg: fmt.Sprintf("Unable to create a temporary Zip file for acquisition with ID %s",
				vars["acq_id"]),
		}
	}

	defer os.Remove(zipFile.Name())

	ziparchive := zip.NewWriter(zipFile)

	// We strive for speed here, so we use the lowest possible compression
	// level. This usually achieves good performance nevertheless, so it is not a
	// big loss
	ziparchive.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, flate.BestSpeed)
	})

	// Create the directory structure within the ZIP file
	if _, err := ziparchive.Create("Hks/"); err != nil {
		return Error{err: err, msg: "Unable to create directory structure in ZIP file"}
	}

	if _, err := ziparchive.Create("Raws/"); err != nil {
		return Error{err: err, msg: "Unable to create directory structure in ZIP file"}
	}

	if _, err := ziparchive.Create("Sums/"); err != nil {
		return Error{err: err, msg: "Unable to create directory structure in ZIP file"}
	}

	filelist := make([]string, len(acq.RawFiles))
	for i := 0; i < len(acq.RawFiles); i++ {
		filelist[i] = acq.RawFiles[i].FileName
	}
	if err := addFilesToArchive(filelist, "Raws", "FITS file containing raw ASIC data", ziparchive); err != nil {
		return err
	}

	filelist = make([]string, len(acq.SumFiles))
	for i := 0; i < len(acq.SumFiles); i++ {
		filelist[i] = acq.SumFiles[i].FileName
	}
	if err := addFilesToArchive(filelist, "Sums", "FITS file containing scientific ASIC data", ziparchive); err != nil {
		return err
	}

	if acq.AsicHkFileName != "" {
		if err := addFileToArchive("Hks/"+path.Base(acq.AsicHkFileName), acq.AsicHkFileName,
			"FITS file containing ASIC configuration", ziparchive); err != nil {
			return err
		}
	}

	if acq.InternHkFileName != "" {
		if err := addFileToArchive("Hks/"+path.Base(acq.InternHkFileName), acq.InternHkFileName,
			"FITS file containing internal housekeeping data", ziparchive); err != nil {
			return err
		}
	}

	if acq.ExternHkFileName != "" {
		if err := addFileToArchive("Hks/"+path.Base(acq.ExternHkFileName), acq.ExternHkFileName,
			"FITS file containing external housekeeping data", ziparchive); err != nil {
			return err
		}
	}

	if acq.MmrHkFileName != "" {
		if err := addFileToArchive("Hks/"+path.Base(acq.MmrHkFileName), acq.MmrHkFileName,
			"FITS file containing MMR housekeeping data", ziparchive); err != nil {
			return err
		}
	}

	if acq.MgcHkFileName != "" {
		if err := addFileToArchive("Hks/"+path.Base(acq.MgcHkFileName), acq.MgcHkFileName,
			"FITS file containing MGC housekeeping data", ziparchive); err != nil {
			return err
		}
	}

	if acq.CalDataFileName != "" {
		if err := addFileToArchive("Hks/"+path.Base(acq.CalDataFileName), acq.CalDataFileName,
			"FITS file containing calibrator data", ziparchive); err != nil {
			return err
		}
	}

	if acq.CalConfFileName != "" {
		if err := addFileToArchive("Hks/"+path.Base(acq.CalConfFileName), acq.CalConfFileName,
			"FITS file containing the configuration of the calibrator", ziparchive); err != nil {
			return err
		}
	}

	if err := ziparchive.Close(); err != nil {
		return err
	}

	zipFileName := zipFile.Name()
	zipFile.Close()

	http.ServeFile(w, r, zipFileName)
	return nil
}

func (app *App) rawListHandler(w http.ResponseWriter, r *http.Request) error {
	if app == nil {
		panic("app cannot be nil")
	}

	vars := mux.Vars(r)
	var rawFiles []RawDataFile
	if err := app.db.
		Joins("JOIN acquisitions ON raw_data_files.acquisition_id = acquisitions.id").
		Where("acquisitions.acquisition_time = ?", vars["acq_id"]).
		Find(&rawFiles).Error; err != nil {
		return Error{
			err: err,
			msg: fmt.Sprintf("Unable to query for raw files belonging to ID %s",
				vars["acq_id"]),
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
	if err := app.db.
		Joins("JOIN acquisitions ON raw_data_files.acquisition_id = acquisitions.id").
		Where("acquisitions.acquisition_time = ? AND asic_number = ?",
			vars["acq_id"], asicNumber).
		Find(&rawFiles).Error; err != nil {
		return Error{
			err: err,
			msg: fmt.Sprintf("Unable to query for raw file (ASIC %d) belonging to ID %s",
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
	if err := app.db.
		Joins("JOIN acquisitions ON sum_data_files.acquisition_id = acquisitions.id").
		Where("acquisitions.acquisition_time = ?", vars["acq_id"]).
		Find(&sumFiles).Error; err != nil {
		return Error{
			err: err,
			msg: fmt.Sprintf("Unable to query for science files belonging to ID %s",
				vars["acq_id"])}
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
	if err := app.db.
		Joins("JOIN acquisitions ON sum_data_files.acquisition_id = acquisitions.id").
		Where("acquisitions.acquisition_time = ? AND asic_number = ?",
			vars["acq_id"], asicNumber).
		Find(&sumFiles).Error; err != nil {
		return Error{
			err: err,
			msg: fmt.Sprintf("Unable to query for science file (ASIC %d) belonging to ID %s",
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

func (app *App) genericHkHandler(w http.ResponseWriter, r *http.Request, getFileName func(*Acquisition) string) error {
	if app == nil {
		panic("app cannot be nil")
	}

	vars := mux.Vars(r)
	log.WithFields(log.Fields{
		"acq_id": vars["acq_id"],
	}).Debug("REST request for a HK file")

	var acq Acquisition
	if err := app.db.
		Where("acquisition_time = ?", vars["acq_id"]).
		First(&acq).Error; err != nil {
		return Error{
			err: err,
			msg: fmt.Sprintf("Unable to query for acquisition with ID %s",
				vars["acq_id"]),
		}
	}

	fileName := getFileName(&acq)
	if fileName == "" {
		return Error{err: nil, msg: "File not present in the acquisition"}
	}

	log.WithFields(log.Fields{
		"filename": fileName,
		"url":      r.URL.String(),
	}).Info("Going to copy a FITS file over a HTTP connection")

	fitsfile, err := os.Open(fileName)
	if err != nil {
		return Error{
			err: err,
			msg: fmt.Sprintf("Unable to retrieve the FITS file %q",
				fileName),
		}
	}
	defer fitsfile.Close()

	if _, err := io.Copy(w, fitsfile); err != nil {
		return Error{
			err: err,
			msg: "Unable to send the FITS file",
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/fits")
	return nil
}

func (app *App) asicHkHandler(w http.ResponseWriter, r *http.Request) error {
	return app.genericHkHandler(w, r, func(acq *Acquisition) string {
		return acq.AsicHkFileName
	})
}

func (app *App) internHkHandler(w http.ResponseWriter, r *http.Request) error {
	return app.genericHkHandler(w, r, func(acq *Acquisition) string {
		return acq.InternHkFileName
	})
}

func (app *App) externHkHandler(w http.ResponseWriter, r *http.Request) error {
	return app.genericHkHandler(w, r, func(acq *Acquisition) string {
		return acq.ExternHkFileName
	})
}

func (app *App) mmrHkHandler(w http.ResponseWriter, r *http.Request) error {
	return app.genericHkHandler(w, r, func(acq *Acquisition) string {
		return acq.MmrHkFileName
	})
}

func (app *App) mgcHkHandler(w http.ResponseWriter, r *http.Request) error {
	return app.genericHkHandler(w, r, func(acq *Acquisition) string {
		return acq.MgcHkFileName
	})
}

func (app *App) calDataHkHandler(w http.ResponseWriter, r *http.Request) error {
	return app.genericHkHandler(w, r, func(acq *Acquisition) string {
		return acq.CalDataFileName
	})
}

func (app *App) calConfHkHandler(w http.ResponseWriter, r *http.Request) error {
	return app.genericHkHandler(w, r, func(acq *Acquisition) string {
		return acq.CalConfFileName
	})
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

func (app *App) handleErrWrap(f func(w http.ResponseWriter,
	r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := f(w, r)
		if err != nil {
			code := http.StatusInternalServerError
			msg := err.Error()
			if e, ok := err.(Error); ok {
				code = e.code
				msg = e.msg
			}
			if code == 0 {
				code = http.StatusInternalServerError
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

const (
	// Page is allowed to all authenticated users
	authNormal = iota

	// Page is available only to users with administrative privileges
	authAdmin
)

// forceAuth is a middleware that ensures that a page is accessed only
// by users with specified privileges. The value of authLevel can be
// either authNormal (all authenticated users can access the resource)
// or authAdmin (only superusers can access the resource).
func (app *App) forceAuth(f func(w http.ResponseWriter,
	r *http.Request), authLevel int) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := app.session(w, r)
		if session == nil {
			http.Redirect(w, r, "/", 401)
		} else {
			user, err := QueryUserByID(app.db, session.UserID)
			if user == nil || err != nil || (authLevel == authAdmin && !user.Superuser) {
				log.Error("This session doesn't have a valid user")
				http.Redirect(w, r, "/", 404)
			} else {
				log.WithFields(log.Fields{
					"user_email": user.Email,
				}).Info("granting access to protected page")

				f(w, r)
			}
		}
	}
}

func (app *App) initRouter(router *mux.Router) {
	router.HandleFunc("/", app.handleErrWrap(app.homeHandler))
	router.HandleFunc("/login", app.handleErrWrap(loginHandler))
	router.HandleFunc("/logout", app.handleErrWrap(app.logoutHandler))
	router.HandleFunc("/authenticate", app.handleErrWrap(app.authenticateHandler))
	router.HandleFunc("/usermod",
		app.forceAuth(app.handleErrWrap(app.modifyUserHandler), authNormal))
	router.HandleFunc("/changepassword",
		app.forceAuth(app.handleErrWrap(app.changeUserPassword), authNormal))
	router.HandleFunc("/userlist",
		app.forceAuth(app.handleErrWrap(app.userListHandler), authAdmin))
	router.HandleFunc("/createuser",
		app.forceAuth(app.handleErrWrap(app.createUserHandler), authAdmin))
	router.HandleFunc("/createuser/new",
		app.forceAuth(app.handleErrWrap(app.createUser), authAdmin))

	router.HandleFunc("/api/v1/acquisitions",
		app.handleErrWrap(app.acquisitionListHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[-:T0-9]+}",
		app.handleErrWrap(app.acquisitionHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[-:T0-9]+}/archive",
		app.handleErrWrap(app.acquisitionBundleHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[-:T0-9]+}/rawdata",
		app.handleErrWrap(app.rawListHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[-:T0-9]+}/rawdata/{asic_num:[0-9]+}",
		app.handleErrWrap(app.rawFileHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[-:T0-9]+}/sumdata",
		app.handleErrWrap(app.sumListHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[-:T0-9]+}/sumdata/{asic_num:[0-9]+}",
		app.handleErrWrap(app.sumFileHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[-:T0-9]+}/asichk",
		app.handleErrWrap(app.asicHkHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[-:T0-9]+}/internhk",
		app.handleErrWrap(app.internHkHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[-:T0-9]+}/externhk",
		app.handleErrWrap(app.externHkHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[-:T0-9]+}/mmrhk",
		app.handleErrWrap(app.mmrHkHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[-:T0-9]+}/mgchk",
		app.handleErrWrap(app.mgcHkHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[-:T0-9]+}/calconf",
		app.handleErrWrap(app.calConfHkHandler)).Methods("GET")
	router.HandleFunc("/api/v1/acquisitions/{acq_id:[-:T0-9]+}/caldata",
		app.handleErrWrap(app.calDataHkHandler)).Methods("GET")
}
