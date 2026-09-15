package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestUnknownUserDummyUsesConfiguredPasswordCost(t *testing.T) {
	cost, err := bcrypt.Cost([]byte(DummyPasswordHash))
	if err != nil {
		t.Fatalf("dummy password hash is invalid: %v", err)
	}
	if cost != DefaultPasswordCost {
		t.Fatalf("dummy password cost = %d, want %d", cost, DefaultPasswordCost)
	}
	if CheckPassword(DummyPasswordHash, "not-the-dummy-password") {
		t.Fatal("dummy password unexpectedly authenticates")
	}
}
