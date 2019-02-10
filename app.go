package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"superman-detector/models"
)

func CreateEventEndPoint(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var event models.ConnectionEvent
	var response models.GeoStatus
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if err := event.CreateConnection(); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	event.CalculateGeo()
	response.CurrentGeo = event.CurrentGeo
	access1, access2 := event.GetBeforeAfterIpAccess()
	response.CalculateResponse(access1, access2, &event)

	respondWithJson(w, http.StatusCreated, response)

}
func respondWithError(w http.ResponseWriter, code int, msg string) {
	respondWithJson(w, code, map[string]string{"error": msg})
}

func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func main() {
	models.InitDB("GeoLite2-City-Blocks-IPv4.db")
	r := mux.NewRouter()
	r.HandleFunc("/events", CreateEventEndPoint).Methods("POST")
	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatal(err)
	}
}