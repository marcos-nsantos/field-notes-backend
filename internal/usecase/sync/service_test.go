package sync_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/marcos-nsantos/field-notes-backend/internal/domain/entity"
	"github.com/marcos-nsantos/field-notes-backend/internal/mocks"
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/sync"
)

func TestService_BatchSync(t *testing.T) {
	ctx := context.Background()

	t.Run("syncs new notes from client", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		deviceRepo := mocks.NewMockDeviceRepository(ctrl)
		svc := sync.NewService(noteRepo, deviceRepo)

		userID := uuid.New()
		deviceID := uuid.New()
		device := &entity.Device{
			ID:         deviceID,
			UserID:     userID,
			DeviceID:   "device-123",
			SyncCursor: time.Now().Add(-1 * time.Hour),
		}

		deviceRepo.EXPECT().GetByUserAndDeviceID(ctx, userID, "device-123").Return(device, nil)
		noteRepo.EXPECT().GetModifiedSince(ctx, userID, gomock.Any(), 1000).Return([]entity.Note{}, nil)
		noteRepo.EXPECT().BatchUpsert(ctx, gomock.Any()).Return(nil)
		deviceRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

		result, err := svc.BatchSync(ctx, sync.SyncInput{
			UserID:   userID,
			DeviceID: "device-123",
			ClientNotes: []sync.ClientNote{
				{
					ClientID:  "note-1",
					Title:     "New Note",
					Content:   "Content",
					UpdatedAt: time.Now(),
				},
			},
		})

		require.NoError(t, err)
		assert.Empty(t, result.ServerNotes)
		assert.Empty(t, result.Conflicts)
	})

	t.Run("returns server notes since cursor", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		deviceRepo := mocks.NewMockDeviceRepository(ctrl)
		svc := sync.NewService(noteRepo, deviceRepo)

		userID := uuid.New()
		deviceID := uuid.New()
		syncCursor := time.Now().Add(-1 * time.Hour)
		device := &entity.Device{
			ID:         deviceID,
			UserID:     userID,
			DeviceID:   "device-123",
			SyncCursor: syncCursor,
		}

		serverNotes := []entity.Note{
			{ID: uuid.New(), UserID: userID, Title: "Server Note", ClientID: "server-note-1"},
		}

		deviceRepo.EXPECT().GetByUserAndDeviceID(ctx, userID, "device-123").Return(device, nil)
		noteRepo.EXPECT().GetModifiedSince(ctx, userID, syncCursor, 1000).Return(serverNotes, nil)
		deviceRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

		result, err := svc.BatchSync(ctx, sync.SyncInput{
			UserID:      userID,
			DeviceID:    "device-123",
			ClientNotes: []sync.ClientNote{},
		})

		require.NoError(t, err)
		assert.Len(t, result.ServerNotes, 1)
		assert.Equal(t, "Server Note", result.ServerNotes[0].Title)
	})

	t.Run("client wins conflict when more recent", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		deviceRepo := mocks.NewMockDeviceRepository(ctrl)
		svc := sync.NewService(noteRepo, deviceRepo)

		userID := uuid.New()
		deviceID := uuid.New()
		noteID := uuid.New()
		serverTime := time.Now().Add(-1 * time.Hour)
		clientTime := time.Now()

		device := &entity.Device{
			ID:         deviceID,
			UserID:     userID,
			DeviceID:   "device-123",
			SyncCursor: time.Now().Add(-2 * time.Hour),
		}

		serverNote := entity.Note{
			ID:        noteID,
			UserID:    userID,
			Title:     "Server Version",
			ClientID:  "conflict-note",
			UpdatedAt: serverTime,
		}

		deviceRepo.EXPECT().GetByUserAndDeviceID(ctx, userID, "device-123").Return(device, nil)
		noteRepo.EXPECT().GetModifiedSince(ctx, userID, gomock.Any(), 1000).Return([]entity.Note{serverNote}, nil)
		noteRepo.EXPECT().BatchUpsert(ctx, gomock.Any()).Return(nil)
		deviceRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

		result, err := svc.BatchSync(ctx, sync.SyncInput{
			UserID:   userID,
			DeviceID: "device-123",
			ClientNotes: []sync.ClientNote{
				{
					ClientID:  "conflict-note",
					Title:     "Client Version",
					Content:   "Updated by client",
					UpdatedAt: clientTime, // More recent
				},
			},
		})

		require.NoError(t, err)
		assert.Len(t, result.Conflicts, 1)
		assert.Equal(t, "client_wins", result.Conflicts[0].Resolution)
		assert.Equal(t, "conflict-note", result.Conflicts[0].ClientID)
	})

	t.Run("server wins conflict when more recent", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		deviceRepo := mocks.NewMockDeviceRepository(ctrl)
		svc := sync.NewService(noteRepo, deviceRepo)

		userID := uuid.New()
		deviceID := uuid.New()
		noteID := uuid.New()
		serverTime := time.Now()
		clientTime := time.Now().Add(-1 * time.Hour)

		device := &entity.Device{
			ID:         deviceID,
			UserID:     userID,
			DeviceID:   "device-123",
			SyncCursor: time.Now().Add(-2 * time.Hour),
		}

		serverNote := entity.Note{
			ID:        noteID,
			UserID:    userID,
			Title:     "Server Version",
			ClientID:  "conflict-note",
			UpdatedAt: serverTime, // More recent
		}

		deviceRepo.EXPECT().GetByUserAndDeviceID(ctx, userID, "device-123").Return(device, nil)
		noteRepo.EXPECT().GetModifiedSince(ctx, userID, gomock.Any(), 1000).Return([]entity.Note{serverNote}, nil)
		deviceRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

		result, err := svc.BatchSync(ctx, sync.SyncInput{
			UserID:   userID,
			DeviceID: "device-123",
			ClientNotes: []sync.ClientNote{
				{
					ClientID:  "conflict-note",
					Title:     "Client Version",
					Content:   "Updated by client",
					UpdatedAt: clientTime, // Older
				},
			},
		})

		require.NoError(t, err)
		assert.Len(t, result.Conflicts, 1)
		assert.Equal(t, "server_wins", result.Conflicts[0].Resolution)
	})

	t.Run("handles deleted notes from client", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		deviceRepo := mocks.NewMockDeviceRepository(ctrl)
		svc := sync.NewService(noteRepo, deviceRepo)

		userID := uuid.New()
		deviceID := uuid.New()
		device := &entity.Device{
			ID:         deviceID,
			UserID:     userID,
			DeviceID:   "device-123",
			SyncCursor: time.Now().Add(-1 * time.Hour),
		}

		deviceRepo.EXPECT().GetByUserAndDeviceID(ctx, userID, "device-123").Return(device, nil)
		noteRepo.EXPECT().GetModifiedSince(ctx, userID, gomock.Any(), 1000).Return([]entity.Note{}, nil)
		noteRepo.EXPECT().BatchUpsert(ctx, gomock.AssignableToTypeOf([]entity.Note{})).DoAndReturn(
			func(ctx context.Context, notes []entity.Note) error {
				assert.Len(t, notes, 1)
				assert.NotNil(t, notes[0].DeletedAt)
				return nil
			})
		deviceRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

		result, err := svc.BatchSync(ctx, sync.SyncInput{
			UserID:   userID,
			DeviceID: "device-123",
			ClientNotes: []sync.ClientNote{
				{
					ClientID:  "deleted-note",
					Title:     "Deleted Note",
					Content:   "Content",
					UpdatedAt: time.Now(),
					IsDeleted: true,
				},
			},
		})

		require.NoError(t, err)
		assert.Empty(t, result.Conflicts)
	})

	t.Run("updates device sync cursor", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteRepo := mocks.NewMockNoteRepository(ctrl)
		deviceRepo := mocks.NewMockDeviceRepository(ctrl)
		svc := sync.NewService(noteRepo, deviceRepo)

		userID := uuid.New()
		deviceID := uuid.New()
		oldCursor := time.Now().Add(-1 * time.Hour)
		device := &entity.Device{
			ID:         deviceID,
			UserID:     userID,
			DeviceID:   "device-123",
			SyncCursor: oldCursor,
		}

		deviceRepo.EXPECT().GetByUserAndDeviceID(ctx, userID, "device-123").Return(device, nil)
		noteRepo.EXPECT().GetModifiedSince(ctx, userID, oldCursor, 1000).Return([]entity.Note{}, nil)
		deviceRepo.EXPECT().Update(ctx, gomock.AssignableToTypeOf(&entity.Device{})).DoAndReturn(
			func(ctx context.Context, d *entity.Device) error {
				assert.True(t, d.SyncCursor.After(oldCursor))
				return nil
			})

		result, err := svc.BatchSync(ctx, sync.SyncInput{
			UserID:      userID,
			DeviceID:    "device-123",
			ClientNotes: []sync.ClientNote{},
		})

		require.NoError(t, err)
		assert.True(t, result.NewCursor.After(oldCursor))
	})
}
