package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/umahmood/haversine"
)

type ConnectionEvent struct {
	EventUUID  string `json:"event_uuid"`
	Username   string `json:"username"`
	IP         string `json:"ip_address"`
	Timestamp  int    `json:"unix_timestamp"`
	CurrentGeo *Geo   `json:"currentGeo"`
}

type Geo struct {
	EventUUID string  `json:"-"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	Radius    int     `json:"radius"`
}

type GeoStatus struct {
	CurrentGeo         *Geo      `json:"currentGeo"`
	TravelToCurrent    bool      `json:"travelToCurrentGeoSuspicious"`
	TravelFromCurrent  bool      `json:"travelFromCurrentGeoSuspicious"`
	PrecedingIpAccess  *IpAccess `json:"precedingIpAccess"`
	SubsequentIpAccess *IpAccess `json:"subsequentIpAccess"`
}

type IpAccess struct {
	IP        string  `json:"ip"`
	Speed     int     `json:"speed"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	Radius    int     `json:"radius"`
	Timestamp int     `json:"timestamp"`
}

func (c *ConnectionEvent) CreateConnection(db *sql.DB) error {

	statement := fmt.Sprintf("INSERT INTO conn_events(event_uuid, username, ip, unix_timestamp) VALUES('%s', '%s', '%s', %d)", c.EventUUID, c.Username, c.IP, c.Timestamp)
	_, err := db.Exec(statement)
	if err != nil {
		return err
	}
	return nil
}

func (c *ConnectionEvent) CalculateGeo(db *sql.DB) {
	ipArray := strings.Split(c.IP, ".")
	ipRange := strings.Join(ipArray[0:3], ".")
	geo := Geo{EventUUID: c.EventUUID}
	fmt.Println(ipRange)

	statement := `SELECT latitude, longitude, accuracy_radius FROM "GeoLite2-City-Blocks-IPv4" WHERE network LIKE '` + ipRange + `%'`
	fmt.Println(statement)
	err := db.QueryRow(statement).Scan(&geo.Latitude, &geo.Longitude, &geo.Radius)
	if err != nil {
		fmt.Printf("error2")
		fmt.Println(err)
	}
	c.CurrentGeo = &geo

}

func (access *IpAccess) CalculateSpeed(c *ConnectionEvent) {
	pointA := haversine.Coord{Lat: access.Latitude, Lon: access.Longitude}
	pointB := haversine.Coord{Lat: c.CurrentGeo.Latitude, Lon: c.CurrentGeo.Longitude}
	mi, _ := haversine.Distance(pointA, pointB)
	time := Abs(access.Timestamp-c.Timestamp) / 3600
	speed := int(mi) / time
	access.Speed = speed

}

func (c *ConnectionEvent) GetBeforeAfterIpAccess(db *sql.DB) (*IpAccess, *IpAccess) {

	before := IpAccess{}
	after := IpAccess{}
	before_statement := `
  SELECT ip, latitude, longitude, accuracy_radius, unix_timestamp
  FROM geo JOIN conn_events on geo.event_uuid = conn_events.event_uuid
  WHERE unix_timestamp < ` + strconv.Itoa(c.Timestamp) + `
  ORDER BY unix_timestamp
  LIMIT 1
  `
	err := db.QueryRow(before_statement).Scan(&before.IP, &before.Latitude, &before.Longitude, &before.Radius, &before.Timestamp)
	if err != nil {
		fmt.Printf("error2")
		fmt.Println(err)
	}

	after_statement := `
  SELECT ip, latitude, longitude, accuracy_radius, unix_timestamp
  FROM geo JOIN conn_events on geo.event_uuid = conn_events.event_uuid
  WHERE unix_timestamp > ` + strconv.Itoa(c.Timestamp) + `
  ORDER BY unix_timestamp
  LIMIT 1
  `

	err2 := db.QueryRow(after_statement).Scan(&after.IP, &after.Latitude, &after.Longitude, &after.Radius, &after.Timestamp)
	if err2 != nil {
		fmt.Printf("error2")
		fmt.Println(err2)
	}
	return &before, &after
}

func (g *Geo) InsertGeo(db *sql.DB) {
	statement := fmt.Sprintf("INSERT INTO geo(event_uuid, lat, lon, radius) VALUES('%s', '%f', '%f', %d)", g.EventUUID, g.Latitude, g.Longitude, g.Radius)
	_, err := db.Exec(statement)
	if err != nil {
		fmt.Println(err)
	}
}

func (access *IpAccess) IsEmpty() bool {
	if access.IP == "" {
		return true
	}
	return false
}
func (geoStatus GeoStatus) CalculateResponse(access1, access2 *IpAccess, event *ConnectionEvent) {
	if !access1.IsEmpty() {
		access1.CalculateSpeed(event)
		if access1.Speed > 500 {
			geoStatus.TravelToCurrent = true
		} else {
			geoStatus.TravelToCurrent = false
		}
		geoStatus.PrecedingIpAccess = access1
	}
	if !access2.IsEmpty() {
		access2.CalculateSpeed(event)
		if access1.Speed > 500 {
			geoStatus.TravelToCurrent = true
		} else {
			geoStatus.TravelToCurrent = false
		}
		geoStatus.SubsequentIpAccess = access2
	}

}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
