package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/repository/postgres"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/entity"
)

func TestIntegrationUserRepo_Create(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewUserRepo(db.Pool)
	ctx := context.Background()

	t.Run("creates user successfully", func(t *testing.T) {
		db.Truncate(t, "users")

		user := entity.NewUser("test@example.com", "hashedpassword", "Test User")
		err := repo.Create(ctx, user)

		require.NoError(t, err)
		assert.NotEmpty(t, user.ID)
	})

	t.Run("fails with duplicate email", func(t *testing.T) {
		db.Truncate(t, "users")

		user1 := entity.NewUser("duplicate@example.com", "hashedpassword", "User 1")
		err := repo.Create(ctx, user1)
		require.NoError(t, err)

		user2 := entity.NewUser("duplicate@example.com", "hashedpassword", "User 2")
		err = repo.Create(ctx, user2)

		assert.Error(t, err)
	})
}

func TestIntegrationUserRepo_GetByID(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewUserRepo(db.Pool)
	ctx := context.Background()

	t.Run("returns user by ID", func(t *testing.T) {
		db.Truncate(t, "users")

		user := entity.NewUser("test@example.com", "hashedpassword", "Test User")
		err := repo.Create(ctx, user)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, user.ID)

		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
		assert.Equal(t, "test@example.com", found.Email)
		assert.Equal(t, "Test User", found.Name)
	})

	t.Run("returns not found error", func(t *testing.T) {
		db.Truncate(t, "users")

		user := entity.NewUser("test@example.com", "hashedpassword", "Test User")

		found, err := repo.GetByID(ctx, user.ID)

		assert.Nil(t, found)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

func TestIntegrationUserRepo_GetByEmail(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewUserRepo(db.Pool)
	ctx := context.Background()

	t.Run("returns user by email", func(t *testing.T) {
		db.Truncate(t, "users")

		user := entity.NewUser("test@example.com", "hashedpassword", "Test User")
		err := repo.Create(ctx, user)
		require.NoError(t, err)

		found, err := repo.GetByEmail(ctx, "test@example.com")

		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
		assert.Equal(t, "test@example.com", found.Email)
	})

	t.Run("returns not found error", func(t *testing.T) {
		db.Truncate(t, "users")

		found, err := repo.GetByEmail(ctx, "notfound@example.com")

		assert.Nil(t, found)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

func TestIntegrationUserRepo_ExistsByEmail(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewUserRepo(db.Pool)
	ctx := context.Background()

	t.Run("returns true if email exists", func(t *testing.T) {
		db.Truncate(t, "users")

		user := entity.NewUser("exists@example.com", "hashedpassword", "Test User")
		err := repo.Create(ctx, user)
		require.NoError(t, err)

		exists, err := repo.ExistsByEmail(ctx, "exists@example.com")

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false if email does not exist", func(t *testing.T) {
		db.Truncate(t, "users")

		exists, err := repo.ExistsByEmail(ctx, "notexists@example.com")

		require.NoError(t, err)
		assert.False(t, exists)
	})
}
