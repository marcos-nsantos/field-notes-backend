package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type PasswordHasher struct {
	cost int
}

func NewPasswordHasher(cost int) *PasswordHasher {
	if cost < bcrypt.MinCost {
		cost = bcrypt.DefaultCost
	}
	return &PasswordHasher{cost: cost}
}

func (h *PasswordHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}
	return string(hash), nil
}

func (h *PasswordHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
