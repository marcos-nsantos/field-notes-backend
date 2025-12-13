package domain

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNoteNotFound       = errors.New("note not found")
	ErrPhotoNotFound      = errors.New("photo not found")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrTokenExpired       = errors.New("token expired")
	ErrTokenInvalid       = errors.New("token invalid")
	ErrTokenRevoked       = errors.New("token revoked")
	ErrDeviceNotFound     = errors.New("device not found")
	ErrInvalidBoundingBox = errors.New("invalid bounding box")
	ErrInvalidLocation    = errors.New("invalid location")
)
