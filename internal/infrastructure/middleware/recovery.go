package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered",
					zap.Any("error", err),
					zap.String("stack", string(debug.Stack())),
					zap.String("request_id", c.GetString(RequestIDKey)),
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":      "internal server error",
					"request_id": c.GetString(RequestIDKey),
				})
			}
		}()
		c.Next()
	}
}
