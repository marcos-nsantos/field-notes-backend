package request

import "time"

type SyncRequest struct {
	DeviceID   string     `json:"device_id" binding:"required,max=255"`
	SyncCursor *time.Time `json:"sync_cursor"`
	Notes      []SyncNote `json:"notes" binding:"dive"`
}

type SyncNote struct {
	ClientID  string    `json:"client_id" binding:"required,max=36"`
	Title     string    `json:"title" binding:"required,max=255"`
	Content   string    `json:"content" binding:"required"`
	Latitude  *float64  `json:"latitude" binding:"omitempty,min=-90,max=90"`
	Longitude *float64  `json:"longitude" binding:"omitempty,min=-180,max=180"`
	Altitude  *float64  `json:"altitude"`
	Accuracy  *float64  `json:"accuracy" binding:"omitempty,min=0"`
	UpdatedAt time.Time `json:"updated_at" binding:"required"`
	IsDeleted bool      `json:"is_deleted"`
}
