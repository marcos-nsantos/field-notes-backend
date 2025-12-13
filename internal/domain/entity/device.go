package entity

import (
	"time"

	"github.com/google/uuid"
)

type Device struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	DeviceID   string
	Platform   string
	Name       string
	SyncCursor time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewDevice(userID uuid.UUID, deviceID, platform, name string) *Device {
	now := time.Now().UTC()
	return &Device{
		ID:         uuid.New(),
		UserID:     userID,
		DeviceID:   deviceID,
		Platform:   platform,
		Name:       name,
		SyncCursor: time.Time{},
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func (d *Device) UpdateSyncCursor(cursor time.Time) {
	d.SyncCursor = cursor
	d.UpdatedAt = time.Now().UTC()
}
