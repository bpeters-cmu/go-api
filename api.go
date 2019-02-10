package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func CreateEventEndPoint(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var event ConnectionEvent
	var response GeoStatus
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	db := InitDB("GeoLite2-City-Blocks-IPv4.db")
	if err := event.CreateConnection(db); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	event.CalculateGeo(db)
	response.CurrentGeo = event.CurrentGeo
	access1, access2 := event.GetBeforeAfterIpAccess(db)
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
	r := mux.NewRouter()
	r.HandleFunc("/events", CreateEventEndPoint).Methods("POST")
	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatal(err)
	}
}

func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)

	CreateTable(db)
	if err != nil {

	}
	if db == nil {
		panic("db nil")
	}
	return db
}

func CreateTable(db *sql.DB) {
	// create table if not exists
	sql_table := `
  DROP TABLE conn_events;
  DROP TABLE geo;
	CREATE TABLE IF NOT EXISTS conn_events(
		event_uuid TEXT NOT NULL PRIMARY KEY,
		username TEXT,
		ip TEXT,
		unix_timestamp INTEGER
	);
  CREATE TABLE IF NOT EXISTS geo(
    event_uuid TEXT NOT NULL PRIMARY KEY,
    lat REAL,
    lon REAL,
    radius INTEGER
  );
	`

	_, err := db.Exec(sql_table)
	if err != nil {
		fmt.Printf("error1")
	}
}
