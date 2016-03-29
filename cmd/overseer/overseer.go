package main

import (
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
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
			"availability": float32(it.Bikes) / float32(it.Bikes+it.Attachs),
			"bikes":        it.Bikes,
			"free":         it.Attachs,
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

	data := overseer.NewWithCache(path.Join(cachePath, "stations.json"))
	if err := data.Update(); err != nil {
		log.Fatal(err)
	}
	data.UpdateAll()

	go startScrapper(time.Minute)

	router := gin.Default()
	router.GET("/stations", func(c *gin.Context) {
		c.JSON(http.StatusOK, data.List())
	})
	if err := router.Run(); err != nil {
		log.Fatal(err)
	}
}
