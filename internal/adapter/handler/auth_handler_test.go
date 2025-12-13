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
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/auth"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestAuthHandler_Register(t *testing.T) {
	t.Run("registers user successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		authSvc := mocks.NewMockAuthService(ctrl)
		h := handler.NewAuthHandler(authSvc)

		router := setupRouter()
		router.POST("/register", h.Register)

		user := &entity.User{
			ID:    uuid.New(),
			Email: "test@example.com",
			Name:  "Test User",
		}

		authSvc.EXPECT().Register(gomock.Any(), auth.RegisterInput{
			Email:    "test@example.com",
			Password: "password123",
			Name:     "Test User",
		}).Return(user, nil)

		body := `{"email":"test@example.com","password":"password123","name":"Test User"}`
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "test@example.com", resp["email"])
		assert.Equal(t, "Test User", resp["name"])
	})

	t.Run("returns conflict for existing email", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		authSvc := mocks.NewMockAuthService(ctrl)
		h := handler.NewAuthHandler(authSvc)

		router := setupRouter()
		router.POST("/register", h.Register)

		authSvc.EXPECT().Register(gomock.Any(), gomock.Any()).Return(nil, domain.ErrUserAlreadyExists)

		body := `{"email":"existing@example.com","password":"password123","name":"Test User"}`
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("returns validation error for invalid input", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		authSvc := mocks.NewMockAuthService(ctrl)
		h := handler.NewAuthHandler(authSvc)

		router := setupRouter()
		router.POST("/register", h.Register)

		body := `{"email":"invalid","password":"short","name":""}`
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAuthHandler_Login(t *testing.T) {
	t.Run("logs in successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		authSvc := mocks.NewMockAuthService(ctrl)
		h := handler.NewAuthHandler(authSvc)

		router := setupRouter()
		router.POST("/login", h.Login)

		userID := uuid.New()
		user := &entity.User{
			ID:    userID,
			Email: "test@example.com",
			Name:  "Test User",
		}
		tokens := &auth.TokenPair{
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
			ExpiresAt:    time.Now().Add(15 * time.Minute),
		}

		authSvc.EXPECT().Login(gomock.Any(), auth.LoginInput{
			Email:      "test@example.com",
			Password:   "password123",
			DeviceID:   "device-123",
			DeviceName: "iPhone",
			Platform:   "ios",
		}).Return(tokens, user, nil)

		body := `{"email":"test@example.com","password":"password123","device_id":"device-123","device_name":"iPhone","platform":"ios"}`
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "access-token", resp["access_token"])
		assert.Equal(t, "refresh-token", resp["refresh_token"])
	})

	t.Run("returns unauthorized for invalid credentials", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		authSvc := mocks.NewMockAuthService(ctrl)
		h := handler.NewAuthHandler(authSvc)

		router := setupRouter()
		router.POST("/login", h.Login)

		authSvc.EXPECT().Login(gomock.Any(), gomock.Any()).Return(nil, nil, domain.ErrInvalidCredentials)

		body := `{"email":"test@example.com","password":"wrong","device_id":"device-123","platform":"ios"}`
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAuthHandler_Refresh(t *testing.T) {
	t.Run("refreshes token successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		authSvc := mocks.NewMockAuthService(ctrl)
		h := handler.NewAuthHandler(authSvc)

		router := setupRouter()
		router.POST("/refresh", h.Refresh)

		tokens := &auth.TokenPair{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
			ExpiresAt:    time.Now().Add(15 * time.Minute),
		}

		authSvc.EXPECT().Refresh(gomock.Any(), "valid-refresh-token").Return(tokens, nil)

		body := `{"refresh_token":"valid-refresh-token"}`
		req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "new-access-token", resp["access_token"])
		assert.Equal(t, "new-refresh-token", resp["refresh_token"])
	})

	t.Run("returns unauthorized for expired token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		authSvc := mocks.NewMockAuthService(ctrl)
		h := handler.NewAuthHandler(authSvc)

		router := setupRouter()
		router.POST("/refresh", h.Refresh)

		authSvc.EXPECT().Refresh(gomock.Any(), "expired-token").Return(nil, domain.ErrTokenExpired)

		body := `{"refresh_token":"expired-token"}`
		req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns unauthorized for revoked token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		authSvc := mocks.NewMockAuthService(ctrl)
		h := handler.NewAuthHandler(authSvc)

		router := setupRouter()
		router.POST("/refresh", h.Refresh)

		authSvc.EXPECT().Refresh(gomock.Any(), "revoked-token").Return(nil, domain.ErrTokenRevoked)

		body := `{"refresh_token":"revoked-token"}`
		req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	t.Run("logs out successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		authSvc := mocks.NewMockAuthService(ctrl)
		h := handler.NewAuthHandler(authSvc)

		router := setupRouter()
		userID := uuid.New()
		router.POST("/logout", func(c *gin.Context) {
			c.Set("user_id", userID)
			h.Logout(c)
		})

		authSvc.EXPECT().Logout(gomock.Any(), userID).Return(nil)

		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}
