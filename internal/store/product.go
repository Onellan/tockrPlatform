package store

import (
	"context"
	"errors"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
)

var (
	ErrProductNotFound                  = errors.New("product not found")
	ErrProductRetired                   = errors.New("product is retired")
	ErrEntitlementNotFound              = errors.New("organisation product entitlement not found")
	ErrEntitlementInactive              = errors.New("organisation product entitlement is inactive")
	ErrDuplicateOrganisationEntitlement = errors.New("active organisation product entitlement already exists")
	ErrUnauthorisedProductAction        = errors.New("product action is not authorised")
)

// ProductStore is the narrow Platform caller contract for catalogue and
// organisation-level entitlement authority. User assignments and effective
// product access are deliberately owned by the later PF-B5-S02 seam.
type ProductStore interface {
	ListProducts(context.Context, string) ([]domain.Product, error)
	RetireProduct(context.Context, string, string, string, time.Time) error
	EntitleOrganisation(context.Context, string, string, string, string, time.Time) (domain.OrganisationProductEntitlement, error)
	RevokeOrganisationEntitlement(context.Context, string, string, string, string, time.Time) error
	ListOrganisationProductEntitlements(context.Context, string, string) ([]domain.OrganisationProductEntitlement, error)
}
