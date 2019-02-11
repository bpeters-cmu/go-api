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

func TestCreateEndpointWithPreceding(t *testing.T) {
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

	connEvent2 := &models.ConnectionEvent{
		EventUUID: "85ad929a-db03-4bf4-9541-8f728fa12e43",
		Username:  "bob",
		IP:        "91.207.175.104",
		Timestamp: 1515774800,
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
}
