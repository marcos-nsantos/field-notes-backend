package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/repository/postgres"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/entity"
)

func createTestUserAndNote(t *testing.T, db *TestDB) (*entity.User, *entity.Note) {
	t.Helper()
	userRepo := postgres.NewUserRepo(db.Pool)
	noteRepo := postgres.NewNoteRepo(db.Pool)
	ctx := context.Background()

	user := entity.NewUser("test@example.com", "hashedpassword", "Test User")
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	note := entity.NewNote(user.ID, "Test Note", "Content", nil, "")
	err = noteRepo.Create(ctx, note)
	require.NoError(t, err)

	return user, note
}

func TestIntegrationPhotoRepo_Create(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewPhotoRepo(db.Pool)
	ctx := context.Background()

	t.Run("creates photo successfully", func(t *testing.T) {
		db.Truncate(t, "photos", "notes", "users")
		_, note := createTestUserAndNote(t, db)

		photo := entity.NewPhoto(note.ID, "http://storage/photo.jpg", "notes/123/photo.jpg", "image/jpeg", 1024, 800, 600)
		err := repo.Create(ctx, photo)

		require.NoError(t, err)
		assert.NotEmpty(t, photo.ID)
	})
}

func TestIntegrationPhotoRepo_GetByID(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewPhotoRepo(db.Pool)
	ctx := context.Background()

	t.Run("returns photo by ID", func(t *testing.T) {
		db.Truncate(t, "photos", "notes", "users")
		_, note := createTestUserAndNote(t, db)

		photo := entity.NewPhoto(note.ID, "http://storage/photo.jpg", "notes/123/photo.jpg", "image/jpeg", 1024, 800, 600)
		err := repo.Create(ctx, photo)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, photo.ID)

		require.NoError(t, err)
		assert.Equal(t, photo.ID, found.ID)
		assert.Equal(t, "http://storage/photo.jpg", found.URL)
		assert.Equal(t, int64(1024), found.Size)
	})

	t.Run("returns not found error", func(t *testing.T) {
		db.Truncate(t, "photos", "notes", "users")

		found, err := repo.GetByID(ctx, uuid.New())

		assert.Nil(t, found)
		assert.ErrorIs(t, err, domain.ErrPhotoNotFound)
	})
}

func TestIntegrationPhotoRepo_GetByNoteID(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewPhotoRepo(db.Pool)
	ctx := context.Background()

	t.Run("returns all photos for note", func(t *testing.T) {
		db.Truncate(t, "photos", "notes", "users")
		_, note := createTestUserAndNote(t, db)

		photo1 := entity.NewPhoto(note.ID, "http://storage/photo1.jpg", "notes/123/photo1.jpg", "image/jpeg", 1024, 800, 600)
		err := repo.Create(ctx, photo1)
		require.NoError(t, err)

		photo2 := entity.NewPhoto(note.ID, "http://storage/photo2.jpg", "notes/123/photo2.jpg", "image/jpeg", 2048, 1920, 1080)
		err = repo.Create(ctx, photo2)
		require.NoError(t, err)

		photos, err := repo.GetByNoteID(ctx, note.ID)

		require.NoError(t, err)
		assert.Len(t, photos, 2)
	})

	t.Run("returns empty slice for note without photos", func(t *testing.T) {
		db.Truncate(t, "photos", "notes", "users")
		_, note := createTestUserAndNote(t, db)

		photos, err := repo.GetByNoteID(ctx, note.ID)

		require.NoError(t, err)
		assert.Empty(t, photos)
	})

	t.Run("does not return photos from other notes", func(t *testing.T) {
		db.Truncate(t, "photos", "notes", "users")
		userRepo := postgres.NewUserRepo(db.Pool)
		noteRepo := postgres.NewNoteRepo(db.Pool)

		user := entity.NewUser("test@example.com", "hashedpassword", "Test User")
		err := userRepo.Create(ctx, user)
		require.NoError(t, err)

		note1 := entity.NewNote(user.ID, "Note 1", "Content", nil, "n1")
		err = noteRepo.Create(ctx, note1)
		require.NoError(t, err)

		note2 := entity.NewNote(user.ID, "Note 2", "Content", nil, "n2")
		err = noteRepo.Create(ctx, note2)
		require.NoError(t, err)

		photo := entity.NewPhoto(note1.ID, "http://storage/photo.jpg", "notes/123/photo.jpg", "image/jpeg", 1024, 800, 600)
		err = repo.Create(ctx, photo)
		require.NoError(t, err)

		photos, err := repo.GetByNoteID(ctx, note2.ID)

		require.NoError(t, err)
		assert.Empty(t, photos)
	})
}

func TestIntegrationPhotoRepo_Delete(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewPhotoRepo(db.Pool)
	ctx := context.Background()

	t.Run("deletes photo successfully", func(t *testing.T) {
		db.Truncate(t, "photos", "notes", "users")
		_, note := createTestUserAndNote(t, db)

		photo := entity.NewPhoto(note.ID, "http://storage/photo.jpg", "notes/123/photo.jpg", "image/jpeg", 1024, 800, 600)
		err := repo.Create(ctx, photo)
		require.NoError(t, err)

		err = repo.Delete(ctx, photo.ID)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, photo.ID)
		assert.Nil(t, found)
		assert.ErrorIs(t, err, domain.ErrPhotoNotFound)
	})
}
