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
	"html/template"
	"net/http"
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

func getTestListHandler(w http.ResponseWriter, r *http.Request) {
}

func mainEventLoop(app *App) {
	router := mux.NewRouter()

	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/",
		http.FileServer(http.Dir(app.config.StaticPath))))

	router.Use(logMiddleware)

	router.HandleFunc("/", homeHandler)
	router.HandleFunc("/authenticate", authenticateHandler)
	router.HandleFunc("/api/v1/tests", getTestListHandler).Methods("GET")

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
