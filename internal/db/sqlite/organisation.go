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
)

var (
	ErrOrganisationNotFound           = errors.New("organisation not found")
	ErrOrganisationArchived           = errors.New("organisation is archived")
	ErrMembershipNotFound             = errors.New("organisation membership not found")
	ErrUnauthorisedOrganisationAction = errors.New("organisation action is not authorised")
	ErrOwnerMutationNotAuthorised     = errors.New("organisation owner mutation is not authorised")
	ErrDuplicateOrganisationMember    = errors.New("active organisation membership already exists")
)

const (
	auditOrganisation = "Organisation"

	eventOrganisationCreated   = "organisation_created"
	eventMembershipAdded       = "membership_added"
	eventMembershipRoleChanged = "membership_role_changed"
	eventMembershipDeactivated = "membership_deactivated"
	eventOrganisationArchived  = "organisation_archived"
)

type organisationMutationDetails struct {
	OrganisationID string `json:"organisation_id"`
	MembershipID   string `json:"membership_id,omitempty"`
	UserID         string `json:"user_id,omitempty"`
	Role           string `json:"role,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

func (s *Store) CreateOrganisation(ctx context.Context, actorUserID string, organisation domain.Organisation, reason string, at time.Time) (domain.Organisation, domain.OrganisationMembership, error) {
	if err := validateMutationReason(reason); err != nil {
		return domain.Organisation{}, domain.OrganisationMembership{}, err
	}
	if organisation.ID == "" {
		var err error
		organisation.ID, err = domain.NewOrganisationID()
		if err != nil {
			return domain.Organisation{}, domain.OrganisationMembership{}, err
		}
	}
	organisation.Name = strings.TrimSpace(organisation.Name)
	organisation.Status = domain.OrganisationActive
	organisation.CreatedAt = at.UTC()
	organisation.ArchivedAt = nil
	if err := organisation.Validate(); err != nil {
		return domain.Organisation{}, domain.OrganisationMembership{}, err
	}
	if organisation.CreatedAt.IsZero() {
		return domain.Organisation{}, domain.OrganisationMembership{}, errors.New("organisation creation time is required")
	}
	membershipID, err := domain.NewOrganisationMembershipID()
	if err != nil {
		return domain.Organisation{}, domain.OrganisationMembership{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Organisation{}, domain.OrganisationMembership{}, fmt.Errorf("begin organisation creation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	actorInternalID, err := activeUserIDTx(ctx, tx, actorUserID)
	if err != nil {
		return domain.Organisation{}, domain.OrganisationMembership{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO organisations(public_id,name,status,created_at) VALUES(?,?,?,?)`, organisation.ID, organisation.Name, string(organisation.Status), formatTime(organisation.CreatedAt)); err != nil {
		return domain.Organisation{}, domain.OrganisationMembership{}, fmt.Errorf("create organisation: %w", err)
	}
	var organisationInternalID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM organisations WHERE public_id=?`, organisation.ID).Scan(&organisationInternalID); err != nil {
		return domain.Organisation{}, domain.OrganisationMembership{}, fmt.Errorf("resolve created organisation: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO organisation_memberships(public_id,organisation_id,user_id,role,active,assigned_by,assigned_at) VALUES(?,?,?,?,1,?,?)`, membershipID, organisationInternalID, actorInternalID, string(domain.OrganisationOwner), actorInternalID, formatTime(organisation.CreatedAt)); err != nil {
		return domain.Organisation{}, domain.OrganisationMembership{}, fmt.Errorf("assign organisation owner: %w", err)
	}
	if err := recordOrganisationAuditTx(ctx, tx, actorInternalID, organisation.ID, eventOrganisationCreated, organisationMutationDetails{OrganisationID: organisation.ID, Reason: strings.TrimSpace(reason)}, organisation.CreatedAt); err != nil {
		return domain.Organisation{}, domain.OrganisationMembership{}, err
	}
	if err := recordOrganisationAuditTx(ctx, tx, actorInternalID, organisation.ID, eventMembershipAdded, organisationMutationDetails{OrganisationID: organisation.ID, MembershipID: membershipID, UserID: actorUserID, Role: string(domain.OrganisationOwner), Reason: strings.TrimSpace(reason)}, organisation.CreatedAt); err != nil {
		return domain.Organisation{}, domain.OrganisationMembership{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Organisation{}, domain.OrganisationMembership{}, fmt.Errorf("commit organisation creation: %w", err)
	}
	return organisation, domain.OrganisationMembership{ID: membershipID, OrganisationID: organisation.ID, UserID: actorUserID, Role: domain.OrganisationOwner, Active: true, AssignedBy: actorUserID, AssignedAt: organisation.CreatedAt}, nil
}

func (s *Store) GetOrganisation(ctx context.Context, requesterUserID, organisationID string) (*domain.Organisation, error) {
	var organisation domain.Organisation
	var status, created string
	var archived sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT o.public_id,o.name,o.status,o.created_at,o.archived_at
		FROM organisations o JOIN organisation_memberships m ON m.organisation_id=o.id AND m.active=1
		JOIN users u ON u.id=m.user_id AND u.active=1
		WHERE o.public_id=? AND o.status='active' AND u.public_id=?`, organisationID, requesterUserID).
		Scan(&organisation.ID, &organisation.Name, &status, &created, &archived)
	if errors.Is(err, sql.ErrNoRows) {
		var archivedStatus string
		if archiveErr := s.db.QueryRowContext(ctx, `SELECT status FROM organisations WHERE public_id=?`, organisationID).Scan(&archivedStatus); errors.Is(archiveErr, sql.ErrNoRows) {
			return nil, ErrOrganisationNotFound
		} else if archiveErr != nil {
			return nil, fmt.Errorf("check organisation scope: %w", archiveErr)
		}
		return nil, ErrUnauthorisedOrganisationAction
	}
	if err != nil {
		return nil, fmt.Errorf("read organisation: %w", err)
	}
	organisation.Status = domain.OrganisationStatus(status)
	organisation.CreatedAt = parseTime(created)
	if archived.Valid {
		value := parseTime(archived.String)
		organisation.ArchivedAt = &value
	}
	return &organisation, nil
}

func (s *Store) GetOrganisationMembership(ctx context.Context, requesterUserID, organisationID, targetUserID string) (*domain.OrganisationMembership, error) {
	var membership domain.OrganisationMembership
	var role, assignedAt string
	err := s.db.QueryRowContext(ctx, `SELECT m.public_id,o.public_id,u.public_id,m.role,m.active,assigned.public_id,m.assigned_at
		FROM organisation_memberships m
		JOIN organisations o ON o.id=m.organisation_id AND o.status='active'
		JOIN users u ON u.id=m.user_id AND u.active=1
		JOIN users assigned ON assigned.id=m.assigned_by
		WHERE o.public_id=? AND u.public_id=? AND m.active=1 AND EXISTS (
			SELECT 1 FROM organisation_memberships requester
			JOIN users requester_user ON requester_user.id=requester.user_id AND requester_user.active=1
			WHERE requester.organisation_id=o.id AND requester.active=1 AND requester_user.public_id=?
		)`, organisationID, targetUserID, requesterUserID).
		Scan(&membership.ID, &membership.OrganisationID, &membership.UserID, &role, &membership.Active, &membership.AssignedBy, &assignedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrMembershipNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read organisation membership: %w", err)
	}
	membership.Role = domain.OrganisationRole(role)
	membership.AssignedAt = parseTime(assignedAt)
	return &membership, nil
}

func (s *Store) AddOrganisationMember(ctx context.Context, actorUserID, organisationID, targetUserID string, role domain.OrganisationRole, reason string, at time.Time) (domain.OrganisationMembership, error) {
	if role != domain.OrganisationAdmin && role != domain.OrganisationMember {
		return domain.OrganisationMembership{}, domain.ErrInvalidOrganisationRole
	}
	if err := validateMutationReason(reason); err != nil {
		return domain.OrganisationMembership{}, err
	}
	if at.IsZero() {
		return domain.OrganisationMembership{}, errors.New("membership assignment time is required")
	}
	membershipID, err := domain.NewOrganisationMembershipID()
	if err != nil {
		return domain.OrganisationMembership{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.OrganisationMembership{}, fmt.Errorf("begin membership addition: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	actorInternalID, actorRole, organisationInternalID, err := authorisedOrganisationMutationTx(ctx, tx, actorUserID, organisationID)
	if err != nil {
		return domain.OrganisationMembership{}, err
	}
	if actorRole == domain.OrganisationAdmin && role == domain.OrganisationAdmin {
		return domain.OrganisationMembership{}, ErrUnauthorisedOrganisationAction
	}
	targetInternalID, err := activeUserIDTx(ctx, tx, targetUserID)
	if err != nil {
		return domain.OrganisationMembership{}, err
	}
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM organisation_memberships WHERE organisation_id=? AND user_id=? AND active=1`, organisationInternalID, targetInternalID).Scan(&exists); err != nil {
		return domain.OrganisationMembership{}, fmt.Errorf("check active membership: %w", err)
	}
	if exists != 0 {
		return domain.OrganisationMembership{}, ErrDuplicateOrganisationMember
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO organisation_memberships(public_id,organisation_id,user_id,role,active,assigned_by,assigned_at) VALUES(?,?,?,?,1,?,?)`, membershipID, organisationInternalID, targetInternalID, string(role), actorInternalID, formatTime(at.UTC())); err != nil {
		return domain.OrganisationMembership{}, fmt.Errorf("add organisation member: %w", err)
	}
	if err := recordOrganisationAuditTx(ctx, tx, actorInternalID, organisationID, eventMembershipAdded, organisationMutationDetails{OrganisationID: organisationID, MembershipID: membershipID, UserID: targetUserID, Role: string(role), Reason: strings.TrimSpace(reason)}, at); err != nil {
		return domain.OrganisationMembership{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.OrganisationMembership{}, fmt.Errorf("commit membership addition: %w", err)
	}
	return domain.OrganisationMembership{ID: membershipID, OrganisationID: organisationID, UserID: targetUserID, Role: role, Active: true, AssignedBy: actorUserID, AssignedAt: at.UTC()}, nil
}

func (s *Store) ChangeOrganisationMemberRole(ctx context.Context, actorUserID, organisationID, targetUserID string, role domain.OrganisationRole, reason string, at time.Time) (domain.OrganisationMembership, error) {
	if role != domain.OrganisationAdmin && role != domain.OrganisationMember {
		return domain.OrganisationMembership{}, domain.ErrInvalidOrganisationRole
	}
	if err := validateMutationReason(reason); err != nil {
		return domain.OrganisationMembership{}, err
	}
	if at.IsZero() {
		return domain.OrganisationMembership{}, errors.New("membership role change time is required")
	}
	membershipID, err := domain.NewOrganisationMembershipID()
	if err != nil {
		return domain.OrganisationMembership{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.OrganisationMembership{}, fmt.Errorf("begin membership role change: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	actorInternalID, actorRole, organisationInternalID, err := authorisedOrganisationMutationTx(ctx, tx, actorUserID, organisationID)
	if err != nil {
		return domain.OrganisationMembership{}, err
	}
	var currentID int64
	var currentRole string
	if err := tx.QueryRowContext(ctx, `SELECT m.id,m.role FROM organisation_memberships m JOIN users u ON u.id=m.user_id AND u.active=1 WHERE m.organisation_id=? AND u.public_id=? AND m.active=1`, organisationInternalID, targetUserID).Scan(&currentID, &currentRole); errors.Is(err, sql.ErrNoRows) {
		return domain.OrganisationMembership{}, ErrMembershipNotFound
	} else if err != nil {
		return domain.OrganisationMembership{}, fmt.Errorf("read membership role: %w", err)
	}
	if currentRole == string(domain.OrganisationOwner) {
		return domain.OrganisationMembership{}, ErrOwnerMutationNotAuthorised
	}
	if actorRole == domain.OrganisationAdmin && currentRole != string(domain.OrganisationMember) {
		return domain.OrganisationMembership{}, ErrUnauthorisedOrganisationAction
	}
	if currentRole == string(role) {
		return domain.OrganisationMembership{}, ErrDuplicateOrganisationMember
	}
	targetInternalID, err := activeUserIDTx(ctx, tx, targetUserID)
	if err != nil {
		return domain.OrganisationMembership{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE organisation_memberships SET active=0,removed_by=?,removed_at=?,removal_reason=? WHERE id=? AND active=1`, actorInternalID, formatTime(at.UTC()), strings.TrimSpace(reason), currentID); err != nil {
		return domain.OrganisationMembership{}, fmt.Errorf("retire membership role: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO organisation_memberships(public_id,organisation_id,user_id,role,active,assigned_by,assigned_at) VALUES(?,?,?,?,1,?,?)`, membershipID, organisationInternalID, targetInternalID, string(role), actorInternalID, formatTime(at.UTC())); err != nil {
		return domain.OrganisationMembership{}, fmt.Errorf("write membership role: %w", err)
	}
	if err := recordOrganisationAuditTx(ctx, tx, actorInternalID, organisationID, eventMembershipRoleChanged, organisationMutationDetails{OrganisationID: organisationID, MembershipID: membershipID, UserID: targetUserID, Role: string(role), Reason: strings.TrimSpace(reason)}, at); err != nil {
		return domain.OrganisationMembership{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.OrganisationMembership{}, fmt.Errorf("commit membership role change: %w", err)
	}
	return domain.OrganisationMembership{ID: membershipID, OrganisationID: organisationID, UserID: targetUserID, Role: role, Active: true, AssignedBy: actorUserID, AssignedAt: at.UTC()}, nil
}

func (s *Store) DeactivateOrganisationMember(ctx context.Context, actorUserID, organisationID, targetUserID, reason string, at time.Time) error {
	if err := validateMutationReason(reason); err != nil {
		return err
	}
	if at.IsZero() {
		return errors.New("membership deactivation time is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin membership deactivation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	actorInternalID, actorRole, organisationInternalID, err := authorisedOrganisationMutationTx(ctx, tx, actorUserID, organisationID)
	if err != nil {
		return err
	}
	var membershipID int64
	var targetRole string
	if err := tx.QueryRowContext(ctx, `SELECT m.id,m.role FROM organisation_memberships m JOIN users u ON u.id=m.user_id AND u.active=1 WHERE m.organisation_id=? AND u.public_id=? AND m.active=1`, organisationInternalID, targetUserID).Scan(&membershipID, &targetRole); errors.Is(err, sql.ErrNoRows) {
		return ErrMembershipNotFound
	} else if err != nil {
		return fmt.Errorf("read membership for deactivation: %w", err)
	}
	if targetRole == string(domain.OrganisationOwner) {
		return ErrOwnerMutationNotAuthorised
	}
	if actorRole == domain.OrganisationAdmin && targetRole != string(domain.OrganisationMember) {
		return ErrUnauthorisedOrganisationAction
	}
	result, err := tx.ExecContext(ctx, `UPDATE organisation_memberships SET active=0,removed_by=?,removed_at=?,removal_reason=? WHERE id=? AND active=1`, actorInternalID, formatTime(at.UTC()), strings.TrimSpace(reason), membershipID)
	if err != nil {
		return fmt.Errorf("deactivate organisation member: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("check membership deactivation: %w", err)
	} else if rows != 1 {
		return ErrMembershipNotFound
	}
	if err := recordOrganisationAuditTx(ctx, tx, actorInternalID, organisationID, eventMembershipDeactivated, organisationMutationDetails{OrganisationID: organisationID, UserID: targetUserID, Reason: strings.TrimSpace(reason)}, at); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit membership deactivation: %w", err)
	}
	return nil
}

func (s *Store) ArchiveOrganisation(ctx context.Context, actorUserID, organisationID, reason string, at time.Time) error {
	if err := validateMutationReason(reason); err != nil {
		return err
	}
	if at.IsZero() {
		return errors.New("organisation archive time is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin organisation archive: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	actorInternalID, actorRole, organisationInternalID, err := authorisedOrganisationMutationTx(ctx, tx, actorUserID, organisationID)
	if err != nil {
		return err
	}
	if actorRole != domain.OrganisationOwner {
		return ErrUnauthorisedOrganisationAction
	}
	result, err := tx.ExecContext(ctx, `UPDATE organisations SET status='archived',archived_at=? WHERE id=? AND status='active'`, formatTime(at.UTC()), organisationInternalID)
	if err != nil {
		return fmt.Errorf("archive organisation: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("check organisation archive: %w", err)
	} else if rows != 1 {
		return ErrOrganisationArchived
	}
	rows, err := tx.QueryContext(ctx, `SELECT m.public_id,u.public_id,m.role FROM organisation_memberships m JOIN users u ON u.id=m.user_id WHERE m.organisation_id=? AND m.active=1 ORDER BY m.id`, organisationInternalID)
	if err != nil {
		return fmt.Errorf("read memberships for archive: %w", err)
	}
	type archivedMembership struct {
		membershipID string
		userID       string
		role         string
	}
	var activeMemberships []archivedMembership
	for rows.Next() {
		var membership archivedMembership
		if err := rows.Scan(&membership.membershipID, &membership.userID, &membership.role); err != nil {
			_ = rows.Close()
			return fmt.Errorf("read membership for archive: %w", err)
		}
		activeMemberships = append(activeMemberships, membership)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterate memberships for archive: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close memberships for archive: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE organisation_memberships SET active=0,removed_by=?,removed_at=?,removal_reason=? WHERE organisation_id=? AND active=1`, actorInternalID, formatTime(at.UTC()), strings.TrimSpace(reason), organisationInternalID); err != nil {
		return fmt.Errorf("deactivate archived memberships: %w", err)
	}
	for _, membership := range activeMemberships {
		if err := recordOrganisationAuditTx(ctx, tx, actorInternalID, organisationID, eventMembershipDeactivated, organisationMutationDetails{OrganisationID: organisationID, MembershipID: membership.membershipID, UserID: membership.userID, Role: membership.role, Reason: strings.TrimSpace(reason)}, at); err != nil {
			return err
		}
	}
	if err := recordOrganisationAuditTx(ctx, tx, actorInternalID, organisationID, eventOrganisationArchived, organisationMutationDetails{OrganisationID: organisationID, Reason: strings.TrimSpace(reason)}, at); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit organisation archive: %w", err)
	}
	return nil
}

func authorisedOrganisationMutationTx(ctx context.Context, tx *sql.Tx, actorUserID, organisationID string) (int64, domain.OrganisationRole, int64, error) {
	var actorInternalID, organisationInternalID int64
	var role, status string
	query := `SELECT u.id,m.role,o.id,o.status FROM users u
		JOIN organisation_memberships m ON m.user_id=u.id AND m.active=1
		JOIN organisations o ON o.id=m.organisation_id
		WHERE u.public_id=? AND u.active=1 AND o.public_id=?`
	if err := tx.QueryRowContext(ctx, query, actorUserID, organisationID).Scan(&actorInternalID, &role, &organisationInternalID, &status); errors.Is(err, sql.ErrNoRows) {
		var exists int
		if checkErr := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM organisations WHERE public_id=?`, organisationID).Scan(&exists); checkErr != nil {
			return 0, "", 0, fmt.Errorf("check organisation existence: %w", checkErr)
		}
		if exists == 0 {
			return 0, "", 0, ErrOrganisationNotFound
		}
		return 0, "", 0, ErrUnauthorisedOrganisationAction
	} else if err != nil {
		return 0, "", 0, fmt.Errorf("resolve organisation authority: %w", err)
	}
	if status != string(domain.OrganisationActive) {
		return 0, "", 0, ErrOrganisationArchived
	}
	parsedRole := domain.OrganisationRole(role)
	if !parsedRole.CanAdminister() {
		return 0, "", 0, ErrUnauthorisedOrganisationAction
	}
	return actorInternalID, parsedRole, organisationInternalID, nil
}

func activeUserIDTx(ctx context.Context, tx *sql.Tx, publicID string) (int64, error) {
	var internalID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE public_id=? AND active=1`, strings.TrimSpace(publicID)).Scan(&internalID); errors.Is(err, sql.ErrNoRows) {
		return 0, ErrUnauthorisedOrganisationAction
	} else if err != nil {
		return 0, fmt.Errorf("resolve active user: %w", err)
	}
	return internalID, nil
}

func validateMutationReason(reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 500 {
		return domain.ErrInvalidReason
	}
	return nil
}

func recordOrganisationAuditTx(ctx context.Context, tx *sql.Tx, actorInternalID int64, organisationID, event string, details organisationMutationDetails, at time.Time) error {
	payload, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("encode organisation audit: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_events(actor_user_id,aggregate_type,aggregate_id,event,details,occurred_at) VALUES(?,?,?,?,?,?)`, actorInternalID, auditOrganisation, organisationID, event, string(payload), formatTime(at.UTC())); err != nil {
		return fmt.Errorf("record organisation audit: %w", err)
	}
	return nil
}

func (s *Store) OrganisationAuditCount(ctx context.Context, organisationID, event string) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE aggregate_type=? AND aggregate_id=? AND event=?`, auditOrganisation, organisationID, event).Scan(&count); err != nil {
		return 0, fmt.Errorf("count organisation audit: %w", err)
	}
	return count, nil
}
