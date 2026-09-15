package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
)

func NewOpaqueToken(bytes int) (string, error) {
	if bytes < 32 {
		bytes = 32
	}
	raw := make([]byte, bytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate opaque token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func VerifyTokenHash(expectedHash, token string) bool {
	actual := HashToken(token)
	return subtle.ConstantTimeCompare([]byte(expectedHash), []byte(actual)) == 1
}
