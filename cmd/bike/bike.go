package main

import (
	"fmt"
	"log"

	"github.com/casimir/overseer"
)

func printStationInfo(station overseer.Station) {
	if station.Bikes == 0 {
		fmt.Printf("No bike... (%s)\n", station.Name)
	} else if station.Bikes == 1 {
		fmt.Printf("Only one bike! (%s)\n", station.Name)
	} else {
		fmt.Printf("%d bikes! (%s)\n", station.Bikes, station.Name)
	}
}

func forId(stations *overseer.StationList, id int) {
	if err := stations.UpdateStation(id); err != nil {
		log.Fatalf(err.Error())
	}
	printStationInfo(stations.Get(id))
}

func forPostion(stations *overseer.StationList, lat, lng float64, n int) {
	glist := overseer.NewGeolist(stations, lat, lng)
	for _, it := range glist[:n] {
		printStationInfo(it.Station)
	}
}

func main() {
	stations, err := overseer.New(true)
	if err != nil {
		log.Fatalf(err.Error())
	}

	if lat, lng, hasGeo := getPosition(); hasGeo {
		if errs := stations.UpdateAll(); len(errs) > 0 {
			log.Fatal(errs[0])
		}
		forPostion(stations, lat, lng, 3)
	} else {
		forId(stations, 83)
	}
}
