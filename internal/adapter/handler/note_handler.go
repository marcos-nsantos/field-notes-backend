package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/handler/dto/request"
	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/handler/dto/response"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/valueobject"
	"github.com/marcos-nsantos/field-notes-backend/internal/pkg/httputil"
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/note"
)

type NoteHandler struct {
	noteSvc NoteService
}

func NewNoteHandler(noteSvc NoteService) *NoteHandler {
	return &NoteHandler{noteSvc: noteSvc}
}

func (h *NoteHandler) Create(c *gin.Context) {
	var req request.CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ValidationError(c, err)
		return
	}

	userID := httputil.GetUserID(c)

	var loc *valueobject.Location
	if req.Latitude != nil && req.Longitude != nil {
		loc = valueobject.NewLocation(*req.Latitude, *req.Longitude, req.Altitude, req.Accuracy)
		if !loc.IsValid() {
			httputil.ErrorWithCode(c, http.StatusBadRequest, "INVALID_LOCATION", "invalid coordinates")
			return
		}
	}

	n, err := h.noteSvc.Create(c.Request.Context(), note.CreateInput{
		UserID:   userID,
		Title:    req.Title,
		Content:  req.Content,
		Location: loc,
		ClientID: req.ClientID,
	})
	if err != nil {
		httputil.InternalError(c)
		return
	}

	httputil.Created(c, response.NoteFromEntity(n))
}

func (h *NoteHandler) List(c *gin.Context) {
	var req request.ListNotesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httputil.ValidationError(c, err)
		return
	}

	userID := httputil.GetUserID(c)

	var bbox *valueobject.BoundingBox
	if req.MinLat != nil && req.MaxLat != nil && req.MinLng != nil && req.MaxLng != nil {
		bbox = valueobject.NewBoundingBox(*req.MinLat, *req.MaxLat, *req.MinLng, *req.MaxLng)
		if !bbox.IsValid() {
			httputil.ErrorWithCode(c, http.StatusBadRequest, "INVALID_BBOX", "invalid bounding box")
			return
		}
	}

	notes, pageInfo, err := h.noteSvc.List(c.Request.Context(), note.ListInput{
		UserID:      userID,
		Page:        req.Page,
		PerPage:     req.PerPage,
		BoundingBox: bbox,
	})
	if err != nil {
		httputil.InternalError(c)
		return
	}

	httputil.OK(c, response.NotesListResponse{
		Notes:      response.NotesFromEntities(notes),
		Pagination: response.PaginationFromInfo(pageInfo),
	})
}

func (h *NoteHandler) Get(c *gin.Context) {
	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httputil.ErrorWithCode(c, http.StatusBadRequest, "INVALID_ID", "invalid note id")
		return
	}

	userID := httputil.GetUserID(c)

	n, err := h.noteSvc.GetByID(c.Request.Context(), userID, noteID)
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

	httputil.OK(c, response.NoteFromEntity(n))
}

func (h *NoteHandler) Update(c *gin.Context) {
	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httputil.ErrorWithCode(c, http.StatusBadRequest, "INVALID_ID", "invalid note id")
		return
	}

	var req request.UpdateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ValidationError(c, err)
		return
	}

	userID := httputil.GetUserID(c)

	var loc *valueobject.Location
	if req.Latitude != nil && req.Longitude != nil {
		loc = valueobject.NewLocation(*req.Latitude, *req.Longitude, req.Altitude, req.Accuracy)
		if !loc.IsValid() {
			httputil.ErrorWithCode(c, http.StatusBadRequest, "INVALID_LOCATION", "invalid coordinates")
			return
		}
	}

	n, err := h.noteSvc.Update(c.Request.Context(), userID, noteID, note.UpdateInput{
		Title:    req.Title,
		Content:  req.Content,
		Location: loc,
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

	httputil.OK(c, response.NoteFromEntity(n))
}

func (h *NoteHandler) Delete(c *gin.Context) {
	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httputil.ErrorWithCode(c, http.StatusBadRequest, "INVALID_ID", "invalid note id")
		return
	}

	userID := httputil.GetUserID(c)

	if err := h.noteSvc.Delete(c.Request.Context(), userID, noteID); err != nil {
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

	httputil.NoContent(c)
}
