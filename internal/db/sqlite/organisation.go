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
	ErrOrganisationNotFound           = store.ErrOrganisationNotFound
	ErrOrganisationArchived           = store.ErrOrganisationArchived
	ErrMembershipNotFound             = store.ErrMembershipNotFound
	ErrUnauthorisedOrganisationAction = store.ErrUnauthorisedOrganisationAction
	ErrOwnerMutationNotAuthorised     = store.ErrOwnerMutationNotAuthorised
	ErrDuplicateOrganisationMember    = store.ErrDuplicateOrganisationMember
)

const (
	auditOrganisation = "Organisation"

	eventOrganisationCreated   = "organisation_created"
	eventMembershipAdded       = "membership_added"
	eventMembershipRoleChanged = "membership_role_changed"
	eventMembershipDeactivated = "membership_deactivated"
	eventOrganisationArchived  = "organisation_archived"
	eventOrganisationRenamed   = "organisation_renamed"
)

type organisationMutationDetails struct {
	OrganisationID string `json:"organisation_id"`
	MembershipID   string `json:"membership_id,omitempty"`
	UserID         string `json:"user_id,omitempty"`
	Role           string `json:"role,omitempty"`
	Name           string `json:"name,omitempty"`
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

func (s *Store) ListUserOrganisations(ctx context.Context, requesterUserID string) ([]store.UserOrganisation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT o.public_id,o.name,o.status,o.created_at,o.archived_at,COALESCE(m.role,'')
		FROM organisations o
		LEFT JOIN organisation_memberships m ON m.organisation_id=o.id AND m.active=1
			AND m.user_id=(SELECT id FROM users WHERE public_id=? AND active=1)
		WHERE o.status='active' AND (
			m.id IS NOT NULL OR EXISTS (
				SELECT 1 FROM users u
				JOIN system_role_assignments sra ON sra.user_id=u.id AND sra.role='system_admin' AND sra.active=1
				WHERE u.public_id=? AND u.active=1
			)
		)
		ORDER BY o.name,o.public_id`, strings.TrimSpace(requesterUserID), strings.TrimSpace(requesterUserID))
	if err != nil {
		return nil, fmt.Errorf("list user organisations: %w", err)
	}
	defer rows.Close()
	organisations := make([]store.UserOrganisation, 0)
	for rows.Next() {
		var item store.UserOrganisation
		var status, created, role string
		var archived sql.NullString
		if err := rows.Scan(&item.Organisation.ID, &item.Organisation.Name, &status, &created, &archived, &role); err != nil {
			return nil, fmt.Errorf("read user organisation: %w", err)
		}
		item.Organisation.Status = domain.OrganisationStatus(status)
		item.Organisation.CreatedAt = parseTime(created)
		if archived.Valid {
			value := parseTime(archived.String)
			item.Organisation.ArchivedAt = &value
		}
		item.Role = domain.OrganisationRole(role)
		organisations = append(organisations, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user organisations: %w", err)
	}
	return organisations, nil
}

func (s *Store) GetOrganisation(ctx context.Context, requesterUserID, organisationID string) (*domain.Organisation, error) {
	var organisation domain.Organisation
	var status, created string
	var archived sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT o.public_id,o.name,o.status,o.created_at,o.archived_at
		FROM organisations o
		WHERE o.public_id=? AND o.status='active' AND (
			EXISTS (SELECT 1 FROM organisation_memberships m JOIN users u ON u.id=m.user_id AND u.active=1 WHERE m.organisation_id=o.id AND m.active=1 AND u.public_id=?)
			OR EXISTS (SELECT 1 FROM users u JOIN system_role_assignments sra ON sra.user_id=u.id AND sra.role='system_admin' AND sra.active=1 WHERE u.public_id=? AND u.active=1)
		)`, organisationID, requesterUserID, requesterUserID).
		Scan(&organisation.ID, &organisation.Name, &status, &created, &archived)
	if errors.Is(err, sql.ErrNoRows) {
		var exists int
		if archiveErr := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM organisations WHERE public_id=?`, organisationID).Scan(&exists); archiveErr != nil {
			return nil, fmt.Errorf("check organisation scope: %w", archiveErr)
		} else if exists == 0 {
			return nil, ErrOrganisationNotFound
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
		WHERE o.public_id=? AND u.public_id=? AND m.active=1 AND (EXISTS (
			SELECT 1 FROM organisation_memberships requester
			JOIN users requester_user ON requester_user.id=requester.user_id AND requester_user.active=1
			WHERE requester.organisation_id=o.id AND requester.active=1 AND requester_user.public_id=?
		) OR EXISTS (
			SELECT 1 FROM users requester_user
			JOIN system_role_assignments requester_role ON requester_role.user_id=requester_user.id AND requester_role.role='system_admin' AND requester_role.active=1
			WHERE requester_user.public_id=? AND requester_user.active=1
		))`, organisationID, targetUserID, requesterUserID, requesterUserID).
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
	actorInternalID, actorRole, organisationInternalID, systemAdmin, err := authorisedOrganisationMutationTx(ctx, tx, actorUserID, organisationID)
	if err != nil {
		return domain.OrganisationMembership{}, err
	}
	if !systemAdmin && actorRole == domain.OrganisationAdmin && role == domain.OrganisationAdmin {
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
	actorInternalID, actorRole, organisationInternalID, systemAdmin, err := authorisedOrganisationMutationTx(ctx, tx, actorUserID, organisationID)
	if err != nil {
		return domain.OrganisationMembership{}, err
	}
	var currentID, currentVersion int64
	var currentRole string
	if err := tx.QueryRowContext(ctx, `SELECT m.id,m.membership_version,m.role FROM organisation_memberships m JOIN users u ON u.id=m.user_id AND u.active=1 WHERE m.organisation_id=? AND u.public_id=? AND m.active=1`, organisationInternalID, targetUserID).Scan(&currentID, &currentVersion, &currentRole); errors.Is(err, sql.ErrNoRows) {
		return domain.OrganisationMembership{}, ErrMembershipNotFound
	} else if err != nil {
		return domain.OrganisationMembership{}, fmt.Errorf("read membership role: %w", err)
	}
	if currentRole == string(domain.OrganisationOwner) {
		return domain.OrganisationMembership{}, ErrOwnerMutationNotAuthorised
	}
	if !systemAdmin && actorRole == domain.OrganisationAdmin && currentRole != string(domain.OrganisationMember) {
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
	if _, err := tx.ExecContext(ctx, `INSERT INTO organisation_memberships(public_id,organisation_id,user_id,role,active,assigned_by,assigned_at,membership_version) VALUES(?,?,?,?,1,?,?,?)`, membershipID, organisationInternalID, targetInternalID, string(role), actorInternalID, formatTime(at.UTC()), currentVersion+1); err != nil {
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
	actorInternalID, actorRole, organisationInternalID, systemAdmin, err := authorisedOrganisationMutationTx(ctx, tx, actorUserID, organisationID)
	if err != nil {
		return err
	}
	var membershipID int64
	var membershipPublicID string
	var targetRole string
	if err := tx.QueryRowContext(ctx, `SELECT m.id,m.public_id,m.role FROM organisation_memberships m JOIN users u ON u.id=m.user_id AND u.active=1 WHERE m.organisation_id=? AND u.public_id=? AND m.active=1`, organisationInternalID, targetUserID).Scan(&membershipID, &membershipPublicID, &targetRole); errors.Is(err, sql.ErrNoRows) {
		return ErrMembershipNotFound
	} else if err != nil {
		return fmt.Errorf("read membership for deactivation: %w", err)
	}
	if targetRole == string(domain.OrganisationOwner) {
		return ErrOwnerMutationNotAuthorised
	}
	if !systemAdmin && actorRole == domain.OrganisationAdmin && targetRole != string(domain.OrganisationMember) {
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
	if err := recordOrganisationAuditTx(ctx, tx, actorInternalID, organisationID, eventMembershipDeactivated, organisationMutationDetails{OrganisationID: organisationID, MembershipID: membershipPublicID, UserID: targetUserID, Role: targetRole, Reason: strings.TrimSpace(reason)}, at); err != nil {
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
	actorInternalID, actorRole, organisationInternalID, systemAdmin, err := authorisedOrganisationMutationTx(ctx, tx, actorUserID, organisationID)
	if err != nil {
		return err
	}
	if !systemAdmin && actorRole != domain.OrganisationOwner {
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

func (s *Store) RenameOrganisation(ctx context.Context, actorUserID, organisationID, name, reason string, at time.Time) (domain.Organisation, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 200 {
		return domain.Organisation{}, domain.ErrInvalidOrganisationName
	}
	if err := validateMutationReason(reason); err != nil {
		return domain.Organisation{}, err
	}
	if at.IsZero() {
		return domain.Organisation{}, errors.New("organisation rename time is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Organisation{}, fmt.Errorf("begin organisation rename: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	actorInternalID, _, organisationInternalID, _, err := authorisedOrganisationMutationTx(ctx, tx, actorUserID, organisationID)
	if err != nil {
		return domain.Organisation{}, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE organisations SET name=? WHERE id=? AND status='active'`, name, organisationInternalID)
	if err != nil {
		return domain.Organisation{}, fmt.Errorf("rename organisation: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil {
		return domain.Organisation{}, fmt.Errorf("check organisation rename: %w", err)
	} else if rows != 1 {
		return domain.Organisation{}, ErrOrganisationArchived
	}
	if err := recordOrganisationAuditTx(ctx, tx, actorInternalID, organisationID, eventOrganisationRenamed, organisationMutationDetails{OrganisationID: organisationID, Name: name, Reason: strings.TrimSpace(reason)}, at); err != nil {
		return domain.Organisation{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Organisation{}, fmt.Errorf("commit organisation rename: %w", err)
	}
	return domain.Organisation{ID: organisationID, Name: name, Status: domain.OrganisationActive}, nil
}

func (s *Store) ListOrganisationMembers(ctx context.Context, requesterUserID, organisationID string) ([]store.OrganisationMemberRecord, error) {
	organisationInternalID, err := s.authorisedOrganisationRead(ctx, requesterUserID, organisationID, true)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT m.public_id,o.public_id,u.public_id,u.email,u.display_name,m.role,m.assigned_at
		FROM organisation_memberships m
		JOIN organisations o ON o.id=m.organisation_id AND o.status='active'
		JOIN users u ON u.id=m.user_id AND u.active=1
		WHERE m.organisation_id=? AND m.active=1 ORDER BY u.display_name,u.public_id`, organisationInternalID)
	if err != nil {
		return nil, fmt.Errorf("list organisation members: %w", err)
	}
	defer rows.Close()
	var members []store.OrganisationMemberRecord
	for rows.Next() {
		var member store.OrganisationMemberRecord
		var role, assignedAt string
		if err := rows.Scan(&member.MembershipID, &member.OrganisationID, &member.UserID, &member.Email, &member.DisplayName, &role, &assignedAt); err != nil {
			return nil, fmt.Errorf("read organisation member: %w", err)
		}
		member.Role = domain.OrganisationRole(role)
		member.AssignedAt = parseTime(assignedAt)
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate organisation members: %w", err)
	}
	return members, nil
}

func (s *Store) ListOrganisationAudit(ctx context.Context, requesterUserID, organisationID string, limit int) ([]store.OrganisationAuditRecord, error) {
	organisationInternalID, err := s.authorisedOrganisationRead(ctx, requesterUserID, organisationID, true)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		return nil, errors.New("organisation audit limit must be between 1 and 100")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT a.id,o.public_id,actor.public_id,a.event,a.details,a.occurred_at
		FROM audit_events a
		JOIN organisations o ON o.public_id=a.aggregate_id AND o.id=?
		JOIN users actor ON actor.id=a.actor_user_id
		WHERE a.aggregate_type=? AND a.aggregate_id=?
		ORDER BY a.occurred_at DESC,a.id DESC LIMIT ?`, organisationInternalID, auditOrganisation, organisationID, limit)
	if err != nil {
		return nil, fmt.Errorf("list organisation audit: %w", err)
	}
	defer rows.Close()
	var events []store.OrganisationAuditRecord
	for rows.Next() {
		var event store.OrganisationAuditRecord
		var occurredAt string
		if err := rows.Scan(&event.ID, &event.OrganisationID, &event.ActorUserID, &event.Event, &event.Details, &occurredAt); err != nil {
			return nil, fmt.Errorf("read organisation audit: %w", err)
		}
		event.OccurredAt = parseTime(occurredAt)
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate organisation audit: %w", err)
	}
	return events, nil
}

func (s *Store) GetOrganisationWorkspaceEntry(ctx context.Context, requesterUserID, organisationID string) (store.OrganisationWorkspaceEntry, error) {
	if _, err := s.GetOrganisation(ctx, requesterUserID, organisationID); err != nil {
		return store.OrganisationWorkspaceEntry{}, err
	}
	workspaces, err := s.ListOrganisationWorkspaces(ctx, requesterUserID, organisationID)
	if err != nil {
		return store.OrganisationWorkspaceEntry{}, err
	}
	entry := store.OrganisationWorkspaceEntry{OrganisationID: organisationID, Resource: "workspaces", Available: len(workspaces) > 0, Workspaces: workspaces}
	if len(workspaces) > 0 {
		entry.DefaultWorkspaceID = workspaces[0].ID
	}
	return entry, nil
}

func (s *Store) authorisedOrganisationRead(ctx context.Context, requesterUserID, organisationID string, adminOnly bool) (int64, error) {
	var organisationInternalID int64
	var status string
	if err := s.db.QueryRowContext(ctx, `SELECT id,status FROM organisations WHERE public_id=?`, organisationID).Scan(&organisationInternalID, &status); errors.Is(err, sql.ErrNoRows) {
		return 0, ErrOrganisationNotFound
	} else if err != nil {
		return 0, fmt.Errorf("resolve organisation read: %w", err)
	}
	if status != string(domain.OrganisationActive) {
		return 0, ErrOrganisationArchived
	}
	var userInternalID int64
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE public_id=? AND active=1`, requesterUserID).Scan(&userInternalID); errors.Is(err, sql.ErrNoRows) {
		return 0, ErrUnauthorisedOrganisationAction
	} else if err != nil {
		return 0, fmt.Errorf("resolve organisation reader: %w", err)
	}
	var systemAdmin int
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM system_role_assignments WHERE user_id=? AND role='system_admin' AND active=1)`, userInternalID).Scan(&systemAdmin); err != nil {
		return 0, fmt.Errorf("resolve organisation reader system role: %w", err)
	}
	if systemAdmin == 1 {
		return organisationInternalID, nil
	}
	var role string
	if err := s.db.QueryRowContext(ctx, `SELECT role FROM organisation_memberships WHERE organisation_id=? AND user_id=? AND active=1`, organisationInternalID, userInternalID).Scan(&role); errors.Is(err, sql.ErrNoRows) {
		return 0, ErrUnauthorisedOrganisationAction
	} else if err != nil {
		return 0, fmt.Errorf("resolve organisation reader membership: %w", err)
	}
	if adminOnly && !domain.OrganisationRole(role).CanAdminister() {
		return 0, ErrUnauthorisedOrganisationAction
	}
	return organisationInternalID, nil
}

func authorisedOrganisationMutationTx(ctx context.Context, tx *sql.Tx, actorUserID, organisationID string) (int64, domain.OrganisationRole, int64, bool, error) {
	var actorInternalID, organisationInternalID int64
	var role, status string
	if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE public_id=? AND active=1`, actorUserID).Scan(&actorInternalID); errors.Is(err, sql.ErrNoRows) {
		return 0, "", 0, false, ErrUnauthorisedOrganisationAction
	} else if err != nil {
		return 0, "", 0, false, fmt.Errorf("resolve mutation actor: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT id,status FROM organisations WHERE public_id=?`, organisationID).Scan(&organisationInternalID, &status); errors.Is(err, sql.ErrNoRows) {
		return 0, "", 0, false, ErrOrganisationNotFound
	} else if err != nil {
		return 0, "", 0, false, fmt.Errorf("resolve mutation organisation: %w", err)
	}
	if status != string(domain.OrganisationActive) {
		return 0, "", 0, false, ErrOrganisationArchived
	}
	var systemAdmin int
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM system_role_assignments WHERE user_id=? AND role='system_admin' AND active=1)`, actorInternalID).Scan(&systemAdmin); err != nil {
		return 0, "", 0, false, fmt.Errorf("resolve system authority: %w", err)
	}
	if systemAdmin == 1 {
		return actorInternalID, "", organisationInternalID, true, nil
	}
	if err := tx.QueryRowContext(ctx, `SELECT role FROM organisation_memberships WHERE organisation_id=? AND user_id=? AND active=1`, organisationInternalID, actorInternalID).Scan(&role); errors.Is(err, sql.ErrNoRows) {
		return 0, "", 0, false, ErrUnauthorisedOrganisationAction
	} else if err != nil {
		return 0, "", 0, false, fmt.Errorf("resolve organisation authority: %w", err)
	}
	parsedRole := domain.OrganisationRole(role)
	if !parsedRole.CanAdminister() {
		return 0, "", 0, false, ErrUnauthorisedOrganisationAction
	}
	return actorInternalID, parsedRole, organisationInternalID, false, nil
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
	if err := appendOrganisationEventTx(ctx, tx, event, details, at); err != nil {
		return fmt.Errorf("record organisation outbox event: %w", err)
	}
	return nil
}

func (s *Store) organisationAuditCount(ctx context.Context, organisationID, event string) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE aggregate_type=? AND aggregate_id=? AND event=?`, auditOrganisation, organisationID, event).Scan(&count); err != nil {
		return 0, fmt.Errorf("count organisation audit: %w", err)
	}
	return count, nil
}
