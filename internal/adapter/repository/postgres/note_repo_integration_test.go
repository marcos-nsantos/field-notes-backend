package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/repository"
	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/repository/postgres"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/entity"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/valueobject"
	"github.com/marcos-nsantos/field-notes-backend/internal/pkg/pagination"
)

func createTestUser(t *testing.T, db *TestDB) *entity.User {
	t.Helper()
	repo := postgres.NewUserRepo(db.Pool)
	user := entity.NewUser("test@example.com", "hashedpassword", "Test User")
	err := repo.Create(context.Background(), user)
	require.NoError(t, err)
	return user
}

func TestIntegrationNoteRepo_Create(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewNoteRepo(db.Pool)
	ctx := context.Background()

	t.Run("creates note successfully", func(t *testing.T) {
		db.Truncate(t, "notes", "users")
		user := createTestUser(t, db)

		loc := valueobject.NewLocation(37.7749, -122.4194, nil, nil)
		note := entity.NewNote(user.ID, "Test Note", "Test content", loc, "client-123")
		err := repo.Create(ctx, note)

		require.NoError(t, err)
		assert.NotEmpty(t, note.ID)
	})

	t.Run("creates note with location", func(t *testing.T) {
		db.Truncate(t, "notes", "users")
		user := createTestUser(t, db)

		altitude := 100.0
		accuracy := 10.0
		loc := valueobject.NewLocation(37.7749, -122.4194, &altitude, &accuracy)
		note := entity.NewNote(user.ID, "Note with Location", "Content", loc, "loc-123")
		err := repo.Create(ctx, note)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, note.ID)
		require.NoError(t, err)
		assert.Equal(t, 37.7749, found.Location.Latitude)
		assert.Equal(t, -122.4194, found.Location.Longitude)
	})
}

func TestIntegrationNoteRepo_GetByID(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewNoteRepo(db.Pool)
	ctx := context.Background()

	t.Run("returns note by ID", func(t *testing.T) {
		db.Truncate(t, "notes", "users")
		user := createTestUser(t, db)

		note := entity.NewNote(user.ID, "Test Note", "Test content", nil, "client-123")
		err := repo.Create(ctx, note)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, note.ID)

		require.NoError(t, err)
		assert.Equal(t, note.ID, found.ID)
		assert.Equal(t, "Test Note", found.Title)
	})

	t.Run("returns not found error", func(t *testing.T) {
		db.Truncate(t, "notes", "users")

		found, err := repo.GetByID(ctx, uuid.New())

		assert.Nil(t, found)
		assert.ErrorIs(t, err, domain.ErrNoteNotFound)
	})
}

func TestIntegrationNoteRepo_GetByClientID(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewNoteRepo(db.Pool)
	ctx := context.Background()

	t.Run("returns note by client ID", func(t *testing.T) {
		db.Truncate(t, "notes", "users")
		user := createTestUser(t, db)

		note := entity.NewNote(user.ID, "Test Note", "Test content", nil, "unique-client-id")
		err := repo.Create(ctx, note)
		require.NoError(t, err)

		found, err := repo.GetByClientID(ctx, user.ID, "unique-client-id")

		require.NoError(t, err)
		assert.Equal(t, note.ID, found.ID)
	})

	t.Run("returns not found for wrong user", func(t *testing.T) {
		db.Truncate(t, "notes", "users")
		user := createTestUser(t, db)

		note := entity.NewNote(user.ID, "Test Note", "Test content", nil, "client-id")
		err := repo.Create(ctx, note)
		require.NoError(t, err)

		found, err := repo.GetByClientID(ctx, uuid.New(), "client-id")

		assert.Nil(t, found)
		assert.ErrorIs(t, err, domain.ErrNoteNotFound)
	})
}

func TestIntegrationNoteRepo_List(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewNoteRepo(db.Pool)
	ctx := context.Background()

	t.Run("lists notes with pagination", func(t *testing.T) {
		db.Truncate(t, "notes", "users")
		user := createTestUser(t, db)

		for i := 0; i < 25; i++ {
			note := entity.NewNote(user.ID, "Note", "Content", nil, "")
			err := repo.Create(ctx, note)
			require.NoError(t, err)
		}

		notes, info, err := repo.List(ctx, user.ID, repository.NoteListParams{
			Pagination: pagination.Params{Page: 1, PerPage: 10},
		})

		require.NoError(t, err)
		assert.Len(t, notes, 10)
		assert.Equal(t, 25, info.TotalItems)
		assert.Equal(t, 3, info.TotalPages)
	})

	t.Run("excludes deleted notes", func(t *testing.T) {
		db.Truncate(t, "notes", "users")
		user := createTestUser(t, db)

		note1 := entity.NewNote(user.ID, "Active Note", "Content", nil, "")
		err := repo.Create(ctx, note1)
		require.NoError(t, err)

		note2 := entity.NewNote(user.ID, "Deleted Note", "Content", nil, "")
		err = repo.Create(ctx, note2)
		require.NoError(t, err)
		err = repo.SoftDelete(ctx, note2.ID)
		require.NoError(t, err)

		notes, info, err := repo.List(ctx, user.ID, repository.NoteListParams{
			Pagination: pagination.Params{Page: 1, PerPage: 10},
		})

		require.NoError(t, err)
		assert.Len(t, notes, 1)
		assert.Equal(t, 1, info.TotalItems)
		assert.Equal(t, "Active Note", notes[0].Title)
	})

	t.Run("filters by bounding box", func(t *testing.T) {
		db.Truncate(t, "notes", "users")
		user := createTestUser(t, db)

		// San Francisco
		sfLoc := valueobject.NewLocation(37.7749, -122.4194, nil, nil)
		sfNote := entity.NewNote(user.ID, "SF Note", "Content", sfLoc, "sf-1")
		err := repo.Create(ctx, sfNote)
		require.NoError(t, err)

		// New York
		nyLoc := valueobject.NewLocation(40.7128, -74.0060, nil, nil)
		nyNote := entity.NewNote(user.ID, "NY Note", "Content", nyLoc, "ny-1")
		err = repo.Create(ctx, nyNote)
		require.NoError(t, err)

		// Query for Bay Area only
		bbox := valueobject.NewBoundingBox(37.0, 38.5, -123.0, -122.0)
		notes, _, err := repo.List(ctx, user.ID, repository.NoteListParams{
			Pagination:  pagination.Params{Page: 1, PerPage: 10},
			BoundingBox: bbox,
		})

		require.NoError(t, err)
		assert.Len(t, notes, 1)
		assert.Equal(t, "SF Note", notes[0].Title)
	})
}

func TestIntegrationNoteRepo_Update(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewNoteRepo(db.Pool)
	ctx := context.Background()

	t.Run("updates note successfully", func(t *testing.T) {
		db.Truncate(t, "notes", "users")
		user := createTestUser(t, db)

		note := entity.NewNote(user.ID, "Original Title", "Original Content", nil, "")
		err := repo.Create(ctx, note)
		require.NoError(t, err)

		note.Title = "Updated Title"
		note.Content = "Updated Content"
		err = repo.Update(ctx, note)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, note.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Title", found.Title)
		assert.Equal(t, "Updated Content", found.Content)
	})
}

func TestIntegrationNoteRepo_SoftDelete(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewNoteRepo(db.Pool)
	ctx := context.Background()

	t.Run("soft deletes note", func(t *testing.T) {
		db.Truncate(t, "notes", "users")
		user := createTestUser(t, db)

		note := entity.NewNote(user.ID, "Test Note", "Content", nil, "")
		err := repo.Create(ctx, note)
		require.NoError(t, err)

		err = repo.SoftDelete(ctx, note.ID)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, note.ID)
		require.NoError(t, err)
		assert.NotNil(t, found.DeletedAt)
	})
}

func TestIntegrationNoteRepo_GetModifiedSince(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewNoteRepo(db.Pool)
	ctx := context.Background()

	t.Run("returns notes modified since timestamp", func(t *testing.T) {
		db.Truncate(t, "notes", "users")
		user := createTestUser(t, db)

		since := time.Now().Add(-1 * time.Hour)

		note := entity.NewNote(user.ID, "Recent Note", "Content", nil, "recent-1")
		err := repo.Create(ctx, note)
		require.NoError(t, err)

		notes, err := repo.GetModifiedSince(ctx, user.ID, since, 100)

		require.NoError(t, err)
		assert.Len(t, notes, 1)
		assert.Equal(t, "Recent Note", notes[0].Title)
	})

	t.Run("includes deleted notes for sync", func(t *testing.T) {
		db.Truncate(t, "notes", "users")
		user := createTestUser(t, db)

		since := time.Now().Add(-1 * time.Hour)

		note := entity.NewNote(user.ID, "Deleted Note", "Content", nil, "deleted-1")
		err := repo.Create(ctx, note)
		require.NoError(t, err)
		err = repo.SoftDelete(ctx, note.ID)
		require.NoError(t, err)

		notes, err := repo.GetModifiedSince(ctx, user.ID, since, 100)

		require.NoError(t, err)
		assert.Len(t, notes, 1)
		assert.NotNil(t, notes[0].DeletedAt)
	})
}

func TestIntegrationNoteRepo_BatchUpsert(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewNoteRepo(db.Pool)
	ctx := context.Background()

	t.Run("inserts new notes", func(t *testing.T) {
		db.Truncate(t, "notes", "users")
		user := createTestUser(t, db)

		notes := []entity.Note{
			*entity.NewNote(user.ID, "Note 1", "Content 1", nil, "batch-1"),
			*entity.NewNote(user.ID, "Note 2", "Content 2", nil, "batch-2"),
		}

		err := repo.BatchUpsert(ctx, notes)
		require.NoError(t, err)

		found1, err := repo.GetByClientID(ctx, user.ID, "batch-1")
		require.NoError(t, err)
		assert.Equal(t, "Note 1", found1.Title)

		found2, err := repo.GetByClientID(ctx, user.ID, "batch-2")
		require.NoError(t, err)
		assert.Equal(t, "Note 2", found2.Title)
	})

	t.Run("updates existing notes with newer timestamp", func(t *testing.T) {
		db.Truncate(t, "notes", "users")
		user := createTestUser(t, db)

		// Create initial note
		note := entity.NewNote(user.ID, "Original", "Original Content", nil, "upsert-1")
		err := repo.Create(ctx, note)
		require.NoError(t, err)

		// Upsert with newer timestamp
		updatedNote := *entity.NewNote(user.ID, "Updated", "Updated Content", nil, "upsert-1")
		updatedNote.UpdatedAt = time.Now().Add(1 * time.Hour)

		err = repo.BatchUpsert(ctx, []entity.Note{updatedNote})
		require.NoError(t, err)

		found, err := repo.GetByClientID(ctx, user.ID, "upsert-1")
		require.NoError(t, err)
		assert.Equal(t, "Updated", found.Title)
	})
}
