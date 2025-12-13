package upload

import (
	"context"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/google/uuid"

	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/repository"
	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/storage"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/entity"
)

type Service struct {
	photoRepo      repository.PhotoRepository
	noteRepo       repository.NoteRepository
	storage        storage.ImageStorage
	imageProcessor storage.ImageProcessor
}

func NewService(
	photoRepo repository.PhotoRepository,
	noteRepo repository.NoteRepository,
	imageStorage storage.ImageStorage,
	imageProcessor storage.ImageProcessor,
) *Service {
	return &Service{
		photoRepo:      photoRepo,
		noteRepo:       noteRepo,
		storage:        imageStorage,
		imageProcessor: imageProcessor,
	}
}

type UploadInput struct {
	UserID      uuid.UUID
	NoteID      uuid.UUID
	File        io.Reader
	Filename    string
	ContentType string
	Size        int64
}

type UploadResult struct {
	Photo     *entity.Photo
	URL       string
	SignedURL string
}

func (s *Service) Upload(ctx context.Context, input UploadInput) (*UploadResult, error) {
	note, err := s.noteRepo.GetByID(ctx, input.NoteID)
	if err != nil {
		return nil, err
	}

	if note.UserID != input.UserID {
		return nil, domain.ErrForbidden
	}

	if note.IsDeleted() {
		return nil, domain.ErrNoteNotFound
	}

	processedReader, finalSize, width, height, err := s.imageProcessor.Process(input.File, input.ContentType)
	if err != nil {
		return nil, fmt.Errorf("processing image: %w", err)
	}

	ext := path.Ext(input.Filename)
	if ext == "" {
		ext = ".jpg"
	}
	key := fmt.Sprintf("notes/%s/%s%s", input.NoteID, uuid.New().String(), ext)

	if err := s.storage.Upload(ctx, key, processedReader, input.ContentType, finalSize); err != nil {
		return nil, fmt.Errorf("uploading to storage: %w", err)
	}

	url := s.storage.GetURL(key)
	signedURL, _ := s.storage.GetSignedURL(key, 24*time.Hour)

	photo := entity.NewPhoto(input.NoteID, url, key, input.ContentType, finalSize, width, height)

	if err := s.photoRepo.Create(ctx, photo); err != nil {
		_ = s.storage.Delete(ctx, key)
		return nil, fmt.Errorf("creating photo record: %w", err)
	}

	return &UploadResult{
		Photo:     photo,
		URL:       url,
		SignedURL: signedURL,
	}, nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID uuid.UUID) error {
	photo, err := s.photoRepo.GetByID(ctx, photoID)
	if err != nil {
		return err
	}

	note, err := s.noteRepo.GetByID(ctx, photo.NoteID)
	if err != nil {
		return err
	}

	if note.UserID != userID {
		return domain.ErrForbidden
	}

	if err := s.photoRepo.Delete(ctx, photoID); err != nil {
		return fmt.Errorf("deleting photo record: %w", err)
	}

	if err := s.storage.Delete(ctx, photo.Key); err != nil {
		return fmt.Errorf("deleting from storage: %w", err)
	}

	return nil
}
