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
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/valueobject"
	"github.com/marcos-nsantos/field-notes-backend/internal/mocks"
	"github.com/marcos-nsantos/field-notes-backend/internal/pkg/pagination"
)

func TestNoteHandler_Create(t *testing.T) {
	t.Run("creates note successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteSvc := mocks.NewMockNoteService(ctrl)
		h := handler.NewNoteHandler(noteSvc)

		router := setupRouter()
		userID := uuid.New()
		router.POST("/notes", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Create(c)
		})

		noteEntity := &entity.Note{
			ID:        uuid.New(),
			UserID:    userID,
			Title:     "Test Note",
			Content:   "Test content",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		noteSvc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(noteEntity, nil)

		body := `{"title":"Test Note","content":"Test content"}`
		req := httptest.NewRequest(http.MethodPost, "/notes", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Test Note", resp["title"])
		assert.Equal(t, "Test content", resp["content"])
	})

	t.Run("creates note with location", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteSvc := mocks.NewMockNoteService(ctrl)
		h := handler.NewNoteHandler(noteSvc)

		router := setupRouter()
		userID := uuid.New()
		router.POST("/notes", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Create(c)
		})

		loc := valueobject.NewLocation(40.7128, -74.0060, nil, nil)
		noteEntity := &entity.Note{
			ID:        uuid.New(),
			UserID:    userID,
			Title:     "NYC Note",
			Content:   "Content from NYC",
			Location:  loc,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		noteSvc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(noteEntity, nil)

		body := `{"title":"NYC Note","content":"Content from NYC","latitude":40.7128,"longitude":-74.0060}`
		req := httptest.NewRequest(http.MethodPost, "/notes", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "NYC Note", resp["title"])
		assert.NotNil(t, resp["location"])
	})

	t.Run("returns validation error for invalid input", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteSvc := mocks.NewMockNoteService(ctrl)
		h := handler.NewNoteHandler(noteSvc)

		router := setupRouter()
		userID := uuid.New()
		router.POST("/notes", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Create(c)
		})

		body := `{"title":"","content":""}`
		req := httptest.NewRequest(http.MethodPost, "/notes", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns error for invalid coordinates", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteSvc := mocks.NewMockNoteService(ctrl)
		h := handler.NewNoteHandler(noteSvc)

		router := setupRouter()
		userID := uuid.New()
		router.POST("/notes", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Create(c)
		})

		body := `{"title":"Test","content":"Test","latitude":999,"longitude":-74.0060}`
		req := httptest.NewRequest(http.MethodPost, "/notes", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestNoteHandler_List(t *testing.T) {
	t.Run("lists notes successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteSvc := mocks.NewMockNoteService(ctrl)
		h := handler.NewNoteHandler(noteSvc)

		router := setupRouter()
		userID := uuid.New()
		router.GET("/notes", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.List(c)
		})

		notes := []entity.Note{
			{
				ID:        uuid.New(),
				UserID:    userID,
				Title:     "Note 1",
				Content:   "Content 1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:        uuid.New(),
				UserID:    userID,
				Title:     "Note 2",
				Content:   "Content 2",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		pageInfo := &pagination.Info{
			Page:       1,
			PerPage:    20,
			TotalItems: 2,
			TotalPages: 1,
			HasNext:    false,
			HasPrev:    false,
		}

		noteSvc.EXPECT().List(gomock.Any(), gomock.Any()).Return(notes, pageInfo, nil)

		req := httptest.NewRequest(http.MethodGet, "/notes", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		notesResp := resp["notes"].([]any)
		assert.Len(t, notesResp, 2)
	})

	t.Run("lists notes with pagination", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteSvc := mocks.NewMockNoteService(ctrl)
		h := handler.NewNoteHandler(noteSvc)

		router := setupRouter()
		userID := uuid.New()
		router.GET("/notes", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.List(c)
		})

		notes := []entity.Note{}
		pageInfo := &pagination.Info{
			Page:       2,
			PerPage:    10,
			TotalItems: 25,
			TotalPages: 3,
			HasNext:    true,
			HasPrev:    true,
		}

		noteSvc.EXPECT().List(gomock.Any(), gomock.Any()).Return(notes, pageInfo, nil)

		req := httptest.NewRequest(http.MethodGet, "/notes?page=2&per_page=10", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		pag := resp["pagination"].(map[string]any)
		assert.Equal(t, float64(2), pag["page"])
		assert.Equal(t, float64(10), pag["per_page"])
	})
}

func TestNoteHandler_Get(t *testing.T) {
	t.Run("gets note successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteSvc := mocks.NewMockNoteService(ctrl)
		h := handler.NewNoteHandler(noteSvc)

		router := setupRouter()
		userID := uuid.New()
		noteID := uuid.New()
		router.GET("/notes/:id", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Get(c)
		})

		noteEntity := &entity.Note{
			ID:        noteID,
			UserID:    userID,
			Title:     "Test Note",
			Content:   "Test content",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		noteSvc.EXPECT().GetByID(gomock.Any(), userID, noteID).Return(noteEntity, nil)

		req := httptest.NewRequest(http.MethodGet, "/notes/"+noteID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Test Note", resp["title"])
	})

	t.Run("returns not found for non-existent note", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteSvc := mocks.NewMockNoteService(ctrl)
		h := handler.NewNoteHandler(noteSvc)

		router := setupRouter()
		userID := uuid.New()
		noteID := uuid.New()
		router.GET("/notes/:id", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Get(c)
		})

		noteSvc.EXPECT().GetByID(gomock.Any(), userID, noteID).Return(nil, domain.ErrNoteNotFound)

		req := httptest.NewRequest(http.MethodGet, "/notes/"+noteID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns forbidden for other user's note", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteSvc := mocks.NewMockNoteService(ctrl)
		h := handler.NewNoteHandler(noteSvc)

		router := setupRouter()
		userID := uuid.New()
		noteID := uuid.New()
		router.GET("/notes/:id", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Get(c)
		})

		noteSvc.EXPECT().GetByID(gomock.Any(), userID, noteID).Return(nil, domain.ErrForbidden)

		req := httptest.NewRequest(http.MethodGet, "/notes/"+noteID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("returns bad request for invalid ID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteSvc := mocks.NewMockNoteService(ctrl)
		h := handler.NewNoteHandler(noteSvc)

		router := setupRouter()
		userID := uuid.New()
		router.GET("/notes/:id", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Get(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/notes/invalid-uuid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestNoteHandler_Update(t *testing.T) {
	t.Run("updates note successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteSvc := mocks.NewMockNoteService(ctrl)
		h := handler.NewNoteHandler(noteSvc)

		router := setupRouter()
		userID := uuid.New()
		noteID := uuid.New()
		router.PUT("/notes/:id", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Update(c)
		})

		updatedNote := &entity.Note{
			ID:        noteID,
			UserID:    userID,
			Title:     "Updated Title",
			Content:   "Updated content",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		noteSvc.EXPECT().Update(gomock.Any(), userID, noteID, gomock.Any()).Return(updatedNote, nil)

		body := `{"title":"Updated Title","content":"Updated content"}`
		req := httptest.NewRequest(http.MethodPut, "/notes/"+noteID.String(), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "Updated Title", resp["title"])
	})

	t.Run("returns not found for non-existent note", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteSvc := mocks.NewMockNoteService(ctrl)
		h := handler.NewNoteHandler(noteSvc)

		router := setupRouter()
		userID := uuid.New()
		noteID := uuid.New()
		router.PUT("/notes/:id", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Update(c)
		})

		noteSvc.EXPECT().Update(gomock.Any(), userID, noteID, gomock.Any()).Return(nil, domain.ErrNoteNotFound)

		body := `{"title":"Updated Title"}`
		req := httptest.NewRequest(http.MethodPut, "/notes/"+noteID.String(), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns forbidden for other user's note", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteSvc := mocks.NewMockNoteService(ctrl)
		h := handler.NewNoteHandler(noteSvc)

		router := setupRouter()
		userID := uuid.New()
		noteID := uuid.New()
		router.PUT("/notes/:id", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Update(c)
		})

		noteSvc.EXPECT().Update(gomock.Any(), userID, noteID, gomock.Any()).Return(nil, domain.ErrForbidden)

		body := `{"title":"Updated Title"}`
		req := httptest.NewRequest(http.MethodPut, "/notes/"+noteID.String(), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestNoteHandler_Delete(t *testing.T) {
	t.Run("deletes note successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteSvc := mocks.NewMockNoteService(ctrl)
		h := handler.NewNoteHandler(noteSvc)

		router := setupRouter()
		userID := uuid.New()
		noteID := uuid.New()
		router.DELETE("/notes/:id", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Delete(c)
		})

		noteSvc.EXPECT().Delete(gomock.Any(), userID, noteID).Return(nil)

		req := httptest.NewRequest(http.MethodDelete, "/notes/"+noteID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("returns not found for non-existent note", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteSvc := mocks.NewMockNoteService(ctrl)
		h := handler.NewNoteHandler(noteSvc)

		router := setupRouter()
		userID := uuid.New()
		noteID := uuid.New()
		router.DELETE("/notes/:id", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Delete(c)
		})

		noteSvc.EXPECT().Delete(gomock.Any(), userID, noteID).Return(domain.ErrNoteNotFound)

		req := httptest.NewRequest(http.MethodDelete, "/notes/"+noteID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns forbidden for other user's note", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		noteSvc := mocks.NewMockNoteService(ctrl)
		h := handler.NewNoteHandler(noteSvc)

		router := setupRouter()
		userID := uuid.New()
		noteID := uuid.New()
		router.DELETE("/notes/:id", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Delete(c)
		})

		noteSvc.EXPECT().Delete(gomock.Any(), userID, noteID).Return(domain.ErrForbidden)

		req := httptest.NewRequest(http.MethodDelete, "/notes/"+noteID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}
