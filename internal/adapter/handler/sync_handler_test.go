package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/handler"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/entity"
	"github.com/marcos-nsantos/field-notes-backend/internal/mocks"
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/sync"
)

func TestSyncHandler_Sync(t *testing.T) {
	t.Run("syncs successfully with no conflicts", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		syncSvc := mocks.NewMockSyncService(ctrl)
		h := handler.NewSyncHandler(syncSvc)

		router := setupRouter()
		userID := uuid.New()
		router.POST("/sync", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Sync(c)
		})

		serverNotes := []entity.Note{
			{
				ID:        uuid.New(),
				UserID:    userID,
				Title:     "Server Note",
				Content:   "From server",
				ClientID:  "server-client-1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}

		result := &sync.SyncResult{
			ServerNotes: serverNotes,
			NewCursor:   time.Now().UTC(),
			Conflicts:   []sync.ConflictInfo{},
		}

		syncSvc.EXPECT().BatchSync(gomock.Any(), gomock.Any()).Return(result, nil)

		body := `{
			"device_id": "device-123",
			"notes": [
				{
					"client_id": "client-note-1",
					"title": "Client Note",
					"content": "From client",
					"updated_at": "2024-01-15T10:00:00Z"
				}
			]
		}`
		req := httptest.NewRequest(http.MethodPost, "/sync", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp["server_notes"])
		assert.NotNil(t, resp["new_cursor"])
	})

	t.Run("syncs with conflicts - client wins", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		syncSvc := mocks.NewMockSyncService(ctrl)
		h := handler.NewSyncHandler(syncSvc)

		router := setupRouter()
		userID := uuid.New()
		router.POST("/sync", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Sync(c)
		})

		serverNote := &entity.Note{
			ID:        uuid.New(),
			UserID:    userID,
			Title:     "Server Version",
			Content:   "Server content",
			ClientID:  "conflict-note",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now().Add(-1 * time.Hour),
		}

		result := &sync.SyncResult{
			ServerNotes: []entity.Note{},
			NewCursor:   time.Now().UTC(),
			Conflicts: []sync.ConflictInfo{
				{
					ClientID:      "conflict-note",
					Resolution:    "client_wins",
					ServerVersion: serverNote,
				},
			},
		}

		syncSvc.EXPECT().BatchSync(gomock.Any(), gomock.Any()).Return(result, nil)

		body := `{
			"device_id": "device-123",
			"notes": [
				{
					"client_id": "conflict-note",
					"title": "Client Version",
					"content": "Client content",
					"updated_at": "2024-01-15T12:00:00Z"
				}
			]
		}`
		req := httptest.NewRequest(http.MethodPost, "/sync", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		conflicts := resp["conflicts"].([]any)
		assert.Len(t, conflicts, 1)
		conflict := conflicts[0].(map[string]any)
		assert.Equal(t, "client_wins", conflict["resolution"])
	})

	t.Run("syncs with cursor", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		syncSvc := mocks.NewMockSyncService(ctrl)
		h := handler.NewSyncHandler(syncSvc)

		router := setupRouter()
		userID := uuid.New()
		router.POST("/sync", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Sync(c)
		})

		result := &sync.SyncResult{
			ServerNotes: []entity.Note{},
			NewCursor:   time.Now().UTC(),
			Conflicts:   []sync.ConflictInfo{},
		}

		syncSvc.EXPECT().BatchSync(gomock.Any(), gomock.Any()).Return(result, nil)

		body := `{
			"device_id": "device-123",
			"sync_cursor": "2024-01-01T00:00:00Z",
			"notes": []
		}`
		req := httptest.NewRequest(http.MethodPost, "/sync", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns error for unregistered device", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		syncSvc := mocks.NewMockSyncService(ctrl)
		h := handler.NewSyncHandler(syncSvc)

		router := setupRouter()
		userID := uuid.New()
		router.POST("/sync", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Sync(c)
		})

		syncSvc.EXPECT().BatchSync(gomock.Any(), gomock.Any()).Return(nil, domain.ErrDeviceNotFound)

		body := `{
			"device_id": "unknown-device",
			"notes": []
		}`
		req := httptest.NewRequest(http.MethodPost, "/sync", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "DEVICE_NOT_FOUND", resp["code"])
	})

	t.Run("returns validation error for missing device_id", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		syncSvc := mocks.NewMockSyncService(ctrl)
		h := handler.NewSyncHandler(syncSvc)

		router := setupRouter()
		userID := uuid.New()
		router.POST("/sync", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Sync(c)
		})

		body := `{"notes": []}`
		req := httptest.NewRequest(http.MethodPost, "/sync", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("syncs with deleted note", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		syncSvc := mocks.NewMockSyncService(ctrl)
		h := handler.NewSyncHandler(syncSvc)

		router := setupRouter()
		userID := uuid.New()
		router.POST("/sync", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Sync(c)
		})

		result := &sync.SyncResult{
			ServerNotes: []entity.Note{},
			NewCursor:   time.Now().UTC(),
			Conflicts:   []sync.ConflictInfo{},
		}

		syncSvc.EXPECT().BatchSync(gomock.Any(), gomock.Any()).Return(result, nil)

		body := `{
			"device_id": "device-123",
			"notes": [
				{
					"client_id": "deleted-note",
					"title": "Deleted Note",
					"content": "This note was deleted",
					"updated_at": "2024-01-15T10:00:00Z",
					"is_deleted": true
				}
			]
		}`
		req := httptest.NewRequest(http.MethodPost, "/sync", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
