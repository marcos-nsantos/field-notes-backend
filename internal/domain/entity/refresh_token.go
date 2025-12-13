package entity

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	DeviceID  uuid.UUID
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

func NewRefreshToken(userID, deviceID uuid.UUID, token string, expiresAt time.Time) *RefreshToken {
	return &RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		DeviceID:  deviceID,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}
}

func (rt *RefreshToken) Revoke() {
	now := time.Now().UTC()
	rt.RevokedAt = &now
}

func (rt *RefreshToken) IsValid() bool {
	return rt.RevokedAt == nil && rt.ExpiresAt.After(time.Now().UTC())
}

func (rt *RefreshToken) IsExpired() bool {
	return rt.ExpiresAt.Before(time.Now().UTC())
}

func (rt *RefreshToken) IsRevoked() bool {
	return rt.RevokedAt != nil
}
