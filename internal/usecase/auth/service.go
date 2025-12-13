package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/repository"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/entity"
	"github.com/marcos-nsantos/field-notes-backend/internal/infrastructure/auth"
)

type Service struct {
	userRepo         repository.UserRepository
	deviceRepo       repository.DeviceRepository
	refreshTokenRepo repository.RefreshTokenRepository
	jwtSvc           *auth.JWTService
	passwordHasher   *auth.PasswordHasher
	refreshTokenTTL  time.Duration
}

func NewService(
	userRepo repository.UserRepository,
	deviceRepo repository.DeviceRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtSvc *auth.JWTService,
	passwordHasher *auth.PasswordHasher,
	refreshTokenTTL time.Duration,
) *Service {
	return &Service{
		userRepo:         userRepo,
		deviceRepo:       deviceRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtSvc:           jwtSvc,
		passwordHasher:   passwordHasher,
		refreshTokenTTL:  refreshTokenTTL,
	}
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type RegisterInput struct {
	Email    string
	Password string
	Name     string
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (*entity.User, error) {
	exists, err := s.userRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("checking email: %w", err)
	}
	if exists {
		return nil, domain.ErrUserAlreadyExists
	}

	hash, err := s.passwordHasher.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user := entity.NewUser(input.Email, hash, input.Name)
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return user, nil
}

type LoginInput struct {
	Email      string
	Password   string
	DeviceID   string
	DeviceName string
	Platform   string
}

func (s *Service) Login(ctx context.Context, input LoginInput) (*TokenPair, *entity.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, nil, domain.ErrInvalidCredentials
	}

	if err := s.passwordHasher.Compare(user.PasswordHash, input.Password); err != nil {
		return nil, nil, domain.ErrInvalidCredentials
	}

	device := entity.NewDevice(user.ID, input.DeviceID, input.Platform, input.DeviceName)
	if err := s.deviceRepo.Upsert(ctx, device); err != nil {
		return nil, nil, fmt.Errorf("upserting device: %w", err)
	}

	device, err = s.deviceRepo.GetByUserAndDeviceID(ctx, user.ID, input.DeviceID)
	if err != nil {
		return nil, nil, fmt.Errorf("getting device: %w", err)
	}

	if err := s.refreshTokenRepo.RevokeByDeviceID(ctx, device.ID); err != nil {
		return nil, nil, fmt.Errorf("revoking old tokens: %w", err)
	}

	tokens, err := s.generateTokenPair(ctx, user.ID, device.ID)
	if err != nil {
		return nil, nil, err
	}

	return tokens, user, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	rt, err := s.refreshTokenRepo.GetByToken(ctx, refreshToken)
	if err != nil {
		return nil, domain.ErrTokenInvalid
	}

	if rt.IsRevoked() {
		return nil, domain.ErrTokenRevoked
	}

	if rt.IsExpired() {
		return nil, domain.ErrTokenExpired
	}

	if err := s.refreshTokenRepo.Revoke(ctx, rt.ID); err != nil {
		return nil, fmt.Errorf("revoking old token: %w", err)
	}

	tokens, err := s.generateTokenPair(ctx, rt.UserID, rt.DeviceID)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func (s *Service) Logout(ctx context.Context, userID uuid.UUID) error {
	if err := s.refreshTokenRepo.RevokeByUserID(ctx, userID); err != nil {
		return fmt.Errorf("revoking tokens: %w", err)
	}
	return nil
}

func (s *Service) LogoutDevice(ctx context.Context, userID uuid.UUID, deviceID string) error {
	device, err := s.deviceRepo.GetByUserAndDeviceID(ctx, userID, deviceID)
	if err != nil {
		return fmt.Errorf("getting device: %w", err)
	}

	if err := s.refreshTokenRepo.RevokeByDeviceID(ctx, device.ID); err != nil {
		return fmt.Errorf("revoking tokens: %w", err)
	}
	return nil
}

func (s *Service) generateTokenPair(ctx context.Context, userID, deviceID uuid.UUID) (*TokenPair, error) {
	accessToken, expiresAt, err := s.jwtSvc.GenerateAccessToken(userID)
	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	refreshTokenStr, err := s.jwtSvc.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}

	rt := entity.NewRefreshToken(
		userID,
		deviceID,
		refreshTokenStr,
		time.Now().UTC().Add(s.refreshTokenTTL),
	)

	if err := s.refreshTokenRepo.Create(ctx, rt); err != nil {
		return nil, fmt.Errorf("storing refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		ExpiresAt:    expiresAt,
	}, nil
}
