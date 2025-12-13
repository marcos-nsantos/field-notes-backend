package valueobject

type BoundingBox struct {
	MinLat float64
	MaxLat float64
	MinLng float64
	MaxLng float64
}

func NewBoundingBox(minLat, maxLat, minLng, maxLng float64) *BoundingBox {
	return &BoundingBox{
		MinLat: minLat,
		MaxLat: maxLat,
		MinLng: minLng,
		MaxLng: maxLng,
	}
}

func (bb *BoundingBox) IsValid() bool {
	return bb.MinLat <= bb.MaxLat &&
		bb.MinLng <= bb.MaxLng &&
		bb.MinLat >= -90 && bb.MaxLat <= 90 &&
		bb.MinLng >= -180 && bb.MaxLng <= 180
}

func (bb *BoundingBox) Contains(lat, lng float64) bool {
	return lat >= bb.MinLat && lat <= bb.MaxLat &&
		lng >= bb.MinLng && lng <= bb.MaxLng
}
