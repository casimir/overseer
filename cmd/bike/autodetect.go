package main

func getPosition(cfg *config) (float64, float64, bool) {
	if wifiTable, hasWifiCfg := cfg.Location["wifi"]; hasWifiCfg {
		if ssid, foundSsid := getSsid(); foundSsid {
			if pos, ok := wifiTable[ssid]; ok {
				return pos.Lat, pos.Lng, true
			}
		}
	}
	return 0.0, 0.0, false
}
