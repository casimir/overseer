package overseer

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
)

const (
	urlStations   = "http://www.vlille.fr/stations/xml-stations.aspx"
	urlTplStation = "http://www.vlille.fr/stations/xml-station.aspx?borne=%d"
)

func fetchURL(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return []byte{}, err
	}
	defer res.Body.Close()
	raw, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return []byte{}, err
	}
	raw = bytes.Replace(raw, []byte("utf-16"), []byte("utf-8"), -1)
	return raw, err
}

type (
	Station struct {
		ID          int
		Name        string
		Lat         float64
		Lng         float64
		Adress      string
		Status      int
		Bikes       int
		Slots       int
		SellTickets bool
	}
)

func (s Station) String() string {
	return fmt.Sprintf("%s (%d/%d)", s.Name, s.Bikes, s.Bikes+s.Slots)
}

type StationSlice []Station

func (ss StationSlice) Len() int           { return len(ss) }
func (ss StationSlice) Less(i, j int) bool { return ss[i].ID < ss[j].ID }
func (ss StationSlice) Swap(i, j int)      { ss[i], ss[j] = ss[j], ss[i] }

type StationList struct {
	cachePath string
	list      map[int]Station
}

func initStations() (*StationList, error) {
	ret := &StationList{list: make(map[int]Station)}
	rawStations, err := fetchURL(urlStations)
	if err != nil {
		return ret, err
	}
	err = ret.updateStations(rawStations)
	return ret, err
}

func New(init bool) (*StationList, error) {
	if !init {
		return &StationList{list: make(map[int]Station)}, nil
	}
	return initStations()
}

func NewWithCache(cachePath string) *StationList {
	return &StationList{
		cachePath: cachePath,
		list:      make(map[int]Station),
	}
}

func (sl StationList) Get(id int) Station {
	return sl.list[id]
}

func (sl StationList) List() StationSlice {
	var ret StationSlice
	for _, it := range sl.list {
		ret = append(ret, it)
	}
	sort.Sort(ret)
	return ret
}

func (sl *StationList) Update() error {
	rawStations, err := fetchURL(urlStations)
	if err != nil {
		return err
	}
	if err := sl.updateStations(rawStations); err != nil {
		return err
	}
	return sl.SaveCache()
}

func (sl *StationList) updateStations(data []byte) error {
	var tmp struct {
		XMLName xml.Name `xml:"markers"`
		Markers []struct {
			ID   int     `xml:"id,attr"`
			Lat  float64 `xml:"lat,attr"`
			Lng  float64 `xml:"lng,attr"`
			Name string  `xml:"name,attr"`
		} `xml:"marker"`
	}
	if err := xml.Unmarshal(data, &tmp); err != nil {
		return err
	}
	for _, it := range tmp.Markers {
		sl.list[it.ID] = Station{ID: it.ID, Lat: it.Lat, Lng: it.Lng, Name: it.Name}
	}
	return nil
}

func (sl *StationList) UpdateStation(id int) error {
	var tmp struct {
		XMLName  xml.Name `xml:"station"`
		Adress   string   `xml:"adress"`
		Status   int      `xml:"status"`
		Bikes    int      `xml:"bikes"`
		Attachs  int      `xml:"attachs"`
		Paiement string   `xml:"paiement"`
	}
	data, err := fetchURL(fmt.Sprintf(urlTplStation, id))
	if err != nil {
		return err
	}
	if err := xml.Unmarshal(data, &tmp); err != nil {
		return err
	}
	station := sl.list[id]
	station.Adress = strings.TrimSpace(tmp.Adress)
	station.Status = tmp.Status
	station.Bikes = tmp.Bikes
	station.Slots = tmp.Attachs
	station.SellTickets = strings.HasPrefix(tmp.Paiement, "AVEC_")
	sl.list[id] = station
	return nil
}

func (sl StationList) UpdateAll() []error {
	var errs []error
	for id := range sl.list {
		if err := sl.UpdateStation(id); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

type CacheError struct {
	internalErr error
}

func (ce *CacheError) Error() string {
	return fmt.Sprintf("Cache operation failed: %s", ce.internalErr)
}

func (sl StationList) SaveCache() error {
	if sl.cachePath == "" {
		return nil
	}

	raw, err := ioutil.ReadFile(sl.cachePath)
	if err != nil && !os.IsNotExist(err) {
		return &CacheError{internalErr: err}
	}
	var list StationSlice
	if err == nil {
		if err := json.Unmarshal(raw, &list); err != nil {
			return &CacheError{internalErr: err}
		}
	}

	tmp, _ := New(false)
	for _, it := range list {
		tmp.list[it.ID] = it
	}
	for id, it := range sl.list {
		tmp.list[id] = it
	}

	raw, err = json.Marshal(tmp.List())
	if err != nil {
		log.Printf("Failed to serialize stations: %s", err)
		return nil
	}
	os.MkdirAll(path.Dir(sl.cachePath), 0755)
	if err := ioutil.WriteFile(sl.cachePath, raw, 0644); err != nil {
		return &CacheError{internalErr: err}
	}
	return nil
}
