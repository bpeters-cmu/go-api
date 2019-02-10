package models

import (
	"database/sql"
	log "github.com/sirupsen/logrus"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func InitDB(filepath string) {
	var err error
	db, err = sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatal(err)
	}
	if db == nil {
		log.Fatal("Could not establish DB conn")
	}
	log.Info("DB conn established")
	CreateTable()
}

func CreateTable() {
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
		log.Fatal("Error creating DB tables")
	}
	log.Info("Created tables conn_events and geo")
}
