package handler

import (
	"context"

	"github.com/google/uuid"

	"github.com/marcos-nsantos/field-notes-backend/internal/domain/entity"
	"github.com/marcos-nsantos/field-notes-backend/internal/pkg/pagination"
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/auth"
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/note"
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/sync"
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/upload"
)

//go:generate mockgen -source=interfaces.go -destination=../../mocks/handler_mocks.go -package=mocks

type AuthService interface {
	Register(ctx context.Context, input auth.RegisterInput) (*entity.User, error)
	Login(ctx context.Context, input auth.LoginInput) (*auth.TokenPair, *entity.User, error)
	Refresh(ctx context.Context, refreshToken string) (*auth.TokenPair, error)
	Logout(ctx context.Context, userID uuid.UUID) error
}

type NoteService interface {
	Create(ctx context.Context, input note.CreateInput) (*entity.Note, error)
	List(ctx context.Context, input note.ListInput) ([]entity.Note, *pagination.Info, error)
	GetByID(ctx context.Context, userID, noteID uuid.UUID) (*entity.Note, error)
	Update(ctx context.Context, userID, noteID uuid.UUID, input note.UpdateInput) (*entity.Note, error)
	Delete(ctx context.Context, userID, noteID uuid.UUID) error
}

type SyncService interface {
	BatchSync(ctx context.Context, input sync.SyncInput) (*sync.SyncResult, error)
}

type UploadService interface {
	Upload(ctx context.Context, input upload.UploadInput) (*upload.UploadResult, error)
	Delete(ctx context.Context, userID, photoID uuid.UUID) error
}
