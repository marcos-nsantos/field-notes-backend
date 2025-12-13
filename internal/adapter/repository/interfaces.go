package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/marcos-nsantos/field-notes-backend/internal/domain/entity"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/valueobject"
	"github.com/marcos-nsantos/field-notes-backend/internal/pkg/pagination"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

type NoteRepository interface {
	Create(ctx context.Context, note *entity.Note) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Note, error)
	GetByClientID(ctx context.Context, userID uuid.UUID, clientID string) (*entity.Note, error)
	List(ctx context.Context, userID uuid.UUID, params NoteListParams) ([]entity.Note, *pagination.Info, error)
	Update(ctx context.Context, note *entity.Note) error
	SoftDelete(ctx context.Context, id uuid.UUID) error

	// Sync operations
	GetModifiedSince(ctx context.Context, userID uuid.UUID, since time.Time, limit int) ([]entity.Note, error)
	BatchUpsert(ctx context.Context, notes []entity.Note) error
}

type NoteListParams struct {
	Pagination     pagination.Params
	BoundingBox    *valueobject.BoundingBox
	IncludeDeleted bool
}

type PhotoRepository interface {
	Create(ctx context.Context, photo *entity.Photo) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Photo, error)
	GetByNoteID(ctx context.Context, noteID uuid.UUID) ([]entity.Photo, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByNoteID(ctx context.Context, noteID uuid.UUID) error
}

type DeviceRepository interface {
	Create(ctx context.Context, device *entity.Device) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Device, error)
	GetByUserAndDeviceID(ctx context.Context, userID uuid.UUID, deviceID string) (*entity.Device, error)
	Update(ctx context.Context, device *entity.Device) error
	Upsert(ctx context.Context, device *entity.Device) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *entity.RefreshToken) error
	GetByToken(ctx context.Context, token string) (*entity.RefreshToken, error)
	RevokeByUserID(ctx context.Context, userID uuid.UUID) error
	RevokeByDeviceID(ctx context.Context, deviceID uuid.UUID) error
	Revoke(ctx context.Context, id uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}
