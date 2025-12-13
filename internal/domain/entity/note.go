package entity

import (
	"time"

	"github.com/google/uuid"

	"github.com/marcos-nsantos/field-notes-backend/internal/domain/valueobject"
)

type Note struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Title     string
	Content   string
	Location  *valueobject.Location
	Photos    []Photo
	ClientID  string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func NewNote(userID uuid.UUID, title, content string, loc *valueobject.Location, clientID string) *Note {
	now := time.Now().UTC()
	return &Note{
		ID:        uuid.New(),
		UserID:    userID,
		Title:     title,
		Content:   content,
		Location:  loc,
		ClientID:  clientID,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (n *Note) Update(title, content string, loc *valueobject.Location) {
	n.Title = title
	n.Content = content
	n.Location = loc
	n.UpdatedAt = time.Now().UTC()
}

func (n *Note) SoftDelete() {
	now := time.Now().UTC()
	n.DeletedAt = &now
	n.UpdatedAt = now
}

func (n *Note) Restore() {
	n.DeletedAt = nil
	n.UpdatedAt = time.Now().UTC()
}

func (n *Note) IsDeleted() bool {
	return n.DeletedAt != nil
}
