package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/marcos-nsantos/field-notes-backend/internal/domain"
)

type JWTService struct {
	secretKey      []byte
	accessTokenTTL time.Duration
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func NewJWTService(secretKey string, accessTokenTTL time.Duration) *JWTService {
	return &JWTService{
		secretKey:      []byte(secretKey),
		accessTokenTTL: accessTokenTTL,
	}
}

func (s *JWTService) GenerateAccessToken(userID uuid.UUID) (string, time.Time, error) {
	expiresAt := time.Now().UTC().Add(s.accessTokenTTL)

	claims := Claims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			NotBefore: jwt.NewNumericDate(time.Now().UTC()),
			Issuer:    "field-notes",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("signing token: %w", err)
	}

	return tokenStr, expiresAt, nil
}

func (s *JWTService) ValidateAccessToken(tokenStr string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})
	if err != nil {
		return uuid.Nil, domain.ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return uuid.Nil, domain.ErrTokenInvalid
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, domain.ErrTokenInvalid
	}

	return userID, nil
}

func (s *JWTService) GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
