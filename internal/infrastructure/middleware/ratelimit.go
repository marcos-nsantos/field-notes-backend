package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/marcos-nsantos/field-notes-backend/internal/infrastructure/config"
)

type RateLimiter struct {
	client         *redis.Client
	requestsPerMin int
	windowSize     time.Duration
}

func NewRateLimiter(client *redis.Client, cfg config.RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		client:         client,
		requestsPerMin: cfg.RequestsPerMin,
		windowSize:     time.Minute,
	}
}

func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		key := fmt.Sprintf("ratelimit:%s", c.ClientIP())

		allowed, remaining, err := rl.isAllowed(ctx, key)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.requestsPerMin))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		if !allowed {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    "RATE_LIMITED",
				"message": "too many requests, please try again later",
			})
			return
		}

		c.Next()
	}
}

func (rl *RateLimiter) isAllowed(ctx context.Context, key string) (bool, int, error) {
	now := time.Now().UnixMilli()
	windowStart := now - rl.windowSize.Milliseconds()

	pipe := rl.client.Pipeline()

	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: now,
	})

	countCmd := pipe.ZCard(ctx, key)

	pipe.Expire(ctx, key, rl.windowSize)

	if _, err := pipe.Exec(ctx); err != nil {
		return true, rl.requestsPerMin, err
	}

	count := int(countCmd.Val())
	remaining := rl.requestsPerMin - count
	if remaining < 0 {
		remaining = 0
	}

	return count <= rl.requestsPerMin, remaining, nil
}
