package types

const (
    BUS_TIP_INFO = "BusTipInfo"
    BUS_GPS_DATA = "BusGPSData"
)

type BusTipInfo struct {
	Name string
	Tip string
}

type BusGPSData struct {
	Name      string
	Latitude  float64
	Longitude float64
}
