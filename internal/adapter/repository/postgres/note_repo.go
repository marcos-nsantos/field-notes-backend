package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/repository"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/entity"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/valueobject"
	"github.com/marcos-nsantos/field-notes-backend/internal/pkg/pagination"
)

type NoteRepo struct {
	pool *pgxpool.Pool
}

func NewNoteRepo(pool *pgxpool.Pool) *NoteRepo {
	return &NoteRepo{pool: pool}
}

func (r *NoteRepo) Create(ctx context.Context, note *entity.Note) error {
	query := `
		INSERT INTO notes (id, user_id, title, content, location, altitude, accuracy, client_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, ST_SetSRID(ST_MakePoint($5, $6), 4326)::geography, $7, $8, $9, $10, $11)
	`
	var lng, lat *float64
	var altitude, accuracy *float64

	if note.Location != nil {
		lng = &note.Location.Longitude
		lat = &note.Location.Latitude
		altitude = note.Location.Altitude
		accuracy = note.Location.Accuracy
	}

	_, err := r.pool.Exec(ctx, query,
		note.ID, note.UserID, note.Title, note.Content,
		lng, lat, altitude, accuracy,
		nullableString(note.ClientID), note.CreatedAt, note.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting note: %w", err)
	}
	return nil
}

func (r *NoteRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Note, error) {
	query := `
		SELECT id, user_id, title, content,
			   ST_Y(location::geometry) as lat, ST_X(location::geometry) as lng,
			   altitude, accuracy, client_id, created_at, updated_at, deleted_at
		FROM notes
		WHERE id = $1
	`
	return r.scanNote(ctx, query, id)
}

func (r *NoteRepo) GetByClientID(ctx context.Context, userID uuid.UUID, clientID string) (*entity.Note, error) {
	query := `
		SELECT id, user_id, title, content,
			   ST_Y(location::geometry) as lat, ST_X(location::geometry) as lng,
			   altitude, accuracy, client_id, created_at, updated_at, deleted_at
		FROM notes
		WHERE user_id = $1 AND client_id = $2
	`
	return r.scanNote(ctx, query, userID, clientID)
}

func (r *NoteRepo) scanNote(ctx context.Context, query string, args ...any) (*entity.Note, error) {
	var note entity.Note
	var lat, lng, altitude, accuracy *float64
	var clientID *string

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&note.ID, &note.UserID, &note.Title, &note.Content,
		&lat, &lng, &altitude, &accuracy,
		&clientID, &note.CreatedAt, &note.UpdatedAt, &note.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNoteNotFound
		}
		return nil, fmt.Errorf("querying note: %w", err)
	}

	if lat != nil && lng != nil {
		note.Location = valueobject.NewLocation(*lat, *lng, altitude, accuracy)
	}
	if clientID != nil {
		note.ClientID = *clientID
	}

	return &note, nil
}

func (r *NoteRepo) List(ctx context.Context, userID uuid.UUID, params repository.NoteListParams) ([]entity.Note, *pagination.Info, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("user_id = $%d", argNum))
	args = append(args, userID)
	argNum++

	if !params.IncludeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}

	if params.BoundingBox != nil {
		bb := params.BoundingBox
		conditions = append(conditions, fmt.Sprintf(`
			ST_Intersects(
				location,
				ST_MakeEnvelope($%d, $%d, $%d, $%d, 4326)::geography
			)
		`, argNum, argNum+1, argNum+2, argNum+3))
		args = append(args, bb.MinLng, bb.MinLat, bb.MaxLng, bb.MaxLat)
		argNum += 4
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM notes WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("counting notes: %w", err)
	}

	// Get notes
	query := fmt.Sprintf(`
		SELECT id, user_id, title, content,
			   ST_Y(location::geometry) as lat, ST_X(location::geometry) as lng,
			   altitude, accuracy, client_id, created_at, updated_at, deleted_at
		FROM notes
		WHERE %s
		ORDER BY updated_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, params.Pagination.Limit(), params.Pagination.Offset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("querying notes: %w", err)
	}
	defer rows.Close()

	var notes []entity.Note
	for rows.Next() {
		var note entity.Note
		var lat, lng, altitude, accuracy *float64
		var clientID *string

		if err := rows.Scan(
			&note.ID, &note.UserID, &note.Title, &note.Content,
			&lat, &lng, &altitude, &accuracy,
			&clientID, &note.CreatedAt, &note.UpdatedAt, &note.DeletedAt,
		); err != nil {
			return nil, nil, fmt.Errorf("scanning note: %w", err)
		}

		if lat != nil && lng != nil {
			note.Location = valueobject.NewLocation(*lat, *lng, altitude, accuracy)
		}
		if clientID != nil {
			note.ClientID = *clientID
		}
		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterating notes: %w", err)
	}

	pageInfo := pagination.NewInfo(params.Pagination.Page, params.Pagination.PerPage, total)
	return notes, pageInfo, nil
}

func (r *NoteRepo) Update(ctx context.Context, note *entity.Note) error {
	query := `
		UPDATE notes
		SET title = $2, content = $3,
			location = ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography,
			altitude = $6, accuracy = $7, updated_at = $8, deleted_at = $9
		WHERE id = $1
	`
	var lng, lat *float64
	var altitude, accuracy *float64

	if note.Location != nil {
		lng = &note.Location.Longitude
		lat = &note.Location.Latitude
		altitude = note.Location.Altitude
		accuracy = note.Location.Accuracy
	}

	result, err := r.pool.Exec(ctx, query,
		note.ID, note.Title, note.Content,
		lng, lat, altitude, accuracy,
		note.UpdatedAt, note.DeletedAt,
	)
	if err != nil {
		return fmt.Errorf("updating note: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNoteNotFound
	}
	return nil
}

func (r *NoteRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE notes
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("soft deleting note: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNoteNotFound
	}
	return nil
}

func (r *NoteRepo) GetModifiedSince(ctx context.Context, userID uuid.UUID, since time.Time, limit int) ([]entity.Note, error) {
	query := `
		SELECT id, user_id, title, content,
			   ST_Y(location::geometry) as lat, ST_X(location::geometry) as lng,
			   altitude, accuracy, client_id, created_at, updated_at, deleted_at
		FROM notes
		WHERE user_id = $1 AND updated_at > $2
		ORDER BY updated_at ASC
		LIMIT $3
	`
	rows, err := r.pool.Query(ctx, query, userID, since, limit)
	if err != nil {
		return nil, fmt.Errorf("querying modified notes: %w", err)
	}
	defer rows.Close()

	var notes []entity.Note
	for rows.Next() {
		var note entity.Note
		var lat, lng, altitude, accuracy *float64
		var clientID *string

		if err := rows.Scan(
			&note.ID, &note.UserID, &note.Title, &note.Content,
			&lat, &lng, &altitude, &accuracy,
			&clientID, &note.CreatedAt, &note.UpdatedAt, &note.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning note: %w", err)
		}

		if lat != nil && lng != nil {
			note.Location = valueobject.NewLocation(*lat, *lng, altitude, accuracy)
		}
		if clientID != nil {
			note.ClientID = *clientID
		}
		notes = append(notes, note)
	}

	return notes, rows.Err()
}

func (r *NoteRepo) BatchUpsert(ctx context.Context, notes []entity.Note) error {
	if len(notes) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, note := range notes {
		var lng, lat *float64
		var altitude, accuracy *float64

		if note.Location != nil {
			lng = &note.Location.Longitude
			lat = &note.Location.Latitude
			altitude = note.Location.Altitude
			accuracy = note.Location.Accuracy
		}

		query := `
			INSERT INTO notes (id, user_id, title, content, location, altitude, accuracy, client_id, created_at, updated_at, deleted_at)
			VALUES ($1, $2, $3, $4, ST_SetSRID(ST_MakePoint($5, $6), 4326)::geography, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (user_id, client_id)
			DO UPDATE SET
				title = EXCLUDED.title,
				content = EXCLUDED.content,
				location = EXCLUDED.location,
				altitude = EXCLUDED.altitude,
				accuracy = EXCLUDED.accuracy,
				updated_at = EXCLUDED.updated_at,
				deleted_at = EXCLUDED.deleted_at
			WHERE notes.updated_at < EXCLUDED.updated_at
		`
		_, err := tx.Exec(ctx, query,
			note.ID, note.UserID, note.Title, note.Content,
			lng, lat, altitude, accuracy,
			nullableString(note.ClientID), note.CreatedAt, note.UpdatedAt, note.DeletedAt,
		)
		if err != nil {
			return fmt.Errorf("upserting note: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
