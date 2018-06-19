package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
)

func TestHandleAcquisitionList(t *testing.T) {
	router := mux.NewRouter()
	initRouter(router)

	writer := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/api/v1/acquisitions", nil)
	router.ServeHTTP(writer, request)

	if writer.Code != 200 {
		t.Errorf("Response code is %v", writer.Code)
	}

	var acq []Acquisition
	if err := json.Unmarshal(writer.Body.Bytes(), &acq); err != nil {
		t.Errorf("Unable to interpret JSON properly (%s): %s", err, string(writer.Body.Bytes()))
	}

	if len(acq) != 4 {
		t.Errorf("Wrong number of acquisitions returned by JSON API: %d", len(acq))
	}
}

func TestHandleAcquisition(t *testing.T) {
	router := mux.NewRouter()
	initRouter(router)

	writer := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/api/v1/acquisitions/1", nil)
	router.ServeHTTP(writer, request)

	if writer.Code != 200 {
		t.Errorf("Response code is %v", writer.Code)
	}

	var acq Acquisition
	if err := json.Unmarshal(writer.Body.Bytes(), &acq); err != nil {
		t.Errorf("Unable to interpret JSON properly (%s): %s", err, string(writer.Body.Bytes()))
	}

	if acq.ID != 1 {
		t.Errorf("Wrong ID (%d) in Acquisition object (%v)", acq.ID, acq)
	}
}

func TestHandleRawFiles(t *testing.T) {
	router := mux.NewRouter()
	initRouter(router)

	writer := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/api/v1/acquisitions/1/rawdata", nil)
	router.ServeHTTP(writer, request)

	if writer.Code != 200 {
		t.Errorf("Response code is %v", writer.Code)
	}

	var rawFiles []RawDataFile
	if err := json.Unmarshal(writer.Body.Bytes(), &rawFiles); err != nil {
		t.Errorf("Unable to interpret JSON properly (%s): %s", err, string(writer.Body.Bytes()))
	}

	if len(rawFiles) != 1 {
		t.Errorf("Wrong number of raw files: %d", len(rawFiles))
	}

	if filepath.Base(rawFiles[0].FileName) != "raw-asic1-2018.04.06.142047.fits" {
		t.Errorf("Wrong file name: %s", rawFiles[0].FileName)
	}
}
