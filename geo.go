package overseer

import (
	"math"
	"sort"
)

const earthDiameter float64 = 12756200.0

func hav(theta float64) float64 {
	return math.Pow(math.Sin(theta/2), 2)
}

func distanceBetween(lat1, lng1, lat2, lng2 float64) float64 {
	lat1 *= math.Pi / 180
	lng1 *= math.Pi / 180
	lat2 *= math.Pi / 180
	lng2 *= math.Pi / 180
	h := hav(lat2-lat1) + math.Cos(lat1)*math.Cos(lat2)*hav(lng2-lng1)
	return earthDiameter * math.Asin(math.Sqrt(h))
}

type GeoStation struct {
	Station  Station
	Distance float64
}

type GeoStationList []GeoStation

func (gsl GeoStationList) Len() int           { return len(gsl) }
func (gsl GeoStationList) Less(i, j int) bool { return gsl[i].Distance < gsl[j].Distance }
func (gsl GeoStationList) Swap(i, j int)      { gsl[i], gsl[j] = gsl[j], gsl[i] }

func NewGeolist(ss *StationList, lat, lng float64) GeoStationList {
	var ret GeoStationList
	for _, it := range ss.list {
		d := distanceBetween(lat, lng, it.Lat, it.Lng)
		ret = append(ret, GeoStation{Station: it, Distance: d})
	}
	sort.Sort(ret)
	return ret
}
