package domain

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

type OrganisationStatus string

const (
	OrganisationActive   OrganisationStatus = "active"
	OrganisationArchived OrganisationStatus = "archived"
)

type OrganisationRole string

const (
	OrganisationOwner  OrganisationRole = "owner"
	OrganisationAdmin  OrganisationRole = "admin"
	OrganisationMember OrganisationRole = "member"
)

var (
	ErrInvalidOrganisation       = errors.New("invalid organisation")
	ErrInvalidOrganisationName   = errors.New("invalid organisation name")
	ErrInvalidOrganisationID     = errors.New("invalid organisation public id")
	ErrInvalidOrganisationStatus = errors.New("invalid organisation status")
	ErrInvalidOrganisationRole   = errors.New("invalid organisation role")
	ErrInvalidMembership         = errors.New("invalid organisation membership")
	ErrInvalidReason             = errors.New("organisation mutation reason is required")
)

type Organisation struct {
	ID         string
	Name       string
	Status     OrganisationStatus
	CreatedAt  time.Time
	ArchivedAt *time.Time
}

type OrganisationMembership struct {
	ID             string
	OrganisationID string
	UserID         string
	Role           OrganisationRole
	Active         bool
	AssignedBy     string
	AssignedAt     time.Time
	RemovedBy      string
	RemovedAt      *time.Time
	RemovalReason  string
}

func NewOrganisationID() (string, error) {
	var raw [18]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate organisation id: %w", err)
	}
	return "org_" + base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func NewOrganisationMembershipID() (string, error) {
	var raw [18]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate organisation membership id: %w", err)
	}
	return "omem_" + base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func (o Organisation) Validate() error {
	if !strings.HasPrefix(o.ID, "org_") || len(o.ID) <= len("org_") {
		return ErrInvalidOrganisationID
	}
	if name := strings.TrimSpace(o.Name); name == "" || len(name) > 200 {
		return ErrInvalidOrganisationName
	}
	if o.Status != OrganisationActive && o.Status != OrganisationArchived {
		return ErrInvalidOrganisationStatus
	}
	if o.Status == OrganisationArchived && o.ArchivedAt == nil {
		return ErrInvalidOrganisation
	}
	return nil
}

func (m OrganisationMembership) Validate() error {
	if !strings.HasPrefix(m.ID, "omem_") || len(m.ID) <= len("omem_") {
		return ErrInvalidMembership
	}
	if !strings.HasPrefix(m.OrganisationID, "org_") || len(m.OrganisationID) <= len("org_") {
		return ErrInvalidMembership
	}
	if !strings.HasPrefix(m.UserID, "usr_") || len(m.UserID) <= len("usr_") {
		return ErrInvalidMembership
	}
	if !strings.HasPrefix(m.AssignedBy, "usr_") || len(m.AssignedBy) <= len("usr_") {
		return ErrInvalidMembership
	}
	if m.Role != OrganisationOwner && m.Role != OrganisationAdmin && m.Role != OrganisationMember {
		return ErrInvalidOrganisationRole
	}
	if m.AssignedAt.IsZero() {
		return ErrInvalidMembership
	}
	if !m.Active {
		if m.RemovedAt == nil || strings.TrimSpace(m.RemovedBy) == "" || strings.TrimSpace(m.RemovalReason) == "" {
			return ErrInvalidMembership
		}
	}
	return nil
}

func (r OrganisationRole) Valid() bool {
	return r == OrganisationOwner || r == OrganisationAdmin || r == OrganisationMember
}

func (r OrganisationRole) CanAdminister() bool {
	return r == OrganisationOwner || r == OrganisationAdmin
}
