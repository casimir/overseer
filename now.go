package overseer

type Now struct {
	Bike   *GeoStation `json:"bike"`
	Slot   *GeoStation `json:"slot"`
	Ticket *GeoStation `json:"ticket"`
}

func NewNow(sl GeoStationList) Now {
	ret := Now{}
	foundBike, foundSlot, foundTicket := false, false, false
	for _, it := range sl {
		if foundBike && foundSlot && foundTicket {
			break
		}
		station := it
		if !foundBike && HasBike(it.Station) {
			ret.Bike = &station
			foundBike = true
		}
		if !foundSlot && HasSlot(it.Station) {
			ret.Slot = &station
			foundSlot = true
		}
		if !foundTicket && SellsTickets(it.Station) {
			ret.Ticket = &station
			foundTicket = true
		}
	}
	return ret
}
