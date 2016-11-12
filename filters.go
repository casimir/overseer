package overseer

func HasBike(st Station) bool {
	return st.Bikes > 0
}

func HasSlot(st Station) bool {
	return st.Slots > 0
}

func SellsTickets(st Station) bool {
	return st.SellsTickets
}
