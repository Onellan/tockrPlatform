package store

import (
	"context"
	"errors"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
)

var (
	ErrOrganisationNotFound           = errors.New("organisation not found")
	ErrOrganisationArchived           = errors.New("organisation is archived")
	ErrMembershipNotFound             = errors.New("organisation membership not found")
	ErrUnauthorisedOrganisationAction = errors.New("organisation action is not authorised")
	ErrOwnerMutationNotAuthorised     = errors.New("organisation owner mutation is not authorised")
	ErrDuplicateOrganisationMember    = errors.New("active organisation membership already exists")
)

type OrganisationMemberRecord struct {
	MembershipID   string
	OrganisationID string
	UserID         string
	Email          string
	DisplayName    string
	Role           domain.OrganisationRole
	AssignedAt     time.Time
}

type OrganisationAuditRecord struct {
	ID             int64
	OrganisationID string
	ActorUserID    string
	Event          string
	Details        string
	OccurredAt     time.Time
}

type OrganisationWorkspaceEntry struct {
	OrganisationID     string
	Resource           string
	Available          bool
	DefaultWorkspaceID string
	Workspaces         []domain.Workspace
}

// UserOrganisation is the minimum Organisation context needed by the
// Platform shell. It contains only Platform-owned identity and membership
// facts; product roles and product records remain outside this read model.
type UserOrganisation struct {
	Organisation domain.Organisation
	Role         domain.OrganisationRole
}

// OrganisationStore is the narrow Platform caller contract for shared
// Organisation authority. Product roles and product records do not appear in
// this contract.
type OrganisationStore interface {
	CreateOrganisation(context.Context, string, domain.Organisation, string, time.Time) (domain.Organisation, domain.OrganisationMembership, error)
	ListUserOrganisations(context.Context, string) ([]UserOrganisation, error)
	GetOrganisation(context.Context, string, string) (*domain.Organisation, error)
	GetOrganisationMembership(context.Context, string, string, string) (*domain.OrganisationMembership, error)
	AddOrganisationMember(context.Context, string, string, string, domain.OrganisationRole, string, time.Time) (domain.OrganisationMembership, error)
	ChangeOrganisationMemberRole(context.Context, string, string, string, domain.OrganisationRole, string, time.Time) (domain.OrganisationMembership, error)
	DeactivateOrganisationMember(context.Context, string, string, string, string, time.Time) error
	ArchiveOrganisation(context.Context, string, string, string, time.Time) error
	RenameOrganisation(context.Context, string, string, string, string, time.Time) (domain.Organisation, error)
	ListOrganisationMembers(context.Context, string, string) ([]OrganisationMemberRecord, error)
	ListOrganisationAudit(context.Context, string, string, int) ([]OrganisationAuditRecord, error)
	GetOrganisationWorkspaceEntry(context.Context, string, string) (OrganisationWorkspaceEntry, error)
}
