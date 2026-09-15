package domain

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidUser     = errors.New("invalid user")
	ErrInvalidEmail    = errors.New("invalid user email")
	ErrInvalidPassword = errors.New("invalid user password")
	ErrInvalidPublicID = errors.New("invalid user public id")
)

// User is the Platform-owned global identity. PasswordHash is intentionally
// populated only for the credential-verification path and is never part of an
// authenticated request projection.
type User struct {
	ID           string
	Email        string
	DisplayName  string
	PasswordHash string
	Active       bool
	MFAEnabled   bool
	CreatedAt    time.Time
	LastLoginAt  *time.Time
}

func NewUserID() (string, error) {
	var raw [18]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate user id: %w", err)
	}
	return "usr_" + base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (u User) Validate() error {
	if !strings.HasPrefix(u.ID, "usr_") || len(u.ID) <= len("usr_") {
		return ErrInvalidPublicID
	}
	if !strings.Contains(NormalizeEmail(u.Email), "@") {
		return ErrInvalidEmail
	}
	if strings.TrimSpace(u.DisplayName) == "" {
		return ErrInvalidUser
	}
	return nil
}
