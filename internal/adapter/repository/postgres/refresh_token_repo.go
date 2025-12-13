package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/marcos-nsantos/field-notes-backend/internal/domain"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/entity"
)

type RefreshTokenRepo struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepo(pool *pgxpool.Pool) *RefreshTokenRepo {
	return &RefreshTokenRepo{pool: pool}
}

func (r *RefreshTokenRepo) Create(ctx context.Context, token *entity.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, device_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query,
		token.ID, token.UserID, token.DeviceID, token.Token, token.ExpiresAt, token.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepo) GetByToken(ctx context.Context, token string) (*entity.RefreshToken, error) {
	query := `
		SELECT id, user_id, device_id, token, expires_at, created_at, revoked_at
		FROM refresh_tokens
		WHERE token = $1
	`
	var rt entity.RefreshToken
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&rt.ID, &rt.UserID, &rt.DeviceID, &rt.Token, &rt.ExpiresAt, &rt.CreatedAt, &rt.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTokenInvalid
		}
		return nil, fmt.Errorf("querying refresh token: %w", err)
	}
	return &rt, nil
}

func (r *RefreshTokenRepo) RevokeByUserID(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("revoking tokens by user: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepo) RevokeByDeviceID(ctx context.Context, deviceID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE device_id = $1 AND revoked_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, deviceID)
	if err != nil {
		return fmt.Errorf("revoking tokens by device: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL
	`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("revoking token: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrTokenInvalid
	}
	return nil
}

func (r *RefreshTokenRepo) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM refresh_tokens WHERE expires_at < NOW() OR revoked_at IS NOT NULL`
	_, err := r.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("deleting expired tokens: %w", err)
	}
	return nil
}
