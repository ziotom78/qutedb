package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleAcquisitionList(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/acquisitions", getAcquisitionListHandler)

	writer := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "/api/v1/acquisitions", nil)
	mux.ServeHTTP(writer, request)

	if writer.Code != 200 {
		t.Errorf("Response code is %v", writer.Code)
	}

	var acq []Acquisition
	json.Unmarshal(writer.Body.Bytes(), &acq)
	if len(acq) != 4 {
		t.Errorf("Wrong number of acquisitions returned by JSON API: %d", len(acq))
	}
}
