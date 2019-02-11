package main

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"superman-detector/models"
	"testing"
)

func Router() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/events", EventHandler).Methods("POST")
	return router
}

//TODO: add more tests, create fixtures for repeated actions, break tests up into smaller units

func TestCreateEndpoint(t *testing.T) {
	connEvent := &models.ConnectionEvent{
		EventUUID: "85ad929a-db03-4bf4-9541-8f728fa12e42",
		Username:  "bob",
		IP:        "206.81.252.6",
		Timestamp: 1514764800,
	}
	requestBody, _ := json.Marshal(connEvent)
	request, _ := http.NewRequest("POST", "/events", bytes.NewBuffer(requestBody))
	response := httptest.NewRecorder()
	models.InitDB("GeoLite2-City-Blocks-IPv4.db")
	Router().ServeHTTP(response, request)
	var geoStatus models.GeoStatus
	// parse request body
	json.NewDecoder(response.Body).Decode(&geoStatus)
	assert.Equal(t, 201, response.Code, "Response code should be 201")
	assert.Equal(t, 39.2548, geoStatus.CurrentGeo.Latitude, "Latitude should be 39.2548")
}

func TestCreateEndpointWithPrecedingSubsequent(t *testing.T) {
	connEvent := &models.ConnectionEvent{
		EventUUID: "85ad929a-db03-4bf4-9541-8f728fa12e42",
		Username:  "bob",
		IP:        "206.81.252.6",
		Timestamp: 1514764800,
	}
	requestBody, _ := json.Marshal(connEvent)
	request, _ := http.NewRequest("POST", "/events", bytes.NewBuffer(requestBody))
	response := httptest.NewRecorder()
	models.InitDB("GeoLite2-City-Blocks-IPv4.db")
	Router().ServeHTTP(response, request)
	assert.Equal(t, 201, response.Code, "Response code should be 201")
	// create event with preceding IP access
	connEvent2 := &models.ConnectionEvent{
		EventUUID: "85ad929a-db03-4bf4-9541-8f728fa12e43",
		Username:  "bob",
		IP:        "91.207.175.104",
		Timestamp: 1515004800,
	}
	requestBody2, _ := json.Marshal(connEvent2)
	request2, _ := http.NewRequest("POST", "/events", bytes.NewBuffer(requestBody2))
	response2 := httptest.NewRecorder()
	Router().ServeHTTP(response2, request2)
	var geoStatus models.GeoStatus
	// parse request body
	json.NewDecoder(response2.Body).Decode(&geoStatus)

	assert.Equal(t, 201, response2.Code, "Response code should be 201")
	assert.NotEmpty(t, geoStatus.PrecedingIpAccess, "Preceding IP access shouldn't be empty")
	// create event with subsequent and preceding IP access
	connEvent3 := &models.ConnectionEvent{
		EventUUID: "85ad929a-db03-4bf4-9541-8f728fa12e44",
		Username:  "bob",
		IP:        "223.255.255.5",
		Timestamp: 1514994800,
	}
	requestBody3, _ := json.Marshal(connEvent3)
	request3, _ := http.NewRequest("POST", "/events", bytes.NewBuffer(requestBody3))
	response3 := httptest.NewRecorder()
	Router().ServeHTTP(response3, request3)
	var geoStatus2 models.GeoStatus
	// parse request body
	json.NewDecoder(response3.Body).Decode(&geoStatus2)

	assert.Equal(t, 201, response3.Code, "Response code should be 201")
	assert.NotEmpty(t, geoStatus2.PrecedingIpAccess, "Preceding IP access shouldn't be empty")
	assert.NotEmpty(t, geoStatus2.SubsequentIpAccess, "Subsequent IP access shouldn't be empty")
	assert.Equal(t, true, geoStatus2.TravelFromCurrent, "Travel from current should be suspicious")
	assert.Equal(t, false, geoStatus2.TravelToCurrent, "Travel to current should not be suspicious")

}
