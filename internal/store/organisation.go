package store

import (
	"context"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
)

// OrganisationStore is the narrow Platform caller contract for shared
// Organisation authority. Product roles and product records do not appear in
// this contract.
type OrganisationStore interface {
	CreateOrganisation(context.Context, string, domain.Organisation, string, time.Time) (domain.Organisation, domain.OrganisationMembership, error)
	GetOrganisation(context.Context, string, string) (*domain.Organisation, error)
	GetOrganisationMembership(context.Context, string, string, string) (*domain.OrganisationMembership, error)
	AddOrganisationMember(context.Context, string, string, string, domain.OrganisationRole, string, time.Time) (domain.OrganisationMembership, error)
	ChangeOrganisationMemberRole(context.Context, string, string, string, domain.OrganisationRole, string, time.Time) (domain.OrganisationMembership, error)
	DeactivateOrganisationMember(context.Context, string, string, string, string, time.Time) error
	ArchiveOrganisation(context.Context, string, string, string, time.Time) error
}
