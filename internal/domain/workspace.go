package domain

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

type WorkspaceStatus string

const (
	WorkspaceActive   WorkspaceStatus = "active"
	WorkspaceArchived WorkspaceStatus = "archived"
)

type WorkspaceRole string

const (
	WorkspaceAdmin  WorkspaceRole = "admin"
	WorkspaceMember WorkspaceRole = "member"
	WorkspaceViewer WorkspaceRole = "viewer"
)

var (
	ErrInvalidWorkspace       = errors.New("invalid workspace")
	ErrInvalidWorkspaceID     = errors.New("invalid workspace public id")
	ErrInvalidWorkspaceName   = errors.New("invalid workspace name")
	ErrInvalidWorkspaceStatus = errors.New("invalid workspace status")
	ErrInvalidWorkspaceRole   = errors.New("invalid workspace role")
	ErrInvalidWorkspaceMember = errors.New("invalid workspace membership")
)

type Workspace struct {
	ID             string
	OrganisationID string
	Name           string
	Status         WorkspaceStatus
	CreatedAt      time.Time
	ArchivedAt     *time.Time
}

type WorkspaceMembership struct {
	ID            string
	WorkspaceID   string
	UserID        string
	Role          WorkspaceRole
	Active        bool
	AssignedBy    string
	AssignedAt    time.Time
	RemovedBy     string
	RemovedAt     *time.Time
	RemovalReason string
}

func NewWorkspaceID() (string, error) {
	var raw [18]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate workspace id: %w", err)
	}
	return "wsp_" + base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func NewWorkspaceMembershipID() (string, error) {
	var raw [18]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate workspace membership id: %w", err)
	}
	return "wmem_" + base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func (w Workspace) Validate() error {
	if !strings.HasPrefix(w.ID, "wsp_") || len(w.ID) <= len("wsp_") {
		return ErrInvalidWorkspaceID
	}
	if !strings.HasPrefix(w.OrganisationID, "org_") || len(w.OrganisationID) <= len("org_") {
		return ErrInvalidWorkspace
	}
	if name := strings.TrimSpace(w.Name); name == "" || len(name) > 200 {
		return ErrInvalidWorkspaceName
	}
	if w.Status != WorkspaceActive && w.Status != WorkspaceArchived {
		return ErrInvalidWorkspaceStatus
	}
	if w.Status == WorkspaceArchived && w.ArchivedAt == nil {
		return ErrInvalidWorkspace
	}
	if w.CreatedAt.IsZero() {
		return ErrInvalidWorkspace
	}
	return nil
}

func (m WorkspaceMembership) Validate() error {
	if !strings.HasPrefix(m.ID, "wmem_") || len(m.ID) <= len("wmem_") {
		return ErrInvalidWorkspaceMember
	}
	if !strings.HasPrefix(m.WorkspaceID, "wsp_") || len(m.WorkspaceID) <= len("wsp_") {
		return ErrInvalidWorkspaceMember
	}
	if !strings.HasPrefix(m.UserID, "usr_") || len(m.UserID) <= len("usr_") {
		return ErrInvalidWorkspaceMember
	}
	if !strings.HasPrefix(m.AssignedBy, "usr_") || len(m.AssignedBy) <= len("usr_") {
		return ErrInvalidWorkspaceMember
	}
	if !m.Role.Valid() || m.AssignedAt.IsZero() {
		return ErrInvalidWorkspaceMember
	}
	if !m.Active && (m.RemovedAt == nil || !strings.HasPrefix(m.RemovedBy, "usr_") || strings.TrimSpace(m.RemovalReason) == "") {
		return ErrInvalidWorkspaceMember
	}
	return nil
}

func (r WorkspaceRole) Valid() bool {
	return r == WorkspaceAdmin || r == WorkspaceMember || r == WorkspaceViewer
}

func (r WorkspaceRole) CanAdminister() bool { return r == WorkspaceAdmin }
