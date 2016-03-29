package main

var wifiTable = map[string]struct{ lat, lng float64 }{
	"Critizr":        {lat: 50.6333540, lng: 3.0203770},
	"Freebox-578CF5": {lat: 50.6263060, lng: 3.0681140},
	"belkin.789":     {lat: 50.6226860, lng: 3.0570390},
}

func getPosition() (float64, float64, bool) {
	if ssid, ok := getSsid(); ok {
		if pos, ok := wifiTable[ssid]; ok {
			return pos.lat, pos.lng, true
		}
	}
	return 0.0, 0.0, false
}
