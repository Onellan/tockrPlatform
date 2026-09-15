package auth

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultPasswordCost is deliberately explicit so policy changes are
	// reviewable and can be applied to new credentials without weakening old
	// hashes.
	DefaultPasswordCost = 12

	// DummyPasswordHash equalises unknown-user and known-user verification.
	// It contains no usable Platform credential.
	DummyPasswordHash = "$2a$12$1Z3c.OVxnh87fotws1nTxONdNrj6noCRzU7ILd46d7eJJyt2FLsJe"
)

var ErrPasswordPolicy = errors.New("password does not meet Platform policy")

func ValidatePassword(password string) error {
	if len([]rune(password)) < 12 || strings.TrimSpace(password) == "" {
		return ErrPasswordPolicy
	}
	return nil
}

func HashPassword(password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), DefaultPasswordCost)
	return string(hash), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
