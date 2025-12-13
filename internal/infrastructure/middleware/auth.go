package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/marcos-nsantos/field-notes-backend/internal/infrastructure/auth"
	"github.com/marcos-nsantos/field-notes-backend/internal/pkg/httputil"
)

const (
	UserIDKey    = "user_id"
	BearerPrefix = "Bearer "
)

type AuthMiddleware struct {
	jwtSvc *auth.JWTService
}

func NewAuthMiddleware(jwtSvc *auth.JWTService) *AuthMiddleware {
	return &AuthMiddleware{jwtSvc: jwtSvc}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			httputil.Error(c, http.StatusUnauthorized, "authorization header required")
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, BearerPrefix) {
			httputil.Error(c, http.StatusUnauthorized, "invalid authorization format")
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, BearerPrefix)
		userID, err := m.jwtSvc.ValidateAccessToken(token)
		if err != nil {
			httputil.Error(c, http.StatusUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}

		c.Set(UserIDKey, userID)
		c.Next()
	}
}
