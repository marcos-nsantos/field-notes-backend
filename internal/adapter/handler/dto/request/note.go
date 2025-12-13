package request

type CreateNoteRequest struct {
	Title     string   `json:"title" binding:"required,max=255"`
	Content   string   `json:"content" binding:"required"`
	Latitude  *float64 `json:"latitude" binding:"omitempty,min=-90,max=90"`
	Longitude *float64 `json:"longitude" binding:"omitempty,min=-180,max=180"`
	Altitude  *float64 `json:"altitude"`
	Accuracy  *float64 `json:"accuracy" binding:"omitempty,min=0"`
	ClientID  string   `json:"client_id" binding:"omitempty,max=36"`
}

type UpdateNoteRequest struct {
	Title     *string  `json:"title" binding:"omitempty,max=255"`
	Content   *string  `json:"content"`
	Latitude  *float64 `json:"latitude" binding:"omitempty,min=-90,max=90"`
	Longitude *float64 `json:"longitude" binding:"omitempty,min=-180,max=180"`
	Altitude  *float64 `json:"altitude"`
	Accuracy  *float64 `json:"accuracy" binding:"omitempty,min=0"`
}

type ListNotesRequest struct {
	Page    int      `form:"page" binding:"omitempty,min=1"`
	PerPage int      `form:"per_page" binding:"omitempty,min=1,max=100"`
	MinLat  *float64 `form:"min_lat" binding:"omitempty,min=-90,max=90"`
	MaxLat  *float64 `form:"max_lat" binding:"omitempty,min=-90,max=90"`
	MinLng  *float64 `form:"min_lng" binding:"omitempty,min=-180,max=180"`
	MaxLng  *float64 `form:"max_lng" binding:"omitempty,min=-180,max=180"`
}
