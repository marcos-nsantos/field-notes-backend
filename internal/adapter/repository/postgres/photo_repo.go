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

type PhotoRepo struct {
	pool *pgxpool.Pool
}

func NewPhotoRepo(pool *pgxpool.Pool) *PhotoRepo {
	return &PhotoRepo{pool: pool}
}

func (r *PhotoRepo) Create(ctx context.Context, photo *entity.Photo) error {
	query := `
		INSERT INTO photos (id, note_id, url, key, mime_type, size, width, height, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query,
		photo.ID, photo.NoteID, photo.URL, photo.Key,
		photo.MimeType, photo.Size, photo.Width, photo.Height, photo.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting photo: %w", err)
	}
	return nil
}

func (r *PhotoRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Photo, error) {
	query := `
		SELECT id, note_id, url, key, mime_type, size, width, height, created_at
		FROM photos
		WHERE id = $1
	`
	var photo entity.Photo
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&photo.ID, &photo.NoteID, &photo.URL, &photo.Key,
		&photo.MimeType, &photo.Size, &photo.Width, &photo.Height, &photo.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPhotoNotFound
		}
		return nil, fmt.Errorf("querying photo: %w", err)
	}
	return &photo, nil
}

func (r *PhotoRepo) GetByNoteID(ctx context.Context, noteID uuid.UUID) ([]entity.Photo, error) {
	query := `
		SELECT id, note_id, url, key, mime_type, size, width, height, created_at
		FROM photos
		WHERE note_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, noteID)
	if err != nil {
		return nil, fmt.Errorf("querying photos: %w", err)
	}
	defer rows.Close()

	var photos []entity.Photo
	for rows.Next() {
		var photo entity.Photo
		if err := rows.Scan(
			&photo.ID, &photo.NoteID, &photo.URL, &photo.Key,
			&photo.MimeType, &photo.Size, &photo.Width, &photo.Height, &photo.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning photo: %w", err)
		}
		photos = append(photos, photo)
	}

	return photos, rows.Err()
}

func (r *PhotoRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM photos WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting photo: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrPhotoNotFound
	}
	return nil
}

func (r *PhotoRepo) DeleteByNoteID(ctx context.Context, noteID uuid.UUID) error {
	query := `DELETE FROM photos WHERE note_id = $1`
	_, err := r.pool.Exec(ctx, query, noteID)
	if err != nil {
		return fmt.Errorf("deleting photos by note: %w", err)
	}
	return nil
}
