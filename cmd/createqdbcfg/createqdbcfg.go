package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/gorilla/securecookie"
	qdb "github.com/ziotom78/qutedb"
)

func main() {
	var hashlength = flag.Int("hashlength", 64,
		"Length (in bytes) of the cookie hash key")
	var blocklength = flag.Int("blocklength", 64,
		"Length (in bytes) of the cookie block key")
	var staticpath = flag.String("staticpath", "./static",
		"Path to the folder containing the static files to be served")
	var repositorypath = flag.String("repositorypath", "./",
		"Path to the folder containing the repository with the FITS files")
	var dbfile = flag.String("dbfile", "./db.sqlite3",
		"Full name (with path) to the SQLite3 database file")
	var servername = flag.String("servername", "127.0.0.1",
		"Name of the server")
	var portnum = flag.Int("port", 8080,
		"Port number for HTTP(s) communications")

	flag.Parse()

	rand.Seed(time.Now().UTC().UnixNano())

	conf := qdb.Configuration{
		DatabaseFile:   *dbfile,
		LogFormat:      "text",
		LogLevel:       "info",
		PortNumber:     *portnum,
		ServerName:     *servername,
		StaticPath:     *staticpath,
		RepositoryPath: *repositorypath,
		ReadTimeout:    15,
		WriteTimeout:   60,
		CookieHashKey:  securecookie.GenerateRandomKey(*hashlength),
		CookieBlockKey: securecookie.GenerateRandomKey(*blocklength),
	}

	json, err := json.MarshalIndent(conf, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(json))
}
