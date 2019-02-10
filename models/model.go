package models

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/umahmood/haversine"
	"strconv"
	"strings"
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

func (c *ConnectionEvent) CreateConnection() error {

	statement := fmt.Sprintf("INSERT INTO conn_events(event_uuid, username, ip, unix_timestamp) VALUES('%s', '%s', '%s', %d)", c.EventUUID, c.Username, c.IP, c.Timestamp)
	_, err := db.Exec(statement)
	if err != nil {
		log.Warn("Error inserting connection event")
		return err
	}
	return nil
}

func (c *ConnectionEvent) CalculateGeo() {
	ipArray := strings.Split(c.IP, ".")
	ipRange := strings.Join(ipArray[0:3], ".")
	geo := Geo{EventUUID: c.EventUUID}

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

func (access *IpAccess) CalculateSpeed(c *ConnectionEvent) {
	pointA := haversine.Coord{Lat: access.Latitude, Lon: access.Longitude}
	pointB := haversine.Coord{Lat: c.CurrentGeo.Latitude, Lon: c.CurrentGeo.Longitude}
	mi, _ := haversine.Distance(pointA, pointB)
	time := Abs(access.Timestamp-c.Timestamp) / 3600
	speed := int(mi) / time
	access.Speed = speed

	log.Info(fmt.Sprintf("Calculated speed of %d", speed))

}

func (c *ConnectionEvent) GetBeforeAfterIpAccess() (*IpAccess, *IpAccess) {
	before := IpAccess{}
	after := IpAccess{}
	before_statement := `
  SELECT ip, lat, lon, radius, unix_timestamp
  FROM geo JOIN conn_events on geo.event_uuid = conn_events.event_uuid
  WHERE unix_timestamp < ` + strconv.Itoa(c.Timestamp) + `
  ORDER BY unix_timestamp
  LIMIT 1
  `
	log.Info(fmt.Sprintf("Running query: %s", before_statement))

	err := db.QueryRow(before_statement).Scan(&before.IP, &before.Latitude, &before.Longitude, &before.Radius, &before.Timestamp)
	if err != nil {
		log.Warn(err)
	}

	after_statement := `
  SELECT ip, lat, lon, radius, unix_timestamp
  FROM geo JOIN conn_events on geo.event_uuid = conn_events.event_uuid
  WHERE unix_timestamp > ` + strconv.Itoa(c.Timestamp) + `
  ORDER BY unix_timestamp
  LIMIT 1
  `

	log.Info(fmt.Sprintf("Running query: %s", after_statement))

	err2 := db.QueryRow(after_statement).Scan(&after.IP, &after.Latitude, &after.Longitude, &after.Radius, &after.Timestamp)
	if err2 != nil {
		log.Warn(err2)
	}
	return &before, &after
}

func (g *Geo) InsertGeo() {
	statement := fmt.Sprintf("INSERT INTO geo(event_uuid, lat, lon, radius) VALUES('%s', '%f', '%f', %d)", g.EventUUID, g.Latitude, g.Longitude, g.Radius)
	_, err := db.Exec(statement)
	if err != nil {
		log.Warn(err)
	}
}

func (access *IpAccess) IsEmpty() bool {
	if access.IP == "" {
		return true
	}
	return false
}
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

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
