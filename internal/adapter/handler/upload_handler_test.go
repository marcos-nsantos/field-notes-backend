package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
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
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/upload"
)

func createMultipartRequest(t *testing.T, url, fieldName, fileName, contentType string, fileContent []byte) (*http.Request, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fileName))
	h.Set("Content-Type", contentType)

	part, err := writer.CreatePart(h)
	require.NoError(t, err)

	_, err = part.Write(fileContent)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, writer.FormDataContentType()
}

func TestUploadHandler_Upload(t *testing.T) {
	t.Run("uploads image successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		uploadSvc := mocks.NewMockUploadService(ctrl)
		h := handler.NewUploadHandler(uploadSvc)

		router := setupRouter()
		userID := uuid.New()
		noteID := uuid.New()
		router.POST("/notes/:note_id/upload", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Upload(c)
		})

		photoID := uuid.New()
		result := &upload.UploadResult{
			Photo: &entity.Photo{
				ID:        photoID,
				NoteID:    noteID,
				URL:       "https://example.com/photo.jpg",
				Key:       "notes/123/photo.jpg",
				MimeType:  "image/jpeg",
				Size:      1024,
				Width:     800,
				Height:    600,
				CreatedAt: time.Now(),
			},
			URL:       "https://example.com/photo.jpg",
			SignedURL: "https://example.com/photo.jpg?signed=xxx",
		}

		uploadSvc.EXPECT().Upload(gomock.Any(), gomock.Any()).Return(result, nil)

		fileContent := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG header
		req, _ := createMultipartRequest(t, "/notes/"+noteID.String()+"/upload", "file", "test.jpg", "image/jpeg", fileContent)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp["photo"])
		assert.NotEmpty(t, resp["url"])
	})

	t.Run("returns error for invalid note ID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		uploadSvc := mocks.NewMockUploadService(ctrl)
		h := handler.NewUploadHandler(uploadSvc)

		router := setupRouter()
		userID := uuid.New()
		router.POST("/notes/:note_id/upload", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Upload(c)
		})

		fileContent := []byte{0xFF, 0xD8, 0xFF, 0xE0}
		req, _ := createMultipartRequest(t, "/notes/invalid-uuid/upload", "file", "test.jpg", "image/jpeg", fileContent)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_ID", resp["code"])
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		uploadSvc := mocks.NewMockUploadService(ctrl)
		h := handler.NewUploadHandler(uploadSvc)

		router := setupRouter()
		userID := uuid.New()
		noteID := uuid.New()
		router.POST("/notes/:note_id/upload", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Upload(c)
		})

		req := httptest.NewRequest(http.MethodPost, "/notes/"+noteID.String()+"/upload", nil)
		req.Header.Set("Content-Type", "multipart/form-data")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_FILE", resp["code"])
	})

	t.Run("returns error for note not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		uploadSvc := mocks.NewMockUploadService(ctrl)
		h := handler.NewUploadHandler(uploadSvc)

		router := setupRouter()
		userID := uuid.New()
		noteID := uuid.New()
		router.POST("/notes/:note_id/upload", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Upload(c)
		})

		uploadSvc.EXPECT().Upload(gomock.Any(), gomock.Any()).Return(nil, domain.ErrNoteNotFound)

		fileContent := []byte{0xFF, 0xD8, 0xFF, 0xE0}
		req, _ := createMultipartRequest(t, "/notes/"+noteID.String()+"/upload", "file", "test.jpg", "image/jpeg", fileContent)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns forbidden for other user's note", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		uploadSvc := mocks.NewMockUploadService(ctrl)
		h := handler.NewUploadHandler(uploadSvc)

		router := setupRouter()
		userID := uuid.New()
		noteID := uuid.New()
		router.POST("/notes/:note_id/upload", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Upload(c)
		})

		uploadSvc.EXPECT().Upload(gomock.Any(), gomock.Any()).Return(nil, domain.ErrForbidden)

		fileContent := []byte{0xFF, 0xD8, 0xFF, 0xE0}
		req, _ := createMultipartRequest(t, "/notes/"+noteID.String()+"/upload", "file", "test.jpg", "image/jpeg", fileContent)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestUploadHandler_Delete(t *testing.T) {
	t.Run("deletes photo successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		uploadSvc := mocks.NewMockUploadService(ctrl)
		h := handler.NewUploadHandler(uploadSvc)

		router := setupRouter()
		userID := uuid.New()
		photoID := uuid.New()
		router.DELETE("/photos/:id", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Delete(c)
		})

		uploadSvc.EXPECT().Delete(gomock.Any(), userID, photoID).Return(nil)

		req := httptest.NewRequest(http.MethodDelete, "/photos/"+photoID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("returns error for invalid photo ID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		uploadSvc := mocks.NewMockUploadService(ctrl)
		h := handler.NewUploadHandler(uploadSvc)

		router := setupRouter()
		userID := uuid.New()
		router.DELETE("/photos/:id", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Delete(c)
		})

		req := httptest.NewRequest(http.MethodDelete, "/photos/invalid-uuid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_ID", resp["code"])
	})

	t.Run("returns not found for non-existent photo", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		uploadSvc := mocks.NewMockUploadService(ctrl)
		h := handler.NewUploadHandler(uploadSvc)

		router := setupRouter()
		userID := uuid.New()
		photoID := uuid.New()
		router.DELETE("/photos/:id", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Delete(c)
		})

		uploadSvc.EXPECT().Delete(gomock.Any(), userID, photoID).Return(domain.ErrPhotoNotFound)

		req := httptest.NewRequest(http.MethodDelete, "/photos/"+photoID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns forbidden for other user's photo", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		uploadSvc := mocks.NewMockUploadService(ctrl)
		h := handler.NewUploadHandler(uploadSvc)

		router := setupRouter()
		userID := uuid.New()
		photoID := uuid.New()
		router.DELETE("/photos/:id", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Delete(c)
		})

		uploadSvc.EXPECT().Delete(gomock.Any(), userID, photoID).Return(domain.ErrForbidden)

		req := httptest.NewRequest(http.MethodDelete, "/photos/"+photoID.String(), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}
