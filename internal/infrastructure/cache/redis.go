package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/marcos-nsantos/field-notes-backend/internal/infrastructure/config"
)

func NewRedisClient(cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connecting to redis: %w", err)
	}

	return client, nil
}
