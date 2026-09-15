package store

import (
	"context"
	"errors"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
)

var (
	ErrWorkspaceNotFound           = errors.New("workspace not found")
	ErrWorkspaceArchived           = errors.New("workspace is archived")
	ErrWorkspaceMembershipNotFound = errors.New("workspace membership not found")
	ErrUnauthorisedWorkspaceAction = errors.New("workspace action is not authorised")
	ErrDuplicateWorkspaceMember    = errors.New("active workspace membership already exists")
)

type WorkspaceMemberRecord struct {
	MembershipID string
	WorkspaceID  string
	UserID       string
	Email        string
	DisplayName  string
	Role         domain.WorkspaceRole
	AssignedAt   time.Time
}

type WorkspaceAuditRecord struct {
	ID          int64
	WorkspaceID string
	ActorUserID string
	Event       string
	Details     string
	OccurredAt  time.Time
}

type WorkspaceStore interface {
	CreateWorkspace(context.Context, string, string, domain.Workspace, string, time.Time) (domain.Workspace, domain.WorkspaceMembership, error)
	GetWorkspace(context.Context, string, string) (*domain.Workspace, error)
	GetWorkspaceMembership(context.Context, string, string, string) (*domain.WorkspaceMembership, error)
	ListOrganisationWorkspaces(context.Context, string, string) ([]domain.Workspace, error)
	AddWorkspaceMember(context.Context, string, string, string, domain.WorkspaceRole, string, time.Time) (domain.WorkspaceMembership, error)
	ChangeWorkspaceMemberRole(context.Context, string, string, string, domain.WorkspaceRole, string, time.Time) (domain.WorkspaceMembership, error)
	DeactivateWorkspaceMember(context.Context, string, string, string, string, time.Time) error
	ArchiveWorkspace(context.Context, string, string, string, time.Time) error
	ListWorkspaceMembers(context.Context, string, string) ([]WorkspaceMemberRecord, error)
	ListWorkspaceAudit(context.Context, string, string, int) ([]WorkspaceAuditRecord, error)
}
