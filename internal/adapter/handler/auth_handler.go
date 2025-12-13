package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/handler/dto/request"
	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/handler/dto/response"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain"
	"github.com/marcos-nsantos/field-notes-backend/internal/pkg/httputil"
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/auth"
)

type AuthHandler struct {
	authSvc *auth.Service
}

func NewAuthHandler(authSvc *auth.Service) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req request.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ValidationError(c, err)
		return
	}

	user, err := h.authSvc.Register(c.Request.Context(), auth.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			httputil.ErrorWithCode(c, http.StatusConflict, "USER_EXISTS", "email already registered")
			return
		}
		httputil.InternalError(c)
		return
	}

	httputil.Created(c, response.UserFromEntity(user))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req request.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ValidationError(c, err)
		return
	}

	tokens, user, err := h.authSvc.Login(c.Request.Context(), auth.LoginInput{
		Email:      req.Email,
		Password:   req.Password,
		DeviceID:   req.DeviceID,
		DeviceName: req.DeviceName,
		Platform:   req.Platform,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			httputil.ErrorWithCode(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
			return
		}
		httputil.InternalError(c)
		return
	}

	httputil.OK(c, response.LoginResponse{
		User:         response.UserFromEntity(user),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresAt:    tokens.ExpiresAt,
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req request.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ValidationError(c, err)
		return
	}

	tokens, err := h.authSvc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTokenExpired):
			httputil.ErrorWithCode(c, http.StatusUnauthorized, "TOKEN_EXPIRED", "refresh token expired")
		case errors.Is(err, domain.ErrTokenRevoked):
			httputil.ErrorWithCode(c, http.StatusUnauthorized, "TOKEN_REVOKED", "refresh token revoked")
		case errors.Is(err, domain.ErrTokenInvalid):
			httputil.ErrorWithCode(c, http.StatusUnauthorized, "TOKEN_INVALID", "invalid refresh token")
		default:
			httputil.InternalError(c)
		}
		return
	}

	httputil.OK(c, response.RefreshResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresAt:    tokens.ExpiresAt,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	userID := httputil.GetUserID(c)
	if err := h.authSvc.Logout(c.Request.Context(), userID); err != nil {
		httputil.InternalError(c)
		return
	}
	httputil.NoContent(c)
}
