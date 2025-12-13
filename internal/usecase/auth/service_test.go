package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/marcos-nsantos/field-notes-backend/internal/domain"
	"github.com/marcos-nsantos/field-notes-backend/internal/domain/entity"
	"github.com/marcos-nsantos/field-notes-backend/internal/infrastructure/auth"
	"github.com/marcos-nsantos/field-notes-backend/internal/mocks"
	authUC "github.com/marcos-nsantos/field-notes-backend/internal/usecase/auth"
)

func TestService_Register(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userRepo := mocks.NewMockUserRepository(ctrl)
		deviceRepo := mocks.NewMockDeviceRepository(ctrl)
		refreshTokenRepo := mocks.NewMockRefreshTokenRepository(ctrl)
		jwtSvc := auth.NewJWTService("test-secret", 15*time.Minute)
		passwordHasher := auth.NewPasswordHasher(4)

		svc := authUC.NewService(userRepo, deviceRepo, refreshTokenRepo, jwtSvc, passwordHasher, 24*time.Hour)

		ctx := context.Background()
		userRepo.EXPECT().ExistsByEmail(ctx, "test@example.com").Return(false, nil)
		userRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		user, err := svc.Register(ctx, authUC.RegisterInput{
			Email:    "test@example.com",
			Password: "password123",
			Name:     "Test User",
		})

		require.NoError(t, err)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "Test User", user.Name)
		assert.NotEmpty(t, user.PasswordHash)
	})

	t.Run("email already exists", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userRepo := mocks.NewMockUserRepository(ctrl)
		svc := authUC.NewService(userRepo, nil, nil, nil, nil, 0)

		ctx := context.Background()
		userRepo.EXPECT().ExistsByEmail(ctx, "existing@example.com").Return(true, nil)

		user, err := svc.Register(ctx, authUC.RegisterInput{
			Email:    "existing@example.com",
			Password: "password123",
			Name:     "Test User",
		})

		assert.Nil(t, user)
		assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
	})
}

func TestService_Login(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userRepo := mocks.NewMockUserRepository(ctrl)
		deviceRepo := mocks.NewMockDeviceRepository(ctrl)
		refreshTokenRepo := mocks.NewMockRefreshTokenRepository(ctrl)
		jwtSvc := auth.NewJWTService("test-secret", 15*time.Minute)
		passwordHasher := auth.NewPasswordHasher(4)

		svc := authUC.NewService(userRepo, deviceRepo, refreshTokenRepo, jwtSvc, passwordHasher, 24*time.Hour)

		ctx := context.Background()
		hashedPassword, _ := passwordHasher.Hash("password123")
		userID := uuid.New()
		deviceID := uuid.New()

		user := &entity.User{
			ID:           userID,
			Email:        "test@example.com",
			PasswordHash: hashedPassword,
			Name:         "Test User",
		}

		device := &entity.Device{
			ID:       deviceID,
			UserID:   userID,
			DeviceID: "device-123",
		}

		userRepo.EXPECT().GetByEmail(ctx, "test@example.com").Return(user, nil)
		deviceRepo.EXPECT().Upsert(ctx, gomock.Any()).Return(nil)
		deviceRepo.EXPECT().GetByUserAndDeviceID(ctx, userID, "device-123").Return(device, nil)
		refreshTokenRepo.EXPECT().RevokeByDeviceID(ctx, deviceID).Return(nil)
		refreshTokenRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		tokens, returnedUser, err := svc.Login(ctx, authUC.LoginInput{
			Email:      "test@example.com",
			Password:   "password123",
			DeviceID:   "device-123",
			DeviceName: "Test Phone",
			Platform:   "ios",
		})

		require.NoError(t, err)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
		assert.Equal(t, userID, returnedUser.ID)
	})

	t.Run("invalid email", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userRepo := mocks.NewMockUserRepository(ctrl)
		svc := authUC.NewService(userRepo, nil, nil, nil, nil, 0)

		ctx := context.Background()
		userRepo.EXPECT().GetByEmail(ctx, "notfound@example.com").Return(nil, domain.ErrUserNotFound)

		tokens, user, err := svc.Login(ctx, authUC.LoginInput{
			Email:    "notfound@example.com",
			Password: "password123",
		})

		assert.Nil(t, tokens)
		assert.Nil(t, user)
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("wrong password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userRepo := mocks.NewMockUserRepository(ctrl)
		passwordHasher := auth.NewPasswordHasher(4)
		svc := authUC.NewService(userRepo, nil, nil, nil, passwordHasher, 0)

		ctx := context.Background()
		hashedPassword, _ := passwordHasher.Hash("correctpassword")
		user := &entity.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			PasswordHash: hashedPassword,
		}

		userRepo.EXPECT().GetByEmail(ctx, "test@example.com").Return(user, nil)

		tokens, returnedUser, err := svc.Login(ctx, authUC.LoginInput{
			Email:    "test@example.com",
			Password: "wrongpassword",
		})

		assert.Nil(t, tokens)
		assert.Nil(t, returnedUser)
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})
}

func TestService_Refresh(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		refreshTokenRepo := mocks.NewMockRefreshTokenRepository(ctrl)
		jwtSvc := auth.NewJWTService("test-secret", 15*time.Minute)

		svc := authUC.NewService(nil, nil, refreshTokenRepo, jwtSvc, nil, 24*time.Hour)

		ctx := context.Background()
		userID := uuid.New()
		deviceID := uuid.New()
		tokenID := uuid.New()

		rt := &entity.RefreshToken{
			ID:        tokenID,
			UserID:    userID,
			DeviceID:  deviceID,
			Token:     "valid-refresh-token",
			ExpiresAt: time.Now().Add(24 * time.Hour),
			CreatedAt: time.Now(),
		}

		refreshTokenRepo.EXPECT().GetByToken(ctx, "valid-refresh-token").Return(rt, nil)
		refreshTokenRepo.EXPECT().Revoke(ctx, tokenID).Return(nil)
		refreshTokenRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		tokens, err := svc.Refresh(ctx, "valid-refresh-token")

		require.NoError(t, err)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
	})

	t.Run("expired token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		refreshTokenRepo := mocks.NewMockRefreshTokenRepository(ctrl)
		svc := authUC.NewService(nil, nil, refreshTokenRepo, nil, nil, 0)

		ctx := context.Background()
		rt := &entity.RefreshToken{
			ID:        uuid.New(),
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}

		refreshTokenRepo.EXPECT().GetByToken(ctx, "expired-token").Return(rt, nil)

		tokens, err := svc.Refresh(ctx, "expired-token")

		assert.Nil(t, tokens)
		assert.ErrorIs(t, err, domain.ErrTokenExpired)
	})

	t.Run("revoked token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		refreshTokenRepo := mocks.NewMockRefreshTokenRepository(ctrl)
		svc := authUC.NewService(nil, nil, refreshTokenRepo, nil, nil, 0)

		ctx := context.Background()
		revokedAt := time.Now()
		rt := &entity.RefreshToken{
			ID:        uuid.New(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
			RevokedAt: &revokedAt,
		}

		refreshTokenRepo.EXPECT().GetByToken(ctx, "revoked-token").Return(rt, nil)

		tokens, err := svc.Refresh(ctx, "revoked-token")

		assert.Nil(t, tokens)
		assert.ErrorIs(t, err, domain.ErrTokenRevoked)
	})

	t.Run("invalid token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		refreshTokenRepo := mocks.NewMockRefreshTokenRepository(ctrl)
		svc := authUC.NewService(nil, nil, refreshTokenRepo, nil, nil, 0)

		ctx := context.Background()
		refreshTokenRepo.EXPECT().GetByToken(ctx, "invalid-token").Return(nil, errors.New("not found"))

		tokens, err := svc.Refresh(ctx, "invalid-token")

		assert.Nil(t, tokens)
		assert.ErrorIs(t, err, domain.ErrTokenInvalid)
	})
}

func TestService_Logout(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		refreshTokenRepo := mocks.NewMockRefreshTokenRepository(ctrl)
		svc := authUC.NewService(nil, nil, refreshTokenRepo, nil, nil, 0)

		ctx := context.Background()
		userID := uuid.New()
		refreshTokenRepo.EXPECT().RevokeByUserID(ctx, userID).Return(nil)

		err := svc.Logout(ctx, userID)

		require.NoError(t, err)
	})
}
