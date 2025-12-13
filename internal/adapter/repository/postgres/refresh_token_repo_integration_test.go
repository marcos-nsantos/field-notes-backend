package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/repository/postgres"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/entity"
)

func createTestUserAndDevice(t *testing.T, db *TestDB) (*entity.User, *entity.Device) {
	t.Helper()
	userRepo := postgres.NewUserRepo(db.Pool)
	deviceRepo := postgres.NewDeviceRepo(db.Pool)
	ctx := context.Background()

	user := entity.NewUser("test@example.com", "hashedpassword", "Test User")
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	device := entity.NewDevice(user.ID, "device-123", "ios", "iPhone 15")
	err = deviceRepo.Create(ctx, device)
	require.NoError(t, err)

	return user, device
}

func TestIntegrationRefreshTokenRepo_Create(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewRefreshTokenRepo(db.Pool)
	ctx := context.Background()

	t.Run("creates refresh token successfully", func(t *testing.T) {
		db.Truncate(t, "refresh_tokens", "devices", "users")
		user, device := createTestUserAndDevice(t, db)

		token := entity.NewRefreshToken(user.ID, device.ID, "test-token-123", time.Now().Add(24*time.Hour))
		err := repo.Create(ctx, token)

		require.NoError(t, err)
		assert.NotEmpty(t, token.ID)
	})
}

func TestIntegrationRefreshTokenRepo_GetByToken(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewRefreshTokenRepo(db.Pool)
	ctx := context.Background()

	t.Run("returns token by value", func(t *testing.T) {
		db.Truncate(t, "refresh_tokens", "devices", "users")
		user, device := createTestUserAndDevice(t, db)

		token := entity.NewRefreshToken(user.ID, device.ID, "unique-token", time.Now().Add(24*time.Hour))
		err := repo.Create(ctx, token)
		require.NoError(t, err)

		found, err := repo.GetByToken(ctx, "unique-token")

		require.NoError(t, err)
		assert.Equal(t, token.ID, found.ID)
		assert.Equal(t, user.ID, found.UserID)
		assert.Equal(t, device.ID, found.DeviceID)
	})

	t.Run("returns not found error", func(t *testing.T) {
		db.Truncate(t, "refresh_tokens", "devices", "users")

		found, err := repo.GetByToken(ctx, "non-existent-token")

		assert.Nil(t, found)
		assert.ErrorIs(t, err, domain.ErrTokenInvalid)
	})
}

func TestIntegrationRefreshTokenRepo_Revoke(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewRefreshTokenRepo(db.Pool)
	ctx := context.Background()

	t.Run("revokes token successfully", func(t *testing.T) {
		db.Truncate(t, "refresh_tokens", "devices", "users")
		user, device := createTestUserAndDevice(t, db)

		token := entity.NewRefreshToken(user.ID, device.ID, "to-revoke", time.Now().Add(24*time.Hour))
		err := repo.Create(ctx, token)
		require.NoError(t, err)

		err = repo.Revoke(ctx, token.ID)
		require.NoError(t, err)

		found, err := repo.GetByToken(ctx, "to-revoke")
		require.NoError(t, err)
		assert.NotNil(t, found.RevokedAt)
	})
}

func TestIntegrationRefreshTokenRepo_RevokeByUserID(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewRefreshTokenRepo(db.Pool)
	ctx := context.Background()

	t.Run("revokes all tokens for user", func(t *testing.T) {
		db.Truncate(t, "refresh_tokens", "devices", "users")
		user, device := createTestUserAndDevice(t, db)

		token1 := entity.NewRefreshToken(user.ID, device.ID, "token-1", time.Now().Add(24*time.Hour))
		err := repo.Create(ctx, token1)
		require.NoError(t, err)

		token2 := entity.NewRefreshToken(user.ID, device.ID, "token-2", time.Now().Add(24*time.Hour))
		err = repo.Create(ctx, token2)
		require.NoError(t, err)

		err = repo.RevokeByUserID(ctx, user.ID)
		require.NoError(t, err)

		found1, _ := repo.GetByToken(ctx, "token-1")
		found2, _ := repo.GetByToken(ctx, "token-2")

		assert.NotNil(t, found1.RevokedAt)
		assert.NotNil(t, found2.RevokedAt)
	})

	t.Run("does not affect other users tokens", func(t *testing.T) {
		db.Truncate(t, "refresh_tokens", "devices", "users")
		user1, device1 := createTestUserAndDevice(t, db)

		userRepo := postgres.NewUserRepo(db.Pool)
		deviceRepo := postgres.NewDeviceRepo(db.Pool)

		user2 := entity.NewUser("other@example.com", "hashedpassword", "Other User")
		err := userRepo.Create(ctx, user2)
		require.NoError(t, err)

		device2 := entity.NewDevice(user2.ID, "device-456", "android", "Pixel")
		err = deviceRepo.Create(ctx, device2)
		require.NoError(t, err)

		token1 := entity.NewRefreshToken(user1.ID, device1.ID, "user1-token", time.Now().Add(24*time.Hour))
		err = repo.Create(ctx, token1)
		require.NoError(t, err)

		token2 := entity.NewRefreshToken(user2.ID, device2.ID, "user2-token", time.Now().Add(24*time.Hour))
		err = repo.Create(ctx, token2)
		require.NoError(t, err)

		err = repo.RevokeByUserID(ctx, user1.ID)
		require.NoError(t, err)

		found1, _ := repo.GetByToken(ctx, "user1-token")
		found2, _ := repo.GetByToken(ctx, "user2-token")

		assert.NotNil(t, found1.RevokedAt)
		assert.Nil(t, found2.RevokedAt)
	})
}

func TestIntegrationRefreshTokenRepo_RevokeByDeviceID(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewRefreshTokenRepo(db.Pool)
	ctx := context.Background()

	t.Run("revokes all tokens for device", func(t *testing.T) {
		db.Truncate(t, "refresh_tokens", "devices", "users")
		user, device := createTestUserAndDevice(t, db)

		token1 := entity.NewRefreshToken(user.ID, device.ID, "device-token-1", time.Now().Add(24*time.Hour))
		err := repo.Create(ctx, token1)
		require.NoError(t, err)

		token2 := entity.NewRefreshToken(user.ID, device.ID, "device-token-2", time.Now().Add(24*time.Hour))
		err = repo.Create(ctx, token2)
		require.NoError(t, err)

		err = repo.RevokeByDeviceID(ctx, device.ID)
		require.NoError(t, err)

		found1, _ := repo.GetByToken(ctx, "device-token-1")
		found2, _ := repo.GetByToken(ctx, "device-token-2")

		assert.NotNil(t, found1.RevokedAt)
		assert.NotNil(t, found2.RevokedAt)
	})
}

func TestIntegrationRefreshTokenRepo_DeleteExpired(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewRefreshTokenRepo(db.Pool)
	ctx := context.Background()

	t.Run("deletes expired tokens", func(t *testing.T) {
		db.Truncate(t, "refresh_tokens", "devices", "users")
		user, device := createTestUserAndDevice(t, db)

		// Create expired token
		expiredToken := &entity.RefreshToken{
			ID:        uuid.New(),
			UserID:    user.ID,
			DeviceID:  device.ID,
			Token:     "expired-token",
			ExpiresAt: time.Now().Add(-24 * time.Hour),
			CreatedAt: time.Now().Add(-48 * time.Hour),
		}
		err := repo.Create(ctx, expiredToken)
		require.NoError(t, err)

		// Create valid token
		validToken := entity.NewRefreshToken(user.ID, device.ID, "valid-token", time.Now().Add(24*time.Hour))
		err = repo.Create(ctx, validToken)
		require.NoError(t, err)

		err = repo.DeleteExpired(ctx)
		require.NoError(t, err)

		// Expired token should be gone
		found, err := repo.GetByToken(ctx, "expired-token")
		assert.Nil(t, found)
		assert.ErrorIs(t, err, domain.ErrTokenInvalid)

		// Valid token should still exist
		found, err = repo.GetByToken(ctx, "valid-token")
		require.NoError(t, err)
		assert.NotNil(t, found)
	})
}
