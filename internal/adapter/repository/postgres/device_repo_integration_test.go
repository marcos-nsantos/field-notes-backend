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

func TestIntegrationDeviceRepo_Create(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewDeviceRepo(db.Pool)
	ctx := context.Background()

	t.Run("creates device successfully", func(t *testing.T) {
		db.Truncate(t, "devices", "users")
		user := createTestUser(t, db)

		device := entity.NewDevice(user.ID, "device-123", "ios", "iPhone 15")
		err := repo.Create(ctx, device)

		require.NoError(t, err)
		assert.NotEmpty(t, device.ID)
	})

	t.Run("fails with duplicate device_id for same user", func(t *testing.T) {
		db.Truncate(t, "devices", "users")
		user := createTestUser(t, db)

		device1 := entity.NewDevice(user.ID, "same-device", "ios", "Device 1")
		err := repo.Create(ctx, device1)
		require.NoError(t, err)

		device2 := entity.NewDevice(user.ID, "same-device", "ios", "Device 2")
		err = repo.Create(ctx, device2)

		assert.Error(t, err)
	})
}

func TestIntegrationDeviceRepo_GetByID(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewDeviceRepo(db.Pool)
	ctx := context.Background()

	t.Run("returns device by ID", func(t *testing.T) {
		db.Truncate(t, "devices", "users")
		user := createTestUser(t, db)

		device := entity.NewDevice(user.ID, "device-123", "ios", "iPhone 15")
		err := repo.Create(ctx, device)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, device.ID)

		require.NoError(t, err)
		assert.Equal(t, device.ID, found.ID)
		assert.Equal(t, "device-123", found.DeviceID)
		assert.Equal(t, "ios", found.Platform)
	})

	t.Run("returns not found error", func(t *testing.T) {
		db.Truncate(t, "devices", "users")

		found, err := repo.GetByID(ctx, uuid.New())

		assert.Nil(t, found)
		assert.ErrorIs(t, err, domain.ErrDeviceNotFound)
	})
}

func TestIntegrationDeviceRepo_GetByUserAndDeviceID(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewDeviceRepo(db.Pool)
	ctx := context.Background()

	t.Run("returns device by user and device ID", func(t *testing.T) {
		db.Truncate(t, "devices", "users")
		user := createTestUser(t, db)

		device := entity.NewDevice(user.ID, "my-device", "android", "Pixel 8")
		err := repo.Create(ctx, device)
		require.NoError(t, err)

		found, err := repo.GetByUserAndDeviceID(ctx, user.ID, "my-device")

		require.NoError(t, err)
		assert.Equal(t, device.ID, found.ID)
	})

	t.Run("returns not found for wrong user", func(t *testing.T) {
		db.Truncate(t, "devices", "users")
		user := createTestUser(t, db)

		device := entity.NewDevice(user.ID, "my-device", "android", "Pixel 8")
		err := repo.Create(ctx, device)
		require.NoError(t, err)

		found, err := repo.GetByUserAndDeviceID(ctx, uuid.New(), "my-device")

		assert.Nil(t, found)
		assert.ErrorIs(t, err, domain.ErrDeviceNotFound)
	})
}

func TestIntegrationDeviceRepo_Update(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewDeviceRepo(db.Pool)
	ctx := context.Background()

	t.Run("updates device sync cursor", func(t *testing.T) {
		db.Truncate(t, "devices", "users")
		user := createTestUser(t, db)

		device := entity.NewDevice(user.ID, "device-123", "ios", "iPhone 15")
		err := repo.Create(ctx, device)
		require.NoError(t, err)

		newCursor := time.Now()
		device.SyncCursor = newCursor
		err = repo.Update(ctx, device)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, device.ID)
		require.NoError(t, err)
		assert.WithinDuration(t, newCursor, found.SyncCursor, time.Second)
	})
}

func TestIntegrationDeviceRepo_Upsert(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Cleanup(t)

	repo := postgres.NewDeviceRepo(db.Pool)
	ctx := context.Background()

	t.Run("creates device if not exists", func(t *testing.T) {
		db.Truncate(t, "devices", "users")
		user := createTestUser(t, db)

		device := entity.NewDevice(user.ID, "new-device", "ios", "iPhone 15")
		err := repo.Upsert(ctx, device)
		require.NoError(t, err)

		found, err := repo.GetByUserAndDeviceID(ctx, user.ID, "new-device")
		require.NoError(t, err)
		assert.Equal(t, "iPhone 15", found.Name)
	})

	t.Run("updates device if exists", func(t *testing.T) {
		db.Truncate(t, "devices", "users")
		user := createTestUser(t, db)

		device := entity.NewDevice(user.ID, "existing-device", "ios", "iPhone 14")
		err := repo.Create(ctx, device)
		require.NoError(t, err)

		updatedDevice := entity.NewDevice(user.ID, "existing-device", "ios", "iPhone 15 Pro")
		err = repo.Upsert(ctx, updatedDevice)
		require.NoError(t, err)

		found, err := repo.GetByUserAndDeviceID(ctx, user.ID, "existing-device")
		require.NoError(t, err)
		assert.Equal(t, "iPhone 15 Pro", found.Name)
	})
}
