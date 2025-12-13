package note_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/marcos-nsantos/field-notes-backend/internal/domain"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/entity"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/valueobject"
	"github.com/marcos-nsantos/field-notes-backend/internal/mocks"
	"github.com/marcos-nsantos/field-notes-backend/internal/pkg/pagination"
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/note"
)

func TestService_Create(t *testing.T) {
	t.Run("creates note successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		svc := note.NewService(noteRepo, photoRepo)

		ctx := context.Background()
		userID := uuid.New()
		loc := valueobject.NewLocation(37.7749, -122.4194, nil, nil)

		noteRepo.EXPECT().GetByClientID(ctx, userID, "client-123").Return(nil, domain.ErrNoteNotFound)
		noteRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		n, err := svc.Create(ctx, note.CreateInput{
			UserID:   userID,
			Title:    "Test Note",
			Content:  "Test content",
			Location: loc,
			ClientID: "client-123",
		})

		require.NoError(t, err)
		assert.Equal(t, "Test Note", n.Title)
		assert.Equal(t, "Test content", n.Content)
		assert.Equal(t, userID, n.UserID)
		assert.Equal(t, loc.Latitude, n.Location.Latitude)
	})

	t.Run("returns existing note with same client_id (idempotent)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		svc := note.NewService(noteRepo, photoRepo)

		ctx := context.Background()
		userID := uuid.New()
		existingNote := &entity.Note{
			ID:       uuid.New(),
			UserID:   userID,
			Title:    "Existing Note",
			Content:  "Existing content",
			ClientID: "client-123",
		}

		noteRepo.EXPECT().GetByClientID(ctx, userID, "client-123").Return(existingNote, nil)

		n, err := svc.Create(ctx, note.CreateInput{
			UserID:   userID,
			Title:    "New Note",
			Content:  "New content",
			ClientID: "client-123",
		})

		require.NoError(t, err)
		assert.Equal(t, existingNote.ID, n.ID)
		assert.Equal(t, "Existing Note", n.Title)
	})

	t.Run("creates note without client_id", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		svc := note.NewService(noteRepo, photoRepo)

		ctx := context.Background()
		userID := uuid.New()

		noteRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		n, err := svc.Create(ctx, note.CreateInput{
			UserID:  userID,
			Title:   "Test Note",
			Content: "Test content",
		})

		require.NoError(t, err)
		assert.Equal(t, "Test Note", n.Title)
		assert.Empty(t, n.ClientID)
	})
}

func TestService_List(t *testing.T) {
	t.Run("lists notes with pagination", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		svc := note.NewService(noteRepo, photoRepo)

		ctx := context.Background()
		userID := uuid.New()
		noteID := uuid.New()

		notes := []entity.Note{
			{ID: noteID, UserID: userID, Title: "Note 1", Content: "Content 1"},
		}
		pageInfo := &pagination.Info{Page: 1, PerPage: 20, TotalItems: 1, TotalPages: 1}

		noteRepo.EXPECT().List(ctx, userID, gomock.Any()).Return(notes, pageInfo, nil)
		photoRepo.EXPECT().GetByNoteID(ctx, noteID).Return([]entity.Photo{}, nil)

		result, info, err := svc.List(ctx, note.ListInput{
			UserID:  userID,
			Page:    1,
			PerPage: 20,
		})

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Note 1", result[0].Title)
		assert.Equal(t, 1, info.TotalItems)
	})

	t.Run("lists notes with bounding box filter", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		svc := note.NewService(noteRepo, photoRepo)

		ctx := context.Background()
		userID := uuid.New()
		noteID := uuid.New()
		bbox := valueobject.NewBoundingBox(37.0, 38.0, -123.0, -122.0)

		notes := []entity.Note{
			{ID: noteID, UserID: userID, Title: "SF Note", Location: valueobject.NewLocation(37.7749, -122.4194, nil, nil)},
		}
		pageInfo := &pagination.Info{Page: 1, PerPage: 20, TotalItems: 1, TotalPages: 1}

		noteRepo.EXPECT().List(ctx, userID, gomock.Any()).Return(notes, pageInfo, nil)
		photoRepo.EXPECT().GetByNoteID(ctx, noteID).Return([]entity.Photo{}, nil)

		result, _, err := svc.List(ctx, note.ListInput{
			UserID:      userID,
			Page:        1,
			PerPage:     20,
			BoundingBox: bbox,
		})

		require.NoError(t, err)
		assert.Len(t, result, 1)
	})
}

func TestService_GetByID(t *testing.T) {
	t.Run("returns note for owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		svc := note.NewService(noteRepo, photoRepo)

		ctx := context.Background()
		userID := uuid.New()
		noteID := uuid.New()
		n := &entity.Note{ID: noteID, UserID: userID, Title: "Test Note"}

		noteRepo.EXPECT().GetByID(ctx, noteID).Return(n, nil)
		photoRepo.EXPECT().GetByNoteID(ctx, noteID).Return([]entity.Photo{}, nil)

		result, err := svc.GetByID(ctx, userID, noteID)

		require.NoError(t, err)
		assert.Equal(t, noteID, result.ID)
	})

	t.Run("returns forbidden for non-owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		svc := note.NewService(noteRepo, photoRepo)

		ctx := context.Background()
		ownerID := uuid.New()
		otherUserID := uuid.New()
		noteID := uuid.New()
		n := &entity.Note{ID: noteID, UserID: ownerID, Title: "Test Note"}

		noteRepo.EXPECT().GetByID(ctx, noteID).Return(n, nil)

		result, err := svc.GetByID(ctx, otherUserID, noteID)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("returns not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		svc := note.NewService(noteRepo, photoRepo)

		ctx := context.Background()
		userID := uuid.New()
		noteID := uuid.New()

		noteRepo.EXPECT().GetByID(ctx, noteID).Return(nil, domain.ErrNoteNotFound)

		result, err := svc.GetByID(ctx, userID, noteID)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, domain.ErrNoteNotFound)
	})

	t.Run("returns not found for deleted note", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		svc := note.NewService(noteRepo, photoRepo)

		ctx := context.Background()
		userID := uuid.New()
		noteID := uuid.New()
		deletedAt := time.Now()
		n := &entity.Note{ID: noteID, UserID: userID, DeletedAt: &deletedAt}

		noteRepo.EXPECT().GetByID(ctx, noteID).Return(n, nil)

		result, err := svc.GetByID(ctx, userID, noteID)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, domain.ErrNoteNotFound)
	})
}

func TestService_Update(t *testing.T) {
	t.Run("updates note successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		svc := note.NewService(noteRepo, photoRepo)

		ctx := context.Background()
		userID := uuid.New()
		noteID := uuid.New()
		n := &entity.Note{ID: noteID, UserID: userID, Title: "Old Title", Content: "Old Content"}

		noteRepo.EXPECT().GetByID(ctx, noteID).Return(n, nil)
		noteRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
		photoRepo.EXPECT().GetByNoteID(ctx, noteID).Return([]entity.Photo{}, nil)

		newTitle := "New Title"
		newContent := "New Content"
		result, err := svc.Update(ctx, userID, noteID, note.UpdateInput{
			Title:   &newTitle,
			Content: &newContent,
		})

		require.NoError(t, err)
		assert.Equal(t, "New Title", result.Title)
		assert.Equal(t, "New Content", result.Content)
	})

	t.Run("returns forbidden for non-owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		svc := note.NewService(noteRepo, photoRepo)

		ctx := context.Background()
		ownerID := uuid.New()
		otherUserID := uuid.New()
		noteID := uuid.New()
		n := &entity.Note{ID: noteID, UserID: ownerID}

		noteRepo.EXPECT().GetByID(ctx, noteID).Return(n, nil)

		newTitle := "New Title"
		result, err := svc.Update(ctx, otherUserID, noteID, note.UpdateInput{Title: &newTitle})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("returns not found for deleted note", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		svc := note.NewService(noteRepo, photoRepo)

		ctx := context.Background()
		userID := uuid.New()
		noteID := uuid.New()
		deletedAt := time.Now()
		n := &entity.Note{ID: noteID, UserID: userID, DeletedAt: &deletedAt}

		noteRepo.EXPECT().GetByID(ctx, noteID).Return(n, nil)

		newTitle := "New Title"
		result, err := svc.Update(ctx, userID, noteID, note.UpdateInput{Title: &newTitle})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, domain.ErrNoteNotFound)
	})
}

func TestService_Delete(t *testing.T) {
	t.Run("soft deletes note successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		svc := note.NewService(noteRepo, photoRepo)

		ctx := context.Background()
		userID := uuid.New()
		noteID := uuid.New()
		n := &entity.Note{ID: noteID, UserID: userID}

		noteRepo.EXPECT().GetByID(ctx, noteID).Return(n, nil)
		noteRepo.EXPECT().SoftDelete(ctx, noteID).Return(nil)

		err := svc.Delete(ctx, userID, noteID)

		require.NoError(t, err)
	})

	t.Run("returns forbidden for non-owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		photoRepo := mocks.NewMockPhotoRepository(ctrl)
		svc := note.NewService(noteRepo, photoRepo)

		ctx := context.Background()
		ownerID := uuid.New()
		otherUserID := uuid.New()
		noteID := uuid.New()
		n := &entity.Note{ID: noteID, UserID: ownerID}

		noteRepo.EXPECT().GetByID(ctx, noteID).Return(n, nil)

		err := svc.Delete(ctx, otherUserID, noteID)

		assert.ErrorIs(t, err, domain.ErrForbidden)
	})
}
