package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/casimir/overseer"
	"github.com/casimir/xdg-go"
	"github.com/k0kubun/pp"
)

const baseURL = "http://overseer.casimir-lab.net"

type config struct {
	Location map[string]map[string]struct{ Lat, Lng float64 }
}

func init() {
	xdg.SetName("bike")
}

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

func main() {
	raw, _ := ioutil.ReadFile(xdg.ConfigPath("config.toml"))
	var cfg config
	if err := toml.Unmarshal(raw, &cfg); err != nil {
		pp.Println(err)
		os.Exit(1)
	}

	stations, err := overseer.New(true)
	if err != nil {
		log.Fatalf(err.Error())
	}

	if lat, lng, hasGeo := getPosition(&cfg); hasGeo {
		url := fmt.Sprintf("%s/near/%f/%f?n=3", baseURL, lat, lng)
		resp, err := http.Get(url)
		if err != nil {
			log.Fatal(err)
		}
		raw, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		var stations overseer.GeoStationList
		if err := json.Unmarshal(raw, &stations); err != nil {
			log.Fatal(err)
		}
		for _, it := range stations {
			printStationInfo(it.Station)
		}
	} else {
		forId(stations, 83)
	}
}
