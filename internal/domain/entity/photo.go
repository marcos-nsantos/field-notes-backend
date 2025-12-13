package entity

import (
	"time"

	"github.com/google/uuid"
)

type Photo struct {
	ID        uuid.UUID
	NoteID    uuid.UUID
	URL       string
	Key       string
	MimeType  string
	Size      int64
	Width     int
	Height    int
	CreatedAt time.Time
}

func NewPhoto(noteID uuid.UUID, url, key, mimeType string, size int64, width, height int) *Photo {
	return &Photo{
		ID:        uuid.New(),
		NoteID:    noteID,
		URL:       url,
		Key:       key,
		MimeType:  mimeType,
		Size:      size,
		Width:     width,
		Height:    height,
		CreatedAt: time.Now().UTC(),
	}
}
