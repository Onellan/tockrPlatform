package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
	"github.com/Onellan/tockrplatform/internal/store"
)

var (
	ErrProductAssignmentNotFound      = store.ErrProductAssignmentNotFound
	ErrProductAssignmentInactive      = store.ErrProductAssignmentInactive
	ErrDuplicateUserProductAssignment = store.ErrDuplicateUserProductAssignment
	ErrProductAccessDenied            = store.ErrProductAccessDenied
)

const (
	eventAssignmentGranted = "product_assignment_granted"
	eventAssignmentRevoked = "product_assignment_revoked"
)

func (s *Store) AssignUserProduct(ctx context.Context, actorUserID, organisationID, targetUserID, productKey, reason string, at time.Time) (domain.UserProductAssignment, error) {
	targetUserID = strings.TrimSpace(targetUserID)
	productKey = strings.TrimSpace(productKey)
	if !validProductKey(productKey) {
		return domain.UserProductAssignment{}, domain.ErrInvalidProductKey
	}
	if err := validateMutationReason(reason); err != nil {
		return domain.UserProductAssignment{}, err
	}
	if at.IsZero() {
		return domain.UserProductAssignment{}, errors.New("user product assignment time is required")
	}
	assignmentID, err := domain.NewUserProductAssignmentID()
	if err != nil {
		return domain.UserProductAssignment{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.UserProductAssignment{}, fmt.Errorf("begin user product assignment: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	actorInternalID, _, organisationInternalID, _, err := authorisedOrganisationMutationTx(ctx, tx, actorUserID, organisationID)
	if err != nil {
		return domain.UserProductAssignment{}, err
	}
	var productStatus string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM products WHERE product_key=?`, productKey).Scan(&productStatus); errors.Is(err, sql.ErrNoRows) {
		return domain.UserProductAssignment{}, ErrProductNotFound
	} else if err != nil {
		return domain.UserProductAssignment{}, fmt.Errorf("resolve assignment product: %w", err)
	}
	if productStatus != string(domain.ProductActive) {
		return domain.UserProductAssignment{}, ErrProductRetired
	}
	var targetInternalID int64
	if err := tx.QueryRowContext(ctx, `SELECT u.id FROM users u JOIN organisation_memberships m ON m.user_id=u.id AND m.organisation_id=? AND m.active=1 WHERE u.public_id=? AND u.active=1`, organisationInternalID, strings.TrimSpace(targetUserID)).Scan(&targetInternalID); errors.Is(err, sql.ErrNoRows) {
		return domain.UserProductAssignment{}, ErrUnauthorisedOrganisationAction
	} else if err != nil {
		return domain.UserProductAssignment{}, fmt.Errorf("resolve assignment target: %w", err)
	}
	var active int
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM user_product_assignments WHERE user_id=? AND organisation_id=? AND product_key=? AND status='active')`, targetInternalID, organisationInternalID, productKey).Scan(&active); err != nil {
		return domain.UserProductAssignment{}, fmt.Errorf("check assignment duplicate: %w", err)
	}
	if active == 1 {
		return domain.UserProductAssignment{}, ErrDuplicateUserProductAssignment
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO user_product_assignments(public_id,user_id,organisation_id,product_key,status,assigned_by,assigned_at) VALUES(?,?,?,?,'active',?,?)`, assignmentID, targetInternalID, organisationInternalID, productKey, actorInternalID, formatTime(at.UTC())); err != nil {
		return domain.UserProductAssignment{}, fmt.Errorf("create user product assignment: %w", err)
	}
	var effectiveActive int
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM organisation_product_entitlements WHERE organisation_id=? AND product_key=? AND status='active')`, organisationInternalID, productKey).Scan(&effectiveActive); err != nil {
		return domain.UserProductAssignment{}, fmt.Errorf("check assignment effective entitlement: %w", err)
	}
	if err := recordOrganisationProductAuditTx(ctx, tx, actorInternalID, organisationID, eventAssignmentGranted, productAuditDetails{ProductKey: productKey, OrganisationID: organisationID, AssignmentID: assignmentID, Status: string(domain.UserProductAssignmentActive), Reason: strings.TrimSpace(reason)}, at); err != nil {
		return domain.UserProductAssignment{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.UserProductAssignment{}, fmt.Errorf("commit user product assignment: %w", err)
	}
	return domain.UserProductAssignment{ID: assignmentID, UserID: targetUserID, OrganisationID: organisationID, ProductKey: productKey, Status: domain.UserProductAssignmentActive, Active: true, EffectiveActive: effectiveActive == 1, AssignedBy: actorUserID, AssignedAt: at.UTC()}, nil
}

func (s *Store) RevokeUserProduct(ctx context.Context, actorUserID, organisationID, assignmentID, reason string, at time.Time) error {
	if err := validateMutationReason(reason); err != nil {
		return err
	}
	if at.IsZero() {
		return errors.New("user product assignment revocation time is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin user product assignment revocation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var assignmentInternalID int64
	var productKey, status string
	if err := tx.QueryRowContext(ctx, `SELECT a.id,a.product_key,a.status FROM user_product_assignments a JOIN organisations o ON o.id=a.organisation_id WHERE o.public_id=? AND a.public_id=?`, strings.TrimSpace(organisationID), strings.TrimSpace(assignmentID)).Scan(&assignmentInternalID, &productKey, &status); errors.Is(err, sql.ErrNoRows) {
		return ErrProductAssignmentNotFound
	} else if err != nil {
		return fmt.Errorf("resolve user product assignment revocation: %w", err)
	}
	actorInternalID, _, _, _, err := authorisedOrganisationMutationTx(ctx, tx, actorUserID, organisationID)
	if err != nil {
		return err
	}
	if status != string(domain.UserProductAssignmentActive) {
		return ErrProductAssignmentInactive
	}
	if _, err := tx.ExecContext(ctx, `UPDATE user_product_assignments SET status='revoked',revoked_by=?,revoked_at=?,revocation_reason=? WHERE id=? AND status='active'`, actorInternalID, formatTime(at.UTC()), strings.TrimSpace(reason), assignmentInternalID); err != nil {
		return fmt.Errorf("revoke user product assignment: %w", err)
	}
	if err := recordOrganisationProductAuditTx(ctx, tx, actorInternalID, organisationID, eventAssignmentRevoked, productAuditDetails{ProductKey: productKey, OrganisationID: organisationID, AssignmentID: assignmentID, Status: string(domain.UserProductAssignmentRevoked), Reason: strings.TrimSpace(reason)}, at); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit user product assignment revocation: %w", err)
	}
	return nil
}

func (s *Store) ListUserProductAssignments(ctx context.Context, requesterUserID, organisationID string) ([]domain.UserProductAssignment, error) {
	organisationInternalID, err := s.authorisedOrganisationRead(ctx, requesterUserID, organisationID, true)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT a.public_id,u.public_id,o.public_id,a.product_key,a.status,
		(a.status='active' AND u.active=1 AND o.status='active' AND p.status='active' AND
		 EXISTS(SELECT 1 FROM organisation_memberships m WHERE m.organisation_id=a.organisation_id AND m.user_id=a.user_id AND m.active=1) AND
		 EXISTS(SELECT 1 FROM organisation_product_entitlements e WHERE e.organisation_id=a.organisation_id AND e.product_key=a.product_key AND e.status='active')),
		assigned.public_id,a.assigned_at,revoked.public_id,a.revoked_at,a.revocation_reason
		FROM user_product_assignments a
		JOIN users u ON u.id=a.user_id
		JOIN organisations o ON o.id=a.organisation_id
		JOIN products p ON p.product_key=a.product_key
		JOIN users assigned ON assigned.id=a.assigned_by
		LEFT JOIN users revoked ON revoked.id=a.revoked_by
		WHERE a.organisation_id=? ORDER BY a.assigned_at,a.id`, organisationInternalID)
	if err != nil {
		return nil, fmt.Errorf("list user product assignments: %w", err)
	}
	defer rows.Close()
	assignments := make([]domain.UserProductAssignment, 0)
	for rows.Next() {
		var assignment domain.UserProductAssignment
		var status, assignedAt string
		var effectiveActive int
		var revokedBy, revokedAt, revocationReason sql.NullString
		if err := rows.Scan(&assignment.ID, &assignment.UserID, &assignment.OrganisationID, &assignment.ProductKey, &status, &effectiveActive, &assignment.AssignedBy, &assignedAt, &revokedBy, &revokedAt, &revocationReason); err != nil {
			return nil, fmt.Errorf("read user product assignment: %w", err)
		}
		assignment.Status = domain.UserProductAssignmentStatus(status)
		assignment.Active = assignment.Status == domain.UserProductAssignmentActive
		assignment.EffectiveActive = effectiveActive == 1
		assignment.AssignedAt = parseTime(assignedAt)
		if revokedBy.Valid {
			assignment.RevokedBy = revokedBy.String
		}
		if revokedAt.Valid {
			value := parseTime(revokedAt.String)
			assignment.RevokedAt = &value
		}
		assignment.RevocationReason = revocationReason.String
		assignments = append(assignments, assignment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user product assignments: %w", err)
	}
	return assignments, nil
}

func (s *Store) ProveProductAccess(ctx context.Context, userID, organisationID, productKey, workspaceID string) (store.ProductAccess, error) {
	if !validProductKey(strings.TrimSpace(productKey)) {
		return store.ProductAccess{}, store.ErrProductAccessDenied
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.ProductAccess{}, fmt.Errorf("begin product access proof: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var access store.ProductAccess
	var organisationRole, workspaceRole string
	err = tx.QueryRowContext(ctx, `SELECT u.public_id,o.public_id,p.product_key,w.public_id,om.role,COALESCE(wm.role,'')
		FROM users u
		JOIN organisations o ON o.public_id=? AND o.status='active'
		JOIN organisation_memberships om ON om.organisation_id=o.id AND om.user_id=u.id AND om.active=1
		JOIN products p ON p.product_key=? AND p.status='active'
		JOIN organisation_product_entitlements e ON e.organisation_id=o.id AND e.product_key=p.product_key AND e.status='active'
		JOIN user_product_assignments a ON a.user_id=u.id AND a.organisation_id=o.id AND a.product_key=p.product_key AND a.status='active'
		JOIN workspaces w ON w.public_id=? AND w.organisation_id=o.id AND w.status='active'
		LEFT JOIN workspace_memberships wm ON wm.workspace_id=w.id AND wm.user_id=u.id AND wm.active=1
		WHERE u.public_id=? AND u.active=1
		AND (om.role IN ('owner','admin') OR wm.id IS NOT NULL)`, organisationID, strings.TrimSpace(productKey), workspaceID, userID).
		Scan(&access.UserID, &access.OrganisationID, &access.ProductKey, &access.WorkspaceID, &organisationRole, &workspaceRole)
	if errors.Is(err, sql.ErrNoRows) {
		return store.ProductAccess{}, store.ErrProductAccessDenied
	}
	if err != nil {
		return store.ProductAccess{}, fmt.Errorf("prove product access: %w", err)
	}
	access.OrganisationRole = domain.OrganisationRole(organisationRole)
	access.WorkspaceRole = domain.WorkspaceRole(workspaceRole)
	return access, nil
}

func validProductKey(value string) bool {
	return strings.HasPrefix(value, "product.") && len(value) > len("product.")
}
