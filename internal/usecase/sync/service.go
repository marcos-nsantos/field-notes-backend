package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/repository"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/entity"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/valueobject"
)

type Service struct {
	noteRepo   repository.NoteRepository
	deviceRepo repository.DeviceRepository
}

func NewService(noteRepo repository.NoteRepository, deviceRepo repository.DeviceRepository) *Service {
	return &Service{
		noteRepo:   noteRepo,
		deviceRepo: deviceRepo,
	}
}

type SyncInput struct {
	UserID      uuid.UUID
	DeviceID    string
	ClientNotes []ClientNote
	SyncCursor  *time.Time
}

type ClientNote struct {
	ClientID  string
	Title     string
	Content   string
	Latitude  *float64
	Longitude *float64
	Altitude  *float64
	Accuracy  *float64
	UpdatedAt time.Time
	IsDeleted bool
}

type SyncResult struct {
	ServerNotes []entity.Note
	NewCursor   time.Time
	Conflicts   []ConflictInfo
}

type ConflictInfo struct {
	ClientID      string
	Resolution    string
	ServerVersion *entity.Note
}

const (
	ResolutionClientWins = "client_wins"
	ResolutionServerWins = "server_wins"
)

func (s *Service) BatchSync(ctx context.Context, input SyncInput) (*SyncResult, error) {
	device, err := s.deviceRepo.GetByUserAndDeviceID(ctx, input.UserID, input.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("getting device: %w", err)
	}

	cursor := device.SyncCursor
	if input.SyncCursor != nil {
		cursor = *input.SyncCursor
	}

	serverNotes, err := s.noteRepo.GetModifiedSince(ctx, input.UserID, cursor, 1000)
	if err != nil {
		return nil, fmt.Errorf("getting server changes: %w", err)
	}

	serverNoteMap := make(map[string]*entity.Note)
	for i := range serverNotes {
		if serverNotes[i].ClientID != "" {
			serverNoteMap[serverNotes[i].ClientID] = &serverNotes[i]
		}
	}

	var conflicts []ConflictInfo
	var notesToUpsert []entity.Note

	for _, cn := range input.ClientNotes {
		if cn.ClientID == "" {
			continue
		}

		serverNote, exists := serverNoteMap[cn.ClientID]

		if exists {
			if cn.UpdatedAt.After(serverNote.UpdatedAt) {
				updatedNote := clientNoteToEntity(cn, input.UserID, serverNote.ID)
				notesToUpsert = append(notesToUpsert, updatedNote)
				conflicts = append(conflicts, ConflictInfo{
					ClientID:      cn.ClientID,
					Resolution:    ResolutionClientWins,
					ServerVersion: serverNote,
				})
			} else {
				conflicts = append(conflicts, ConflictInfo{
					ClientID:      cn.ClientID,
					Resolution:    ResolutionServerWins,
					ServerVersion: serverNote,
				})
			}
		} else {
			newNote := clientNoteToEntity(cn, input.UserID, uuid.Nil)
			notesToUpsert = append(notesToUpsert, newNote)
		}
	}

	if len(notesToUpsert) > 0 {
		if err := s.noteRepo.BatchUpsert(ctx, notesToUpsert); err != nil {
			return nil, fmt.Errorf("upserting notes: %w", err)
		}
	}

	newCursor := time.Now().UTC()

	device.UpdateSyncCursor(newCursor)
	if err := s.deviceRepo.Update(ctx, device); err != nil {
		return nil, fmt.Errorf("updating device cursor: %w", err)
	}

	return &SyncResult{
		ServerNotes: serverNotes,
		NewCursor:   newCursor,
		Conflicts:   conflicts,
	}, nil
}

func clientNoteToEntity(cn ClientNote, userID uuid.UUID, existingID uuid.UUID) entity.Note {
	var loc *valueobject.Location
	if cn.Latitude != nil && cn.Longitude != nil {
		loc = valueobject.NewLocation(*cn.Latitude, *cn.Longitude, cn.Altitude, cn.Accuracy)
	}

	id := existingID
	if id == uuid.Nil {
		id = uuid.New()
	}

	note := entity.Note{
		ID:        id,
		UserID:    userID,
		Title:     cn.Title,
		Content:   cn.Content,
		Location:  loc,
		ClientID:  cn.ClientID,
		CreatedAt: cn.UpdatedAt,
		UpdatedAt: cn.UpdatedAt,
	}

	if cn.IsDeleted {
		deletedAt := cn.UpdatedAt
		note.DeletedAt = &deletedAt
	}

	return note
}
