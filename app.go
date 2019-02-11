package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
	"superman-detector/models"
)

// Handler for posting connection events
func EventHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var event models.ConnectionEvent
	// parse request body
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	// validate required fields
	validationErrors := event.Validate()
	if len(validationErrors) > 0 {
		respondWithJson(w, http.StatusBadRequest, validationErrors)
		return
	}
	// create connection event in DB
	if err := event.CreateConnection(); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// calculate geolocation data for connection event
	event.CalculateGeo()

	var response models.GeoStatus
	// set response geolocation data
	response.CurrentGeo = event.CurrentGeo
	// calculate preceding/subsequent IP access events
	access1, access2 := event.GetBeforeAfterIpAccess()
	// set remaining response data
	response.CalculateResponse(access1, access2, &event)

	respondWithJson(w, http.StatusCreated, response)

}

// This function is referenced from
// https://github.com/demo-apps/mux-postgres-rest/blob/master/app.go#L150
func respondWithError(w http.ResponseWriter, code int, msg string) {
	respondWithJson(w, code, map[string]string{"error": msg})
}

// This function is referenced from
// https://github.com/demo-apps/mux-postgres-rest/blob/master/app.go#L154
func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func main() {
	models.InitDB("GeoLite2-City-Blocks-IPv4.db")
	r := mux.NewRouter()
	r.HandleFunc("/events", EventHandler).Methods("POST")
	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatal(err)
	}
}
