package main

import (
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/casimir/overseer"
	"github.com/gin-gonic/gin"
	"github.com/influxdata/influxdb/client/v2"
)

const appName = "overseer"

var (
	cachePath    = "/var/lib/" + appName
	configPath   = "/etc/" + appName
	influxClient client.Client
)

var dataCache *overseer.StationList

func scrapData() {
	start := time.Now()
	data := overseer.NewWithCache(path.Join(cachePath, "stations.json"))
	if err := data.Update(); err != nil {
		if cerr, ok := err.(*overseer.CacheError); ok {
			log.Print(cerr)
		} else {
			log.Printf("Failed to update data: %s", err)
			return
		}
	}
	data.UpdateAll()
	dataCache = data
	log.Printf("Scrapped data in %s", time.Since(start))

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: "overseer",
	})
	if err != nil {
		log.Printf("Failed to save data: %s", err)
		return
	}
	for _, it := range data.List() {
		tags := map[string]string{
			"id": strconv.Itoa(it.ID),
		}
		fields := map[string]interface{}{
			"availability": float32(it.Bikes) / float32(it.Bikes+it.Slots),
			"bikes":        it.Bikes,
			"free":         it.Slots,
			"status":       strconv.Itoa(it.Status),
		}
		pt, err := client.NewPoint("station", tags, fields, start)
		if err != nil {
			log.Printf("Failed to save data: %s", err)
			continue
		}
		bp.AddPoint(pt)
	}
	if err := influxClient.Write(bp); err != nil {
		log.Printf("Failed to save data: %s", err)
	}
}

func startScrapper(step time.Duration) {
	scrapData()
	for {
		<-time.After(step)
		go scrapData()
	}
}

func filters(c *gin.Context) (ret []overseer.StationFilter) {
	for _, it := range strings.Split(c.Query("filters"), ",") {
		switch it {
		case "bike":
			ret = append(ret, overseer.HasBike)
		case "slot":
			ret = append(ret, overseer.HasSlot)
		case "tickets":
			ret = append(ret, overseer.SellsTickets)
		}
	}
	return
}

func main() {
	var err error
	influxClient, err = client.NewHTTPClient(client.HTTPConfig{
		Addr:     "http://localhost:8086",
		Username: os.Getenv("OVERSEER_INFLUX_USR"),
		Password: os.Getenv("OVERSEER_INFLUX_PWD"),
	})
	if err != nil {
		log.Fatal(err)
	}

	go startScrapper(time.Minute)

	router := gin.Default()
	router.GET("/stations", func(c *gin.Context) {
		c.JSON(http.StatusOK, dataCache.Filter(filters(c)...).List())
	})
	router.GET("/station/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
		}
		c.JSON(http.StatusOK, dataCache.Get(id))
	})
	router.GET("/near/:lat/:lng", func(c *gin.Context) {
		lat, latErr := strconv.ParseFloat(c.Param("lat"), 64)
		lng, lngErr := strconv.ParseFloat(c.Param("lng"), 64)
		if latErr != nil || lngErr != nil {
			c.AbortWithStatus(http.StatusBadRequest)
		}
		stations := overseer.NewGeolist(dataCache.Filter(filters(c)...), lat, lng)
		n := len(stations)
		if num, err := strconv.Atoi(c.Query("n")); err == nil {
			n = num
		}
		c.JSON(http.StatusOK, stations[:n])
	})
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
