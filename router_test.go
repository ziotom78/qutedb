package qutedb

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/astrogo/fitsio"

	"github.com/gorilla/mux"
)

func TestHandleAcquisitionList(t *testing.T) {
	router := mux.NewRouter()
	app.initRouter(router)

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

	if len(acq) != 5 {
		t.Errorf("Wrong number of acquisitions returned by JSON API: %d", len(acq))
	}
}

func TestHandleAcquisition(t *testing.T) {
	router := mux.NewRouter()
	app.initRouter(router)

	writer := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/api/v1/acquisitions/2018-04-06T14:20:35", nil)
	request.Header.Set("Accept", "application/json")
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
	app.initRouter(router)

	writer := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/api/v1/acquisitions/2018-04-06T14:20:35/rawdata", nil)
	router.ServeHTTP(writer, request)

	if writer.Code != 200 {
		t.Errorf("Response code is %v", writer.Code)
	}

	var rawFiles []RawDataFile
	if err := json.Unmarshal(writer.Body.Bytes(), &rawFiles); err != nil {
		t.Errorf("Unable to interpret JSON properly (%s): %s", err, string(writer.Body.Bytes()))
	}

	if len(rawFiles) != 1 {
		t.Fatalf("Wrong number of raw files: %d", len(rawFiles))
	}

	if filepath.Base(rawFiles[0].FileName) != "raw-asic1-2018.04.06.142047.fits" {
		t.Errorf("Wrong file name: %s", rawFiles[0].FileName)
	}
}

func TestHandleRawFile(t *testing.T) {
	router := mux.NewRouter()
	app.initRouter(router)

	writer := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/api/v1/acquisitions/2018-04-06T14:20:35/rawdata/1", nil)
	router.ServeHTTP(writer, request)

	if writer.Code != 200 {
		t.Errorf("Response code is %v", writer.Code)
	}

	f, err := fitsio.Open(writer.Body)
	if err != nil {
		t.Errorf("Unable to decode FITS file: %s", err)
	}
	defer f.Close()

	if f.HDU(1).Header().Get("DATE").Value != "2018-04-06 14:20:35" {
		t.Errorf("Wrong value for DATE in FITS file: %s",
			f.HDU(1).Header().Get("DATE").Value)
	}
}

func TestHandleSumFiles(t *testing.T) {
	router := mux.NewRouter()
	app.initRouter(router)

	writer := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/api/v1/acquisitions/2018-04-06T14:20:35/sumdata", nil)
	router.ServeHTTP(writer, request)

	if writer.Code != 200 {
		t.Errorf("Response code is %v", writer.Code)
	}

	var sumFiles []SumDataFile
	if err := json.Unmarshal(writer.Body.Bytes(), &sumFiles); err != nil {
		t.Errorf("Unable to interpret JSON properly (%s): %s", err, string(writer.Body.Bytes()))
	}

	if len(sumFiles) != 1 {
		t.Fatalf("Wrong number of science files: %d", len(sumFiles))
	}

	if filepath.Base(sumFiles[0].FileName) != "science-asic1-2018.04.06.142047.fits" {
		t.Errorf("Wrong file name: %s", sumFiles[0].FileName)
	}
}

func TestHandleSumFile(t *testing.T) {
	router := mux.NewRouter()
	app.initRouter(router)

	writer := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/api/v1/acquisitions/2018-04-06T14:20:35/sumdata/1", nil)
	router.ServeHTTP(writer, request)

	if writer.Code != 200 {
		t.Errorf("Response code is %v", writer.Code)
	}

	f, err := fitsio.Open(writer.Body)
	if err != nil {
		t.Errorf("Unable to decode FITS file: %s", err)
	}
	defer f.Close()

	if f.HDU(1).Header().Get("DATE").Value != "2018-04-06 14:20:35" {
		t.Errorf("Wrong value for DATE in FITS file: %s",
			f.HDU(1).Header().Get("DATE").Value)
	}
}

func runTestOnHkFile(t *testing.T, url string, expectedDate string) {
	router := mux.NewRouter()
	app.initRouter(router)

	writer := httptest.NewRecorder()

	request, err := http.NewRequest("GET", url+"_doesnotexist", nil)
	if err != nil {
		t.Errorf("Unable to create test request: %v", err)
	}
	router.ServeHTTP(writer, request)
	if writer.Result().StatusCode != 404 {
		t.Errorf("Response code for non-existing URL is %v instead of 404", writer.Code)
	}

	request, err = http.NewRequest("GET", url, nil)
	if err != nil {
		t.Errorf("Unable to create test request: %v", err)
	}

	writer = httptest.NewRecorder()
	router.ServeHTTP(writer, request)

	if writer.Result().StatusCode != 200 {
		t.Errorf("Response code is %v instead of 200 for URL %v", writer.Code, url)
	} else {
		f, err := fitsio.Open(writer.Body)
		if err != nil {
			t.Errorf("Unable to decode FITS file: %s", err)
		}
		defer f.Close()

		if f.HDU(1).Header().Get("DATE").Value != expectedDate {
			t.Errorf("Wrong value for DATE in FITS file: %s",
				f.HDU(1).Header().Get("DATE").Value)
		}
	}
}

func TestAsicHkFile(t *testing.T) {
	runTestOnHkFile(t, "/api/v1/acquisitions/2018-04-06T14:20:35/asichk", "2018-04-06 14:20:35")
}

func TestInternHkFile(t *testing.T) {
	runTestOnHkFile(t, "/api/v1/acquisitions/2019-05-07T18:11:29/internhk", "2019-05-07 18:11:29")
}

func TestExternHkFile(t *testing.T) {
	runTestOnHkFile(t, "/api/v1/acquisitions/2019-05-07T18:11:29/externhk", "2019-05-07 18:11:29")
}
