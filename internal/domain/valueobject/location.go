package valueobject

type Location struct {
	Latitude  float64
	Longitude float64
	Altitude  *float64
	Accuracy  *float64
}

func NewLocation(lat, lng float64, altitude, accuracy *float64) *Location {
	return &Location{
		Latitude:  lat,
		Longitude: lng,
		Altitude:  altitude,
		Accuracy:  accuracy,
	}
}

func (l *Location) IsValid() bool {
	return l.Latitude >= -90 && l.Latitude <= 90 &&
		l.Longitude >= -180 && l.Longitude <= 180
}
