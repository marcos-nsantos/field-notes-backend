package storage

import (
	"context"
	"io"
	"time"
)

type ImageStorage interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) error
	GetURL(key string) string
	GetSignedURL(key string, expiry time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
}

type ImageProcessor interface {
	Process(reader io.Reader) (io.Reader, int64, int, int, error)
}
