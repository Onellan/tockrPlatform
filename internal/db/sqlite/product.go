package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
	"github.com/Onellan/tockrplatform/internal/store"
)

var (
	ErrProductNotFound                  = store.ErrProductNotFound
	ErrProductRetired                   = store.ErrProductRetired
	ErrEntitlementNotFound              = store.ErrEntitlementNotFound
	ErrEntitlementInactive              = store.ErrEntitlementInactive
	ErrDuplicateOrganisationEntitlement = store.ErrDuplicateOrganisationEntitlement
	ErrUnauthorisedProductAction        = store.ErrUnauthorisedProductAction
)

const (
	auditProduct            = "Product"
	eventProductRetired     = "product_retired"
	eventEntitlementGranted = "product_entitlement_granted"
	eventEntitlementRevoked = "product_entitlement_revoked"
)

type productAuditDetails struct {
	ProductKey     string `json:"product_key"`
	OrganisationID string `json:"organisation_id,omitempty"`
	EntitlementID  string `json:"entitlement_id,omitempty"`
	AssignmentID   string `json:"assignment_id,omitempty"`
	Status         string `json:"status,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

func (s *Store) ListProducts(ctx context.Context, requesterUserID string) ([]domain.Product, error) {
	if err := s.requireSystemAdmin(ctx, requesterUserID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT product_key,display_name,status,created_at,retired_at FROM products ORDER BY product_key`)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()
	products := make([]domain.Product, 0, 2)
	for rows.Next() {
		var product domain.Product
		var status, created string
		var retired sql.NullString
		if err := rows.Scan(&product.Key, &product.DisplayName, &status, &created, &retired); err != nil {
			return nil, fmt.Errorf("read product: %w", err)
		}
		product.Status = domain.ProductStatus(status)
		product.CreatedAt = parseTime(created)
		if retired.Valid {
			value := parseTime(retired.String)
			product.RetiredAt = &value
		}
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}
	return products, nil
}

func (s *Store) RetireProduct(ctx context.Context, actorUserID, productKey, reason string, at time.Time) error {
	productKey = strings.TrimSpace(productKey)
	if !strings.HasPrefix(productKey, "product.") || len(productKey) <= len("product.") {
		return domain.ErrInvalidProductKey
	}
	if err := validateMutationReason(reason); err != nil {
		return err
	}
	if at.IsZero() {
		return errors.New("product retirement time is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin product retirement: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	actorInternalID, err := systemAdminInternalTx(ctx, tx, actorUserID)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE products SET status='retired',retired_at=? WHERE product_key=? AND status='active'`, formatTime(at.UTC()), productKey)
	if err != nil {
		return fmt.Errorf("retire product: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("check product retirement: %w", err)
	} else if rows == 0 {
		var status string
		if err := tx.QueryRowContext(ctx, `SELECT status FROM products WHERE product_key=?`, productKey).Scan(&status); errors.Is(err, sql.ErrNoRows) {
			return ErrProductNotFound
		} else if err != nil {
			return fmt.Errorf("read product retirement state: %w", err)
		} else if status == string(domain.ProductRetired) {
			return ErrProductRetired
		}
		return ErrProductNotFound
	}
	if err := recordProductAuditTx(ctx, tx, actorInternalID, productKey, eventProductRetired, productAuditDetails{ProductKey: productKey, Status: string(domain.ProductRetired), Reason: strings.TrimSpace(reason)}, at); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit product retirement: %w", err)
	}
	return nil
}

func (s *Store) EntitleOrganisation(ctx context.Context, actorUserID, organisationID, productKey, reason string, at time.Time) (domain.OrganisationProductEntitlement, error) {
	productKey = strings.TrimSpace(productKey)
	if !strings.HasPrefix(productKey, "product.") || len(productKey) <= len("product.") {
		return domain.OrganisationProductEntitlement{}, domain.ErrInvalidProductKey
	}
	if err := validateMutationReason(reason); err != nil {
		return domain.OrganisationProductEntitlement{}, err
	}
	if at.IsZero() {
		return domain.OrganisationProductEntitlement{}, errors.New("product entitlement time is required")
	}
	entitlementID, err := newEntitlementID()
	if err != nil {
		return domain.OrganisationProductEntitlement{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.OrganisationProductEntitlement{}, fmt.Errorf("begin product entitlement: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	actorInternalID, _, organisationInternalID, _, err := authorisedOrganisationMutationTx(ctx, tx, actorUserID, organisationID)
	if err != nil {
		return domain.OrganisationProductEntitlement{}, err
	}
	var productStatus string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM products WHERE product_key=?`, productKey).Scan(&productStatus); errors.Is(err, sql.ErrNoRows) {
		return domain.OrganisationProductEntitlement{}, ErrProductNotFound
	} else if err != nil {
		return domain.OrganisationProductEntitlement{}, fmt.Errorf("resolve product entitlement product: %w", err)
	}
	if productStatus != string(domain.ProductActive) {
		return domain.OrganisationProductEntitlement{}, ErrProductRetired
	}
	var active int
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM organisation_product_entitlements WHERE organisation_id=? AND product_key=? AND status='active')`, organisationInternalID, productKey).Scan(&active); err != nil {
		return domain.OrganisationProductEntitlement{}, fmt.Errorf("check product entitlement duplicate: %w", err)
	}
	if active == 1 {
		return domain.OrganisationProductEntitlement{}, ErrDuplicateOrganisationEntitlement
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO organisation_product_entitlements(public_id,organisation_id,product_key,status,granted_by,granted_at) VALUES(?,?,?,'active',?,?)`, entitlementID, organisationInternalID, productKey, actorInternalID, formatTime(at.UTC())); err != nil {
		return domain.OrganisationProductEntitlement{}, fmt.Errorf("create product entitlement: %w", err)
	}
	if err := recordOrganisationProductAuditTx(ctx, tx, actorInternalID, organisationID, eventEntitlementGranted, productAuditDetails{ProductKey: productKey, OrganisationID: organisationID, EntitlementID: entitlementID, Status: string(domain.OrganisationProductEntitlementActive), Reason: strings.TrimSpace(reason)}, at); err != nil {
		return domain.OrganisationProductEntitlement{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.OrganisationProductEntitlement{}, fmt.Errorf("commit product entitlement: %w", err)
	}
	return domain.OrganisationProductEntitlement{ID: entitlementID, OrganisationID: organisationID, ProductKey: productKey, Status: domain.OrganisationProductEntitlementActive, Active: true, GrantedBy: actorUserID, GrantedAt: at.UTC()}, nil
}

func (s *Store) RevokeOrganisationEntitlement(ctx context.Context, actorUserID, organisationID, entitlementID, reason string, at time.Time) error {
	if err := validateMutationReason(reason); err != nil {
		return err
	}
	if at.IsZero() {
		return errors.New("product entitlement revocation time is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin product entitlement revocation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var entitlementInternalID int64
	var productKey, status string
	if err := tx.QueryRowContext(ctx, `SELECT e.id,e.product_key,e.status FROM organisation_product_entitlements e JOIN organisations o ON o.id=e.organisation_id WHERE o.public_id=? AND e.public_id=?`, organisationID, strings.TrimSpace(entitlementID)).Scan(&entitlementInternalID, &productKey, &status); errors.Is(err, sql.ErrNoRows) {
		return ErrEntitlementNotFound
	} else if err != nil {
		return fmt.Errorf("resolve product entitlement revocation: %w", err)
	}
	actorInternalID, _, _, _, err := authorisedOrganisationMutationTx(ctx, tx, actorUserID, organisationID)
	if err != nil {
		return err
	}
	if status != string(domain.OrganisationProductEntitlementActive) {
		return ErrEntitlementInactive
	}
	if _, err := tx.ExecContext(ctx, `UPDATE organisation_product_entitlements SET status='revoked',revoked_by=?,revoked_at=?,revocation_reason=? WHERE id=? AND status='active'`, actorInternalID, formatTime(at.UTC()), strings.TrimSpace(reason), entitlementInternalID); err != nil {
		return fmt.Errorf("revoke product entitlement: %w", err)
	}
	if err := recordOrganisationProductAuditTx(ctx, tx, actorInternalID, organisationID, eventEntitlementRevoked, productAuditDetails{ProductKey: productKey, OrganisationID: organisationID, EntitlementID: entitlementID, Status: string(domain.OrganisationProductEntitlementRevoked), Reason: strings.TrimSpace(reason)}, at); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit product entitlement revocation: %w", err)
	}
	return nil
}

func (s *Store) ListOrganisationProductEntitlements(ctx context.Context, requesterUserID, organisationID string) ([]domain.OrganisationProductEntitlement, error) {
	organisationInternalID, err := s.authorisedOrganisationRead(ctx, requesterUserID, organisationID, true)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT e.public_id,o.public_id,e.product_key,e.status,
		(e.status='active' AND o.status='active' AND p.status='active'),granted.public_id,e.granted_at,
		revoked.public_id,e.revoked_at,e.revocation_reason
		FROM organisation_product_entitlements e
		JOIN organisations o ON o.id=e.organisation_id
		JOIN products p ON p.product_key=e.product_key
		JOIN users granted ON granted.id=e.granted_by
		LEFT JOIN users revoked ON revoked.id=e.revoked_by
		WHERE e.organisation_id=? ORDER BY e.granted_at,e.id`, organisationInternalID)
	if err != nil {
		return nil, fmt.Errorf("list organisation product entitlements: %w", err)
	}
	defer rows.Close()
	entitlements := make([]domain.OrganisationProductEntitlement, 0)
	for rows.Next() {
		var entitlement domain.OrganisationProductEntitlement
		var status, grantedAt string
		var active int
		var revokedBy, revokedAt, revocationReason sql.NullString
		if err := rows.Scan(&entitlement.ID, &entitlement.OrganisationID, &entitlement.ProductKey, &status, &active, &entitlement.GrantedBy, &grantedAt, &revokedBy, &revokedAt, &revocationReason); err != nil {
			return nil, fmt.Errorf("read organisation product entitlement: %w", err)
		}
		entitlement.Status = domain.OrganisationProductEntitlementStatus(status)
		entitlement.Active = active == 1
		entitlement.GrantedAt = parseTime(grantedAt)
		if revokedBy.Valid {
			entitlement.RevokedBy = revokedBy.String
		}
		if revokedAt.Valid {
			value := parseTime(revokedAt.String)
			entitlement.RevokedAt = &value
		}
		entitlement.RevocationReason = revocationReason.String
		entitlements = append(entitlements, entitlement)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate organisation product entitlements: %w", err)
	}
	return entitlements, nil
}

func (s *Store) requireSystemAdmin(ctx context.Context, requesterUserID string) error {
	var active int
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users u JOIN system_role_assignments sra ON sra.user_id=u.id AND sra.role='system_admin' AND sra.active=1 WHERE u.public_id=? AND u.active=1)`, strings.TrimSpace(requesterUserID)).Scan(&active); err != nil {
		return fmt.Errorf("resolve product system authority: %w", err)
	}
	if active != 1 {
		return ErrUnauthorisedProductAction
	}
	return nil
}

func systemAdminInternalTx(ctx context.Context, tx *sql.Tx, actorUserID string) (int64, error) {
	var actorInternalID int64
	if err := tx.QueryRowContext(ctx, `SELECT u.id FROM users u JOIN system_role_assignments sra ON sra.user_id=u.id AND sra.role='system_admin' AND sra.active=1 WHERE u.public_id=? AND u.active=1`, strings.TrimSpace(actorUserID)).Scan(&actorInternalID); errors.Is(err, sql.ErrNoRows) {
		return 0, ErrUnauthorisedProductAction
	} else if err != nil {
		return 0, fmt.Errorf("resolve product system actor: %w", err)
	}
	return actorInternalID, nil
}

func recordProductAuditTx(ctx context.Context, tx *sql.Tx, actorInternalID int64, productKey, event string, details productAuditDetails, at time.Time) error {
	payload, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("encode product audit: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_events(actor_user_id,aggregate_type,aggregate_id,event,details,occurred_at) VALUES(?,?,?,?,?,?)`, actorInternalID, auditProduct, productKey, event, string(payload), formatTime(at.UTC())); err != nil {
		return fmt.Errorf("record product audit: %w", err)
	}
	return nil
}

func recordOrganisationProductAuditTx(ctx context.Context, tx *sql.Tx, actorInternalID int64, organisationID, event string, details productAuditDetails, at time.Time) error {
	payload, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("encode organisation product audit: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_events(actor_user_id,aggregate_type,aggregate_id,event,details,occurred_at) VALUES(?,?,?,?,?,?)`, actorInternalID, auditOrganisation, organisationID, event, string(payload), formatTime(at.UTC())); err != nil {
		return fmt.Errorf("record organisation product audit: %w", err)
	}
	return nil
}

func newEntitlementID() (string, error) {
	return domain.NewOrganisationProductEntitlementID()
}
