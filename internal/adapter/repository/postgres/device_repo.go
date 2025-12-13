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

type DeviceRepo struct {
	pool *pgxpool.Pool
}

func NewDeviceRepo(pool *pgxpool.Pool) *DeviceRepo {
	return &DeviceRepo{pool: pool}
}

func (r *DeviceRepo) Create(ctx context.Context, device *entity.Device) error {
	query := `
		INSERT INTO devices (id, user_id, device_id, platform, name, sync_cursor, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		device.ID, device.UserID, device.DeviceID, device.Platform,
		device.Name, device.SyncCursor, device.CreatedAt, device.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting device: %w", err)
	}
	return nil
}

func (r *DeviceRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Device, error) {
	query := `
		SELECT id, user_id, device_id, platform, name, sync_cursor, created_at, updated_at
		FROM devices
		WHERE id = $1
	`
	var device entity.Device
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&device.ID, &device.UserID, &device.DeviceID, &device.Platform,
		&device.Name, &device.SyncCursor, &device.CreatedAt, &device.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrDeviceNotFound
		}
		return nil, fmt.Errorf("querying device: %w", err)
	}
	return &device, nil
}

func (r *DeviceRepo) GetByUserAndDeviceID(ctx context.Context, userID uuid.UUID, deviceID string) (*entity.Device, error) {
	query := `
		SELECT id, user_id, device_id, platform, name, sync_cursor, created_at, updated_at
		FROM devices
		WHERE user_id = $1 AND device_id = $2
	`
	var device entity.Device
	err := r.pool.QueryRow(ctx, query, userID, deviceID).Scan(
		&device.ID, &device.UserID, &device.DeviceID, &device.Platform,
		&device.Name, &device.SyncCursor, &device.CreatedAt, &device.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrDeviceNotFound
		}
		return nil, fmt.Errorf("querying device: %w", err)
	}
	return &device, nil
}

func (r *DeviceRepo) Update(ctx context.Context, device *entity.Device) error {
	query := `
		UPDATE devices
		SET platform = $2, name = $3, sync_cursor = $4, updated_at = $5
		WHERE id = $1
	`
	result, err := r.pool.Exec(ctx, query,
		device.ID, device.Platform, device.Name, device.SyncCursor, device.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("updating device: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrDeviceNotFound
	}
	return nil
}

func (r *DeviceRepo) Upsert(ctx context.Context, device *entity.Device) error {
	query := `
		INSERT INTO devices (id, user_id, device_id, platform, name, sync_cursor, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, device_id)
		DO UPDATE SET platform = EXCLUDED.platform, name = EXCLUDED.name, updated_at = EXCLUDED.updated_at
	`
	_, err := r.pool.Exec(ctx, query,
		device.ID, device.UserID, device.DeviceID, device.Platform,
		device.Name, device.SyncCursor, device.CreatedAt, device.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upserting device: %w", err)
	}
	return nil
}
