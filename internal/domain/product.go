package domain

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ProductStatus string

const (
	ProductActive  ProductStatus = "active"
	ProductRetired ProductStatus = "retired"
)

type OrganisationProductEntitlementStatus string

const (
	OrganisationProductEntitlementActive  OrganisationProductEntitlementStatus = "active"
	OrganisationProductEntitlementRevoked OrganisationProductEntitlementStatus = "revoked"
)

var (
	ErrInvalidProduct           = errors.New("invalid product")
	ErrInvalidProductKey        = errors.New("invalid product key")
	ErrInvalidProductStatus     = errors.New("invalid product status")
	ErrInvalidEntitlement       = errors.New("invalid organisation product entitlement")
	ErrInvalidEntitlementID     = errors.New("invalid organisation product entitlement id")
	ErrInvalidEntitlementStatus = errors.New("invalid organisation product entitlement status")
)

type Product struct {
	Key         string
	DisplayName string
	Status      ProductStatus
	CreatedAt   time.Time
	RetiredAt   *time.Time
}

type OrganisationProductEntitlement struct {
	ID               string
	OrganisationID   string
	ProductKey       string
	Status           OrganisationProductEntitlementStatus
	Active           bool
	GrantedBy        string
	GrantedAt        time.Time
	RevokedBy        string
	RevokedAt        *time.Time
	RevocationReason string
}

func NewOrganisationProductEntitlementID() (string, error) {
	var raw [18]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate organisation product entitlement id: %w", err)
	}
	return "ent_" + base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func (p Product) Validate() error {
	if !strings.HasPrefix(p.Key, "product.") || len(p.Key) <= len("product.") {
		return ErrInvalidProductKey
	}
	if strings.TrimSpace(p.DisplayName) == "" || len(p.DisplayName) > 200 {
		return ErrInvalidProduct
	}
	if p.Status != ProductActive && p.Status != ProductRetired {
		return ErrInvalidProductStatus
	}
	if p.Status == ProductRetired && p.RetiredAt == nil {
		return ErrInvalidProduct
	}
	if p.Status == ProductActive && p.RetiredAt != nil {
		return ErrInvalidProduct
	}
	if p.CreatedAt.IsZero() {
		return ErrInvalidProduct
	}
	return nil
}

func (e OrganisationProductEntitlement) Validate() error {
	if !strings.HasPrefix(e.ID, "ent_") || len(e.ID) <= len("ent_") {
		return ErrInvalidEntitlementID
	}
	if !strings.HasPrefix(e.OrganisationID, "org_") || len(e.OrganisationID) <= len("org_") ||
		!strings.HasPrefix(e.ProductKey, "product.") || len(e.ProductKey) <= len("product.") ||
		!strings.HasPrefix(e.GrantedBy, "usr_") || len(e.GrantedBy) <= len("usr_") {
		return ErrInvalidEntitlement
	}
	if e.Status != OrganisationProductEntitlementActive && e.Status != OrganisationProductEntitlementRevoked {
		return ErrInvalidEntitlementStatus
	}
	if e.GrantedAt.IsZero() {
		return ErrInvalidEntitlement
	}
	if e.Status == OrganisationProductEntitlementRevoked &&
		(e.RevokedAt == nil || !strings.HasPrefix(e.RevokedBy, "usr_") || strings.TrimSpace(e.RevocationReason) == "") {
		return ErrInvalidEntitlement
	}
	if e.Status == OrganisationProductEntitlementActive && (e.RevokedAt != nil || e.RevokedBy != "" || e.RevocationReason != "") {
		return ErrInvalidEntitlement
	}
	return nil
}
