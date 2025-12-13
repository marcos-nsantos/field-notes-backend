package httputil

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/marcos-nsantos/field-notes-backend/internal/pkg/apperror"
)

type ErrorResponse struct {
	Error     string `json:"error"`
	Code      string `json:"code,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, data)
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func Error(c *gin.Context, status int, message string) {
	c.JSON(status, ErrorResponse{
		Error:     message,
		RequestID: GetRequestID(c),
	})
}

func ErrorWithCode(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{
		Error:     message,
		Code:      code,
		RequestID: GetRequestID(c),
	})
}

func ValidationError(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error:     err.Error(),
		Code:      "VALIDATION_ERROR",
		RequestID: GetRequestID(c),
	})
}

func InternalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error:     "internal server error",
		Code:      "INTERNAL_ERROR",
		RequestID: GetRequestID(c),
	})
}

func HandleError(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.StatusCode, ErrorResponse{
			Error:     appErr.Message,
			Code:      appErr.Code,
			RequestID: GetRequestID(c),
		})
		return
	}
	InternalError(c)
}

func GetUserID(c *gin.Context) uuid.UUID {
	if id, exists := c.Get("user_id"); exists {
		return id.(uuid.UUID)
	}
	return uuid.Nil
}

func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get("request_id"); exists {
		return id.(string)
	}
	return ""
}
