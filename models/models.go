package models

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/umahmood/haversine"
	"net/url"
	"strconv"
	"strings"
)

// struct for parsing incoming requests
type ConnectionEvent struct {
	EventUUID  string `json:"event_uuid"`
	Username   string `json:"username"`
	IP         string `json:"ip_address"`
	Timestamp  int    `json:"unix_timestamp"`
	CurrentGeo *Geo   `json:"currentGeo"`
}

// struct to contain geolocation data for an event
type Geo struct {
	EventUUID string  `json:"-"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	Radius    int     `json:"radius"`
}

// struct to hold response data
type GeoStatus struct {
	CurrentGeo         *Geo      `json:"currentGeo"`
	TravelToCurrent    bool      `json:"travelToCurrentGeoSuspicious"`
	TravelFromCurrent  bool      `json:"travelFromCurrentGeoSuspicious"`
	PrecedingIpAccess  *IpAccess `json:"precedingIpAccess"`
	SubsequentIpAccess *IpAccess `json:"subsequentIpAccess"`
}

// struct for containing preceding/subsequent IP access data
type IpAccess struct {
	IP        string  `json:"ip"`
	Speed     int     `json:"speed"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	Radius    int     `json:"radius"`
	Timestamp int     `json:"timestamp"`
}

//Inserts ConnectionEvent into DB
func (c *ConnectionEvent) CreateConnection() error {

	statement := fmt.Sprintf("INSERT INTO conn_events(event_uuid, username, ip, unix_timestamp) VALUES('%s', '%s', '%s', %d)", c.EventUUID, c.Username, c.IP, c.Timestamp)
	_, err := db.Exec(statement)
	if err != nil {
		log.Warn("Error inserting connection event")
		return err
	}
	return nil
}

//Looks up geolocation info for a ConnectionEvent in the DB
func (c *ConnectionEvent) CalculateGeo() {
	ipArray := strings.Split(c.IP, ".")
	ipRange := strings.Join(ipArray[0:3], ".")
	geo := Geo{EventUUID: c.EventUUID}

	//SQL statement matches first 3 segments of IP address to CIDR block in DB
	statement := `
  SELECT latitude, longitude, accuracy_radius
  FROM "GeoLite2-City-Blocks-IPv4"
  WHERE network LIKE '` + ipRange + `%'
  `
	log.Info(fmt.Sprintf("Running query: %s", statement))
	err := db.QueryRow(statement).Scan(&geo.Latitude, &geo.Longitude, &geo.Radius)
	if err != nil {
		log.Warn(err)
	}
	c.CurrentGeo = &geo
	geo.InsertGeo()
}

//validates API request body is not missing fields
func (c *ConnectionEvent) Validate() url.Values {
	errs := url.Values{}

	if c.EventUUID == "" {
		errs.Add("event_uuid", "event_uuid is a required field")
	}
	if c.Username == "" {
		errs.Add("username", "username is a required field")
	}
	if c.IP == "" {
		errs.Add("ip", "ip is a required field")
	}
	if c.Timestamp == 0 {
		errs.Add("timestamp", "timestamp is a required field")
	}

	return errs
}

//Gets info on the preceding and subsequent ConnectionEvents
func (c *ConnectionEvent) GetBeforeAfterIpAccess() (*IpAccess, *IpAccess) {
	before := IpAccess{}
	after := IpAccess{}

	//sort by timestamp and grab the row before current timestamp,
	// excluding current connection
	before_statement := `
  SELECT ip, lat, lon, radius, unix_timestamp
  FROM geo JOIN conn_events on geo.event_uuid = conn_events.event_uuid
  WHERE username='` + c.Username + `'
    AND unix_timestamp <= ` + strconv.Itoa(c.Timestamp) + `
    AND conn_events.event_uuid != '` + c.EventUUID + `'
  ORDER BY unix_timestamp
  LIMIT 1
  `
	log.Info(fmt.Sprintf("Running query: %s", before_statement))

	err := db.QueryRow(before_statement).Scan(&before.IP, &before.Latitude, &before.Longitude, &before.Radius, &before.Timestamp)
	if err != nil {
		log.Warn(err)
	} else {
		log.Info("Found a preceding IP event")
	}

	//sort by timestamp and grab the row after current timestamp
	after_statement := `
  SELECT ip, lat, lon, radius, unix_timestamp
  FROM geo JOIN conn_events on geo.event_uuid = conn_events.event_uuid
  WHERE username='` + c.Username + `' AND unix_timestamp > ` + strconv.Itoa(c.Timestamp) + `
  ORDER BY unix_timestamp
  LIMIT 1
  `

	log.Info(fmt.Sprintf("Running query: %s", after_statement))

	err2 := db.QueryRow(after_statement).Scan(&after.IP, &after.Latitude, &after.Longitude, &after.Radius, &after.Timestamp)
	if err2 != nil {
		log.Warn(err2)
	} else {
		log.Info("Found a subsequent IP event")
	}

	return &before, &after
}

//inserts geo location info in DB
func (g *Geo) InsertGeo() {
	statement := fmt.Sprintf("INSERT INTO geo(event_uuid, lat, lon, radius) VALUES('%s', '%f', '%f', %d)", g.EventUUID, g.Latitude, g.Longitude, g.Radius)
	_, err := db.Exec(statement)
	if err != nil {
		log.Warn(err)
	}
}

//calculates speed using two sets of coordinates and the haversine formula to calculate distance in miles and determine speed in mph
func (access *IpAccess) CalculateSpeed(c *ConnectionEvent) {
	pointA := haversine.Coord{Lat: access.Latitude, Lon: access.Longitude}
	pointB := haversine.Coord{Lat: c.CurrentGeo.Latitude, Lon: c.CurrentGeo.Longitude}
	mi, _ := haversine.Distance(pointA, pointB)
	//divide difference in timestamps by 3600 seconds to determine hours
	time := float64(Abs(access.Timestamp-c.Timestamp)) / 3600
	speed := mi / time
	// truncates down to int type
	access.Speed = int(speed)
	log.Info(fmt.Sprintf("Calculated speed of: %d for user: %s", access.Speed, c.Username))
}

// determine if IpAccess struct is empty
func (access *IpAccess) IsEmpty() bool {
	if access.IP == "" {
		return true
	}
	return false
}

//glue function to calculate remain values required for response
func (geoStatus *GeoStatus) CalculateResponse(access1, access2 *IpAccess, event *ConnectionEvent) {
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

// returns absolute value of int
func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
