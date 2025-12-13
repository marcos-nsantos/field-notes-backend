package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/handler/dto/response"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain"
	"github.com/marcos-nsantos/field-notes-backend/internal/pkg/httputil"
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/upload"
)

const maxUploadSize = 10 << 20 // 10MB

type UploadHandler struct {
	uploadSvc UploadService
}

func NewUploadHandler(uploadSvc UploadService) *UploadHandler {
	return &UploadHandler{uploadSvc: uploadSvc}
}

func (h *UploadHandler) Upload(c *gin.Context) {
	noteID, err := uuid.Parse(c.Param("note_id"))
	if err != nil {
		httputil.ErrorWithCode(c, http.StatusBadRequest, "INVALID_ID", "invalid note id")
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		httputil.ErrorWithCode(c, http.StatusBadRequest, "INVALID_FILE", "file is required")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if !isAllowedImageType(contentType) {
		httputil.ErrorWithCode(c, http.StatusBadRequest, "INVALID_TYPE", "only jpeg and png images are allowed")
		return
	}

	userID := httputil.GetUserID(c)

	result, err := h.uploadSvc.Upload(c.Request.Context(), upload.UploadInput{
		UserID:      userID,
		NoteID:      noteID,
		File:        file,
		Filename:    header.Filename,
		ContentType: contentType,
		Size:        header.Size,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNoteNotFound):
			httputil.ErrorWithCode(c, http.StatusNotFound, "NOT_FOUND", "note not found")
		case errors.Is(err, domain.ErrForbidden):
			httputil.ErrorWithCode(c, http.StatusForbidden, "FORBIDDEN", "access denied")
		default:
			httputil.InternalError(c)
		}
		return
	}

	httputil.Created(c, response.UploadResultToResponse(result))
}

func (h *UploadHandler) Delete(c *gin.Context) {
	photoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httputil.ErrorWithCode(c, http.StatusBadRequest, "INVALID_ID", "invalid photo id")
		return
	}

	userID := httputil.GetUserID(c)

	if err := h.uploadSvc.Delete(c.Request.Context(), userID, photoID); err != nil {
		switch {
		case errors.Is(err, domain.ErrPhotoNotFound):
			httputil.ErrorWithCode(c, http.StatusNotFound, "NOT_FOUND", "photo not found")
		case errors.Is(err, domain.ErrForbidden):
			httputil.ErrorWithCode(c, http.StatusForbidden, "FORBIDDEN", "access denied")
		default:
			httputil.InternalError(c)
		}
		return
	}

	httputil.NoContent(c)
}

func isAllowedImageType(contentType string) bool {
	return contentType == "image/jpeg" || contentType == "image/png" || contentType == "image/jpg"
}
