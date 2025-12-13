package note

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/repository"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/entity"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/valueobject"
	"github.com/marcos-nsantos/field-notes-backend/internal/pkg/pagination"
)

type Service struct {
	noteRepo  repository.NoteRepository
	photoRepo repository.PhotoRepository
}

func NewService(noteRepo repository.NoteRepository, photoRepo repository.PhotoRepository) *Service {
	return &Service{
		noteRepo:  noteRepo,
		photoRepo: photoRepo,
	}
}

type CreateInput struct {
	UserID   uuid.UUID
	Title    string
	Content  string
	Location *valueobject.Location
	ClientID string
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*entity.Note, error) {
	if input.ClientID != "" {
		existing, err := s.noteRepo.GetByClientID(ctx, input.UserID, input.ClientID)
		if err == nil && existing != nil {
			return existing, nil
		}
	}

	note := entity.NewNote(input.UserID, input.Title, input.Content, input.Location, input.ClientID)

	if err := s.noteRepo.Create(ctx, note); err != nil {
		return nil, fmt.Errorf("creating note: %w", err)
	}

	return note, nil
}

type ListInput struct {
	UserID      uuid.UUID
	Page        int
	PerPage     int
	BoundingBox *valueobject.BoundingBox
}

func (s *Service) List(ctx context.Context, input ListInput) ([]entity.Note, *pagination.Info, error) {
	params := repository.NoteListParams{
		Pagination:     pagination.NewParams(input.Page, input.PerPage),
		BoundingBox:    input.BoundingBox,
		IncludeDeleted: false,
	}

	notes, pageInfo, err := s.noteRepo.List(ctx, input.UserID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("listing notes: %w", err)
	}

	for i := range notes {
		photos, err := s.photoRepo.GetByNoteID(ctx, notes[i].ID)
		if err != nil {
			return nil, nil, fmt.Errorf("loading photos: %w", err)
		}
		notes[i].Photos = photos
	}

	return notes, pageInfo, nil
}

func (s *Service) GetByID(ctx context.Context, userID, noteID uuid.UUID) (*entity.Note, error) {
	note, err := s.noteRepo.GetByID(ctx, noteID)
	if err != nil {
		return nil, err
	}

	if note.UserID != userID {
		return nil, domain.ErrForbidden
	}

	if note.IsDeleted() {
		return nil, domain.ErrNoteNotFound
	}

	photos, err := s.photoRepo.GetByNoteID(ctx, noteID)
	if err != nil {
		return nil, fmt.Errorf("loading photos: %w", err)
	}
	note.Photos = photos

	return note, nil
}

type UpdateInput struct {
	Title    *string
	Content  *string
	Location *valueobject.Location
}

func (s *Service) Update(ctx context.Context, userID, noteID uuid.UUID, input UpdateInput) (*entity.Note, error) {
	note, err := s.noteRepo.GetByID(ctx, noteID)
	if err != nil {
		return nil, err
	}

	if note.UserID != userID {
		return nil, domain.ErrForbidden
	}

	if note.IsDeleted() {
		return nil, domain.ErrNoteNotFound
	}

	title := note.Title
	content := note.Content
	location := note.Location

	if input.Title != nil {
		title = *input.Title
	}
	if input.Content != nil {
		content = *input.Content
	}
	if input.Location != nil {
		location = input.Location
	}

	note.Update(title, content, location)

	if err := s.noteRepo.Update(ctx, note); err != nil {
		return nil, fmt.Errorf("updating note: %w", err)
	}

	photos, err := s.photoRepo.GetByNoteID(ctx, noteID)
	if err != nil {
		return nil, fmt.Errorf("loading photos: %w", err)
	}
	note.Photos = photos

	return note, nil
}

func (s *Service) Delete(ctx context.Context, userID, noteID uuid.UUID) error {
	note, err := s.noteRepo.GetByID(ctx, noteID)
	if err != nil {
		return err
	}

	if note.UserID != userID {
		return domain.ErrForbidden
	}

	if err := s.noteRepo.SoftDelete(ctx, noteID); err != nil {
		return fmt.Errorf("deleting note: %w", err)
	}

	return nil
}
