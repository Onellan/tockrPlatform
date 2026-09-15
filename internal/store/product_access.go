package store

import (
	"context"
	"errors"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
)

var (
	ErrProductAssignmentNotFound      = errors.New("user product assignment not found")
	ErrProductAssignmentInactive      = errors.New("user product assignment is inactive")
	ErrDuplicateUserProductAssignment = errors.New("active user product assignment already exists")
	ErrProductAccessDenied            = errors.New("product access is not authorised")
)

type ProductAccess struct {
	UserID           string
	OrganisationID   string
	ProductKey       string
	WorkspaceID      string
	OrganisationRole domain.OrganisationRole
	WorkspaceRole    domain.WorkspaceRole
}

// ProductAccessStore owns the assignment lifecycle and the single shared
// effective-access predicate. Product-specific roles remain outside this
// contract and are evaluated by the product after Platform access succeeds.
type ProductAccessStore interface {
	AssignUserProduct(context.Context, string, string, string, string, string, time.Time) (domain.UserProductAssignment, error)
	RevokeUserProduct(context.Context, string, string, string, string, time.Time) error
	ListUserProductAssignments(context.Context, string, string) ([]domain.UserProductAssignment, error)
	ProveProductAccess(context.Context, string, string, string, string) (ProductAccess, error)
}
