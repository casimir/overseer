package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/casimir/overseer"
	"github.com/gin-gonic/gin"
	client "github.com/influxdata/influxdb/client/v2"
)

const appName = "overseer"

var (
	cachePath    = "/var/lib/" + appName
	influxClient client.Client
)

var dataCache *overseer.StationList

func scrapData() {
	start := time.Now()
	data, cacheErr := overseer.NewWithCache(path.Join(cachePath, "stations.json"))
	if cacheErr != nil {
		log.Println("could not load cache file, it will be initiated on first fetching")
	}
	if err := data.Update(); err != nil {
		if cerr, ok := err.(*overseer.CacheError); ok {
			log.Print(cerr)
		} else {
			log.Printf("failed to update data: %s", err)
			return
		}
	}
	data.UpdateAll()
	dataCache = data
	log.Printf("scrapped data in %s", time.Since(start))

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: "overseer",
	})
	if err != nil {
		log.Printf("failed to save data: %s", err)
		return
	}
	now := time.Now()
	_, week := now.ISOWeek()
	dow := int(now.Weekday())
	for _, it := range data.List() {
		tags := map[string]string{
			"id":      strconv.Itoa(it.ID),
			"week":    strconv.Itoa(week),
			"weekday": strconv.Itoa(dow),
		}
		fields := map[string]interface{}{
			"availability": float32(it.Bikes) / float32(it.Bikes+it.Slots),
			"bikes":        it.Bikes,
			"free":         it.Slots,
			"status":       strconv.Itoa(it.Status),
		}
		pt, err := client.NewPoint("station", tags, fields, start)
		if err != nil {
			log.Printf("failed to save data: %s", err)
			continue
		}
		bp.AddPoint(pt)
	}
	if err := influxClient.Write(bp); err != nil {
		log.Printf("failed to save data: %s", err)
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

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type,Content-Length,Accept-Encoding,X-CSRF-Token,Authorization,accept,origin,Cache-Control,X-Requested-With")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET,OPTIONS,POST,PUT")
			c.AbortWithStatus(http.StatusNoContent)
		} else {
			c.Next()
		}
	}
}

func parseLocation(location string) (float64, float64, error) {
	// for now handle @lat,lng
	// later handle indexed string
	invalidLocErr := fmt.Errorf("invalid location: %q", location)
	if location == "" || location[0] != '@' {
		return 0, 0, invalidLocErr
	}
	parts := strings.Split(location[1:], ",")
	if len(parts) != 2 {
		return 0, 0, invalidLocErr
	}
	lat, latErr := strconv.ParseFloat(parts[0], 64)
	if latErr != nil {
		return 0, 0, latErr
	}
	lng, lngErr := strconv.ParseFloat(parts[1], 64)
	if lngErr != nil {
		return 0, 0, lngErr
	}
	return lat, lng, nil
}

var (
	fakeMode     = flag.Bool("fake", false, "don't scrap or connect to DB and serve fake data")
	fakeModeFile = flag.String("fakefile", "", "data file to use in fake mode (implies -fake)")
)

func main() {
	flag.Parse()

	if *fakeMode || *fakeModeFile != "" {
		fakeCachePath := *fakeModeFile
		if fakeCachePath == "" {
			gopath := os.Getenv("GOPATH")
			pkg := "github.com/casimir/overseer"
			fakeCachePath = path.Join(gopath, "src", pkg, "_testdata/stations.json")
		}
		log.Printf("using fake data from %s\n", fakeCachePath)
		cache, err := overseer.NewWithCache(fakeCachePath)
		if err != nil {
			log.Fatalf("could not load data: %s\n", err)
		}
		dataCache = cache
	} else {
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
	}

	router := gin.Default()
	router.Use(CORSMiddleware())
	router.GET("/stations", func(c *gin.Context) {
		c.JSON(http.StatusOK, dataCache.Filter(filters(c)...).List())
	})
	router.GET("/station/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		c.JSON(http.StatusOK, dataCache.Get(id))
	})
	router.GET("/near/:location", func(c *gin.Context) {
		location := c.Param("location")
		lat, lng, err := parseLocation(location)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		stations := overseer.NewGeolist(dataCache.Filter(filters(c)...), lat, lng)
		n := len(stations)
		if num, err := strconv.Atoi(c.Query("n")); err == nil {
			n = num
		}
		c.JSON(http.StatusOK, stations[:n])
	})
	router.GET("/now/:location", func(c *gin.Context) {
		location := c.Param("location")
		lat, lng, err := parseLocation(location)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		stations := overseer.NewGeolist(dataCache, lat, lng)
		c.JSON(http.StatusOK, overseer.NewNow(stations))
	})
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
