package upload_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/marcos-nsantos/field-notes-backend/internal/domain"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/entity"
	"github.com/marcos-nsantos/field-notes-backend/internal/mocks"
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/upload"
)

func TestService_Upload(t *testing.T) {
	t.Run("uploads image successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		noteRepo := mocks.NewMockNoteRepository(ctrl)
		storage := mocks.NewMockImageStorage(ctrl)
		imageProcessor := mocks.NewMockImageProcessor(ctrl)
		svc := upload.NewService(photoRepo, noteRepo, storage, imageProcessor)

		ctx := context.Background()
		userID := uuid.New()
		noteID := uuid.New()
		note := &entity.Note{ID: noteID, UserID: userID, Title: "Test Note"}

		fileContent := []byte("fake image data")
		processedContent := []byte("processed image data")
		processedReader := bytes.NewReader(processedContent)

		noteRepo.EXPECT().GetByID(ctx, noteID).Return(note, nil)
		imageProcessor.EXPECT().Process(gomock.Any(), "image/jpeg").Return(processedReader, int64(len(processedContent)), 800, 600, nil)
		storage.EXPECT().Upload(ctx, gomock.Any(), processedReader, "image/jpeg", int64(len(processedContent))).Return(nil)
		storage.EXPECT().GetURL(gomock.Any()).Return("http://storage/photo.jpg")
		storage.EXPECT().GetSignedURL(gomock.Any(), 24*time.Hour).Return("http://storage/photo.jpg?signed=1", nil)
		photoRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		result, err := svc.Upload(ctx, upload.UploadInput{
			UserID:      userID,
			NoteID:      noteID,
			File:        bytes.NewReader(fileContent),
			Filename:    "photo.jpg",
			ContentType: "image/jpeg",
			Size:        int64(len(fileContent)),
		})

		require.NoError(t, err)
		assert.NotNil(t, result.Photo)
		assert.Equal(t, "http://storage/photo.jpg", result.URL)
		assert.Contains(t, result.SignedURL, "signed")
	})

	t.Run("returns forbidden for non-owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		noteRepo := mocks.NewMockNoteRepository(ctrl)
		storage := mocks.NewMockImageStorage(ctrl)
		imageProcessor := mocks.NewMockImageProcessor(ctrl)
		svc := upload.NewService(photoRepo, noteRepo, storage, imageProcessor)

		ctx := context.Background()
		ownerID := uuid.New()
		otherUserID := uuid.New()
		noteID := uuid.New()
		note := &entity.Note{ID: noteID, UserID: ownerID, Title: "Test Note"}

		noteRepo.EXPECT().GetByID(ctx, noteID).Return(note, nil)

		result, err := svc.Upload(ctx, upload.UploadInput{
			UserID:      otherUserID,
			NoteID:      noteID,
			File:        bytes.NewReader([]byte("data")),
			Filename:    "photo.jpg",
			ContentType: "image/jpeg",
			Size:        4,
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("returns not found for deleted note", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		noteRepo := mocks.NewMockNoteRepository(ctrl)
		storage := mocks.NewMockImageStorage(ctrl)
		imageProcessor := mocks.NewMockImageProcessor(ctrl)
		svc := upload.NewService(photoRepo, noteRepo, storage, imageProcessor)

		ctx := context.Background()
		userID := uuid.New()
		noteID := uuid.New()
		deletedAt := time.Now()
		note := &entity.Note{ID: noteID, UserID: userID, Title: "Test Note", DeletedAt: &deletedAt}

		noteRepo.EXPECT().GetByID(ctx, noteID).Return(note, nil)

		result, err := svc.Upload(ctx, upload.UploadInput{
			UserID:      userID,
			NoteID:      noteID,
			File:        bytes.NewReader([]byte("data")),
			Filename:    "photo.jpg",
			ContentType: "image/jpeg",
			Size:        4,
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, domain.ErrNoteNotFound)
	})

	t.Run("returns not found for missing note", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		noteRepo := mocks.NewMockNoteRepository(ctrl)
		storage := mocks.NewMockImageStorage(ctrl)
		imageProcessor := mocks.NewMockImageProcessor(ctrl)
		svc := upload.NewService(photoRepo, noteRepo, storage, imageProcessor)

		ctx := context.Background()
		userID := uuid.New()
		noteID := uuid.New()

		noteRepo.EXPECT().GetByID(ctx, noteID).Return(nil, domain.ErrNoteNotFound)

		result, err := svc.Upload(ctx, upload.UploadInput{
			UserID:      userID,
			NoteID:      noteID,
			File:        bytes.NewReader([]byte("data")),
			Filename:    "photo.jpg",
			ContentType: "image/jpeg",
			Size:        4,
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, domain.ErrNoteNotFound)
	})

	t.Run("cleans up storage on db error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		noteRepo := mocks.NewMockNoteRepository(ctrl)
		storageClient := mocks.NewMockImageStorage(ctrl)
		imageProcessor := mocks.NewMockImageProcessor(ctrl)
		svc := upload.NewService(photoRepo, noteRepo, storageClient, imageProcessor)

		ctx := context.Background()
		userID := uuid.New()
		noteID := uuid.New()
		note := &entity.Note{ID: noteID, UserID: userID, Title: "Test Note"}

		processedReader := bytes.NewReader([]byte("processed"))

		noteRepo.EXPECT().GetByID(ctx, noteID).Return(note, nil)
		imageProcessor.EXPECT().Process(gomock.Any(), "image/jpeg").Return(processedReader, int64(9), 800, 600, nil)
		storageClient.EXPECT().Upload(ctx, gomock.Any(), processedReader, "image/jpeg", int64(9)).Return(nil)
		storageClient.EXPECT().GetURL(gomock.Any()).Return("http://storage/photo.jpg")
		storageClient.EXPECT().GetSignedURL(gomock.Any(), 24*time.Hour).Return("http://storage/photo.jpg?signed=1", nil)
		photoRepo.EXPECT().Create(ctx, gomock.Any()).Return(domain.ErrPhotoNotFound)
		storageClient.EXPECT().Delete(ctx, gomock.Any()).Return(nil)

		result, err := svc.Upload(ctx, upload.UploadInput{
			UserID:      userID,
			NoteID:      noteID,
			File:        bytes.NewReader([]byte("data")),
			Filename:    "photo.jpg",
			ContentType: "image/jpeg",
			Size:        4,
		})

		assert.Nil(t, result)
		assert.Error(t, err)
	})
}

func TestService_Delete(t *testing.T) {
	t.Run("deletes photo successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		noteRepo := mocks.NewMockNoteRepository(ctrl)
		storageClient := mocks.NewMockImageStorage(ctrl)
		imageProcessor := mocks.NewMockImageProcessor(ctrl)
		svc := upload.NewService(photoRepo, noteRepo, storageClient, imageProcessor)

		ctx := context.Background()
		userID := uuid.New()
		noteID := uuid.New()
		photoID := uuid.New()
		photo := &entity.Photo{ID: photoID, NoteID: noteID, Key: "notes/123/photo.jpg"}
		note := &entity.Note{ID: noteID, UserID: userID}

		photoRepo.EXPECT().GetByID(ctx, photoID).Return(photo, nil)
		noteRepo.EXPECT().GetByID(ctx, noteID).Return(note, nil)
		photoRepo.EXPECT().Delete(ctx, photoID).Return(nil)
		storageClient.EXPECT().Delete(ctx, "notes/123/photo.jpg").Return(nil)

		err := svc.Delete(ctx, userID, photoID)

		require.NoError(t, err)
	})

	t.Run("returns forbidden for non-owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		noteRepo := mocks.NewMockNoteRepository(ctrl)
		storageClient := mocks.NewMockImageStorage(ctrl)
		imageProcessor := mocks.NewMockImageProcessor(ctrl)
		svc := upload.NewService(photoRepo, noteRepo, storageClient, imageProcessor)

		ctx := context.Background()
		ownerID := uuid.New()
		otherUserID := uuid.New()
		noteID := uuid.New()
		photoID := uuid.New()
		photo := &entity.Photo{ID: photoID, NoteID: noteID}
		note := &entity.Note{ID: noteID, UserID: ownerID}

		photoRepo.EXPECT().GetByID(ctx, photoID).Return(photo, nil)
		noteRepo.EXPECT().GetByID(ctx, noteID).Return(note, nil)

		err := svc.Delete(ctx, otherUserID, photoID)

		assert.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("returns not found for missing photo", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		noteRepo := mocks.NewMockNoteRepository(ctrl)
		storageClient := mocks.NewMockImageStorage(ctrl)
		imageProcessor := mocks.NewMockImageProcessor(ctrl)
		svc := upload.NewService(photoRepo, noteRepo, storageClient, imageProcessor)

		ctx := context.Background()
		userID := uuid.New()
		photoID := uuid.New()

		photoRepo.EXPECT().GetByID(ctx, photoID).Return(nil, domain.ErrPhotoNotFound)

		err := svc.Delete(ctx, userID, photoID)

		assert.ErrorIs(t, err, domain.ErrPhotoNotFound)
	})
}
