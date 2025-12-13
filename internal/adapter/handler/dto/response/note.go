package response

import (
	"time"

	"github.com/google/uuid"

	"github.com/marcos-nsantos/field-notes-backend/internal/domain/entity"
	"github.com/marcos-nsantos/field-notes-backend/internal/pkg/pagination"
)

type NoteResponse struct {
	ID        uuid.UUID         `json:"id"`
	Title     string            `json:"title"`
	Content   string            `json:"content"`
	Location  *LocationResponse `json:"location,omitempty"`
	Photos    []PhotoResponse   `json:"photos"`
	ClientID  string            `json:"client_id,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	DeletedAt *time.Time        `json:"deleted_at,omitempty"`
}

type LocationResponse struct {
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Altitude  *float64 `json:"altitude,omitempty"`
	Accuracy  *float64 `json:"accuracy,omitempty"`
}

type PhotoResponse struct {
	ID        uuid.UUID `json:"id"`
	URL       string    `json:"url"`
	MimeType  string    `json:"mime_type"`
	Size      int64     `json:"size"`
	Width     int       `json:"width,omitempty"`
	Height    int       `json:"height,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type PaginationResponse struct {
	Page       int  `json:"page"`
	PerPage    int  `json:"per_page"`
	TotalItems int  `json:"total_items"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

type NotesListResponse struct {
	Notes      []NoteResponse     `json:"notes"`
	Pagination PaginationResponse `json:"pagination"`
}

func NoteFromEntity(n *entity.Note) NoteResponse {
	resp := NoteResponse{
		ID:        n.ID,
		Title:     n.Title,
		Content:   n.Content,
		ClientID:  n.ClientID,
		Photos:    make([]PhotoResponse, 0, len(n.Photos)),
		CreatedAt: n.CreatedAt,
		UpdatedAt: n.UpdatedAt,
		DeletedAt: n.DeletedAt,
	}

	if n.Location != nil {
		resp.Location = &LocationResponse{
			Latitude:  n.Location.Latitude,
			Longitude: n.Location.Longitude,
			Altitude:  n.Location.Altitude,
			Accuracy:  n.Location.Accuracy,
		}
	}

	for _, p := range n.Photos {
		resp.Photos = append(resp.Photos, PhotoFromEntity(&p))
	}

	return resp
}

func NotesFromEntities(notes []entity.Note) []NoteResponse {
	result := make([]NoteResponse, 0, len(notes))
	for _, n := range notes {
		result = append(result, NoteFromEntity(&n))
	}
	return result
}

func PhotoFromEntity(p *entity.Photo) PhotoResponse {
	return PhotoResponse{
		ID:        p.ID,
		URL:       p.URL,
		MimeType:  p.MimeType,
		Size:      p.Size,
		Width:     p.Width,
		Height:    p.Height,
		CreatedAt: p.CreatedAt,
	}
}

func PaginationFromInfo(info *pagination.Info) PaginationResponse {
	return PaginationResponse{
		Page:       info.Page,
		PerPage:    info.PerPage,
		TotalItems: info.TotalItems,
		TotalPages: info.TotalPages,
		HasNext:    info.HasNext,
		HasPrev:    info.HasPrev,
	}
}
