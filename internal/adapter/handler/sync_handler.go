package handler

import (
	"errors"

	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/handler/dto/request"
	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/handler/dto/response"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain"
	"github.com/marcos-nsantos/field-notes-backend/internal/pkg/httputil"
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/sync"
)

type SyncHandler struct {
	syncSvc SyncService
}

func NewSyncHandler(syncSvc SyncService) *SyncHandler {
	return &SyncHandler{syncSvc: syncSvc}
}

// Sync godoc
//
//	@Summary		Sync notes
//	@Description	Sync notes between client and server using last-write-wins strategy
//	@Tags			sync
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		request.SyncRequest	true	"Sync data with client notes"
//	@Success		200		{object}	response.SyncResponse
//	@Failure		400		{object}	httputil.ErrorResponse	"Device not found or validation error"
//	@Failure		401		{object}	httputil.ErrorResponse
//	@Router			/sync [post]
func (h *SyncHandler) Sync(c *gin.Context) {
	var req request.SyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ValidationError(c, err)
		return
	}

	userID := httputil.GetUserID(c)

	clientNotes := make([]sync.ClientNote, 0, len(req.Notes))
	for _, n := range req.Notes {
		clientNotes = append(clientNotes, sync.ClientNote{
			ClientID:  n.ClientID,
			Title:     n.Title,
			Content:   n.Content,
			Latitude:  n.Latitude,
			Longitude: n.Longitude,
			Altitude:  n.Altitude,
			Accuracy:  n.Accuracy,
			UpdatedAt: n.UpdatedAt,
			IsDeleted: n.IsDeleted,
		})
	}

	result, err := h.syncSvc.BatchSync(c.Request.Context(), sync.SyncInput{
		UserID:      userID,
		DeviceID:    req.DeviceID,
		ClientNotes: clientNotes,
		SyncCursor:  req.SyncCursor,
	})
	if err != nil {
		if errors.Is(err, domain.ErrDeviceNotFound) {
			httputil.ErrorWithCode(c, http.StatusBadRequest, "DEVICE_NOT_FOUND", "device not registered, please login first")
			return
		}
		httputil.InternalError(c)
		return
	}

	httputil.OK(c, response.SyncResultToResponse(result))
}
