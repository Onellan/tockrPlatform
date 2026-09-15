package domain

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

type UserProductAssignmentStatus string

const (
	UserProductAssignmentActive  UserProductAssignmentStatus = "active"
	UserProductAssignmentRevoked UserProductAssignmentStatus = "revoked"
)

var (
	ErrInvalidProductAssignment       = errors.New("invalid user product assignment")
	ErrInvalidProductAssignmentID     = errors.New("invalid user product assignment id")
	ErrInvalidProductAssignmentStatus = errors.New("invalid user product assignment status")
)

type UserProductAssignment struct {
	ID               string
	UserID           string
	OrganisationID   string
	ProductKey       string
	Status           UserProductAssignmentStatus
	Active           bool
	EffectiveActive  bool
	AssignedBy       string
	AssignedAt       time.Time
	RevokedBy        string
	RevokedAt        *time.Time
	RevocationReason string
}

func NewUserProductAssignmentID() (string, error) {
	var raw [18]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate user product assignment id: %w", err)
	}
	return "upa_" + base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func (a UserProductAssignment) Validate() error {
	if !strings.HasPrefix(a.ID, "upa_") || len(a.ID) <= len("upa_") {
		return ErrInvalidProductAssignmentID
	}
	if !strings.HasPrefix(a.UserID, "usr_") || len(a.UserID) <= len("usr_") ||
		!strings.HasPrefix(a.OrganisationID, "org_") || len(a.OrganisationID) <= len("org_") ||
		!strings.HasPrefix(a.ProductKey, "product.") || len(a.ProductKey) <= len("product.") ||
		!strings.HasPrefix(a.AssignedBy, "usr_") || len(a.AssignedBy) <= len("usr_") {
		return ErrInvalidProductAssignment
	}
	if a.Status != UserProductAssignmentActive && a.Status != UserProductAssignmentRevoked {
		return ErrInvalidProductAssignmentStatus
	}
	if a.AssignedAt.IsZero() {
		return ErrInvalidProductAssignment
	}
	if a.Status == UserProductAssignmentRevoked &&
		(a.RevokedAt == nil || !strings.HasPrefix(a.RevokedBy, "usr_") || strings.TrimSpace(a.RevocationReason) == "") {
		return ErrInvalidProductAssignment
	}
	if a.Status == UserProductAssignmentActive && (a.RevokedAt != nil || a.RevokedBy != "" || a.RevocationReason != "") {
		return ErrInvalidProductAssignment
	}
	return nil
}
