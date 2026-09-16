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
	ErrWorkspaceNotFound           = store.ErrWorkspaceNotFound
	ErrWorkspaceArchived           = store.ErrWorkspaceArchived
	ErrWorkspaceMembershipNotFound = store.ErrWorkspaceMembershipNotFound
	ErrUnauthorisedWorkspaceAction = store.ErrUnauthorisedWorkspaceAction
	ErrDuplicateWorkspaceMember    = store.ErrDuplicateWorkspaceMember
)

const (
	auditWorkspace = "Workspace"

	eventWorkspaceCreated         = "workspace_created"
	eventWorkspaceMembershipAdded = "workspace_membership_added"
	eventWorkspaceRoleChanged     = "workspace_membership_role_changed"
	eventWorkspaceMemberRemoved   = "workspace_membership_deactivated"
	eventWorkspaceArchived        = "workspace_archived"
)

type workspaceMutationDetails struct {
	WorkspaceID  string `json:"workspace_id"`
	MembershipID string `json:"membership_id,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	Role         string `json:"role,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

func (s *Store) CreateWorkspace(ctx context.Context, actorUserID, organisationID string, workspace domain.Workspace, reason string, at time.Time) (domain.Workspace, domain.WorkspaceMembership, error) {
	if err := validateMutationReason(reason); err != nil {
		return domain.Workspace{}, domain.WorkspaceMembership{}, err
	}
	if at.IsZero() {
		return domain.Workspace{}, domain.WorkspaceMembership{}, errors.New("workspace creation time is required")
	}
	if workspace.ID == "" {
		var err error
		workspace.ID, err = domain.NewWorkspaceID()
		if err != nil {
			return domain.Workspace{}, domain.WorkspaceMembership{}, err
		}
	}
	workspace.OrganisationID = organisationID
	workspace.Name = strings.TrimSpace(workspace.Name)
	workspace.Status = domain.WorkspaceActive
	workspace.CreatedAt = at.UTC()
	workspace.ArchivedAt = nil
	if err := workspace.Validate(); err != nil {
		return domain.Workspace{}, domain.WorkspaceMembership{}, err
	}
	membershipID, err := domain.NewWorkspaceMembershipID()
	if err != nil {
		return domain.Workspace{}, domain.WorkspaceMembership{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Workspace{}, domain.WorkspaceMembership{}, fmt.Errorf("begin workspace creation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	actorInternalID, _, organisationInternalID, _, err := authorisedOrganisationMutationTx(ctx, tx, actorUserID, organisationID)
	if err != nil {
		return domain.Workspace{}, domain.WorkspaceMembership{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO workspaces(public_id,organisation_id,name,status,created_at) VALUES(?,?,?,?,?)`, workspace.ID, organisationInternalID, workspace.Name, string(workspace.Status), formatTime(workspace.CreatedAt)); err != nil {
		return domain.Workspace{}, domain.WorkspaceMembership{}, fmt.Errorf("create workspace: %w", err)
	}
	var workspaceInternalID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM workspaces WHERE public_id=?`, workspace.ID).Scan(&workspaceInternalID); err != nil {
		return domain.Workspace{}, domain.WorkspaceMembership{}, fmt.Errorf("resolve created workspace: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO workspace_memberships(public_id,workspace_id,user_id,role,active,assigned_by,assigned_at) VALUES(?,?,?,?,1,?,?)`, membershipID, workspaceInternalID, actorInternalID, string(domain.WorkspaceAdmin), actorInternalID, formatTime(workspace.CreatedAt)); err != nil {
		return domain.Workspace{}, domain.WorkspaceMembership{}, fmt.Errorf("assign workspace administrator: %w", err)
	}
	if err := recordWorkspaceAuditTx(ctx, tx, actorInternalID, workspace.ID, eventWorkspaceCreated, workspaceMutationDetails{WorkspaceID: workspace.ID, Reason: strings.TrimSpace(reason)}, workspace.CreatedAt); err != nil {
		return domain.Workspace{}, domain.WorkspaceMembership{}, err
	}
	if err := recordWorkspaceAuditTx(ctx, tx, actorInternalID, workspace.ID, eventWorkspaceMembershipAdded, workspaceMutationDetails{WorkspaceID: workspace.ID, MembershipID: membershipID, UserID: actorUserID, Role: string(domain.WorkspaceAdmin), Reason: strings.TrimSpace(reason)}, workspace.CreatedAt); err != nil {
		return domain.Workspace{}, domain.WorkspaceMembership{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Workspace{}, domain.WorkspaceMembership{}, fmt.Errorf("commit workspace creation: %w", err)
	}
	return workspace, domain.WorkspaceMembership{ID: membershipID, WorkspaceID: workspace.ID, UserID: actorUserID, Role: domain.WorkspaceAdmin, Active: true, AssignedBy: actorUserID, AssignedAt: workspace.CreatedAt}, nil
}

func (s *Store) GetWorkspace(ctx context.Context, requesterUserID, workspaceID string) (*domain.Workspace, error) {
	if _, err := s.authorisedWorkspaceRead(ctx, requesterUserID, workspaceID, false); err != nil {
		return nil, err
	}
	var workspace domain.Workspace
	var status, created string
	var archived sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT w.public_id,o.public_id,w.name,w.status,w.created_at,w.archived_at
		FROM workspaces w JOIN organisations o ON o.id=w.organisation_id
		WHERE w.public_id=? AND w.status='active' AND o.status='active'`, workspaceID).Scan(&workspace.ID, &workspace.OrganisationID, &workspace.Name, &status, &created, &archived); err != nil {
		return nil, fmt.Errorf("read workspace: %w", err)
	}
	workspace.Status = domain.WorkspaceStatus(status)
	workspace.CreatedAt = parseTime(created)
	if archived.Valid {
		value := parseTime(archived.String)
		workspace.ArchivedAt = &value
	}
	return &workspace, nil
}

func (s *Store) GetWorkspaceMembership(ctx context.Context, requesterUserID, workspaceID, targetUserID string) (*domain.WorkspaceMembership, error) {
	adminOnly := strings.TrimSpace(requesterUserID) != strings.TrimSpace(targetUserID)
	if _, err := s.authorisedWorkspaceRead(ctx, requesterUserID, workspaceID, adminOnly); err != nil {
		return nil, err
	}
	var membership domain.WorkspaceMembership
	var role, assignedAt string
	err := s.db.QueryRowContext(ctx, `SELECT m.public_id,w.public_id,u.public_id,m.role,m.active,assigned.public_id,m.assigned_at
		FROM workspace_memberships m
		JOIN workspaces w ON w.id=m.workspace_id AND w.status='active'
		JOIN organisations o ON o.id=w.organisation_id AND o.status='active'
		JOIN users u ON u.id=m.user_id AND u.active=1
		JOIN users assigned ON assigned.id=m.assigned_by
		WHERE w.public_id=? AND u.public_id=? AND m.active=1`, workspaceID, targetUserID).
		Scan(&membership.ID, &membership.WorkspaceID, &membership.UserID, &role, &membership.Active, &membership.AssignedBy, &assignedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrWorkspaceMembershipNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read workspace membership: %w", err)
	}
	membership.Role = domain.WorkspaceRole(role)
	membership.AssignedAt = parseTime(assignedAt)
	return &membership, nil
}

func (s *Store) ListOrganisationWorkspaces(ctx context.Context, requesterUserID, organisationID string) ([]domain.Workspace, error) {
	organisationInternalID, err := s.authorisedOrganisationRead(ctx, requesterUserID, organisationID, false)
	if err != nil {
		return nil, err
	}
	var role string
	if err := s.db.QueryRowContext(ctx, `SELECT role FROM organisation_memberships WHERE organisation_id=? AND user_id=(SELECT id FROM users WHERE public_id=?) AND active=1`, organisationInternalID, requesterUserID).Scan(&role); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("resolve workspace listing role: %w", err)
	}
	var systemAdmin int
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM system_role_assignments WHERE user_id=(SELECT id FROM users WHERE public_id=?) AND role='system_admin' AND active=1)`, requesterUserID).Scan(&systemAdmin); err != nil {
		return nil, fmt.Errorf("resolve workspace listing system role: %w", err)
	}
	query := `SELECT w.public_id,o.public_id,w.name,w.status,w.created_at,w.archived_at
		FROM workspaces w JOIN organisations o ON o.id=w.organisation_id AND o.status='active'
		WHERE w.organisation_id=? AND w.status='active'`
	args := []any{organisationInternalID}
	if systemAdmin != 1 && !domain.OrganisationRole(role).CanAdminister() {
		query += ` AND EXISTS (SELECT 1 FROM workspace_memberships wm JOIN users wu ON wu.id=wm.user_id AND wu.active=1 WHERE wm.workspace_id=w.id AND wm.user_id=(SELECT id FROM users WHERE public_id=?) AND wm.active=1)`
		args = append(args, requesterUserID)
	}
	query += ` ORDER BY w.name,w.public_id`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list organisation workspaces: %w", err)
	}
	defer rows.Close()
	workspaces := make([]domain.Workspace, 0)
	for rows.Next() {
		var workspace domain.Workspace
		var status, created string
		var archived sql.NullString
		if err := rows.Scan(&workspace.ID, &workspace.OrganisationID, &workspace.Name, &status, &created, &archived); err != nil {
			return nil, fmt.Errorf("read organisation workspace: %w", err)
		}
		workspace.Status = domain.WorkspaceStatus(status)
		workspace.CreatedAt = parseTime(created)
		if archived.Valid {
			value := parseTime(archived.String)
			workspace.ArchivedAt = &value
		}
		workspaces = append(workspaces, workspace)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate organisation workspaces: %w", err)
	}
	return workspaces, nil
}

func (s *Store) AddWorkspaceMember(ctx context.Context, actorUserID, workspaceID, targetUserID string, role domain.WorkspaceRole, reason string, at time.Time) (domain.WorkspaceMembership, error) {
	if !role.Valid() {
		return domain.WorkspaceMembership{}, domain.ErrInvalidWorkspaceRole
	}
	if err := validateMutationReason(reason); err != nil {
		return domain.WorkspaceMembership{}, err
	}
	if at.IsZero() {
		return domain.WorkspaceMembership{}, errors.New("workspace membership assignment time is required")
	}
	membershipID, err := domain.NewWorkspaceMembershipID()
	if err != nil {
		return domain.WorkspaceMembership{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.WorkspaceMembership{}, fmt.Errorf("begin workspace membership assignment: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	actorInternalID, _, workspaceInternalID, _, err := authorisedWorkspaceMutationTx(ctx, tx, actorUserID, workspaceID, true)
	if err != nil {
		return domain.WorkspaceMembership{}, err
	}
	if err := activeWorkspaceTargetUserTx(ctx, tx, workspaceInternalID, targetUserID); err != nil {
		return domain.WorkspaceMembership{}, err
	}
	var active int
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_memberships wm JOIN users u ON u.id=wm.user_id AND u.active=1 WHERE wm.workspace_id=? AND u.public_id=? AND wm.active=1)`, workspaceInternalID, targetUserID).Scan(&active); err != nil {
		return domain.WorkspaceMembership{}, fmt.Errorf("check workspace membership duplicate: %w", err)
	}
	if active == 1 {
		return domain.WorkspaceMembership{}, ErrDuplicateWorkspaceMember
	}
	targetInternalID, err := targetUserIDInternal(ctx, tx, targetUserID)
	if err != nil {
		return domain.WorkspaceMembership{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO workspace_memberships(public_id,workspace_id,user_id,role,active,assigned_by,assigned_at) VALUES(?,?,?,?,1,?,?)`, membershipID, workspaceInternalID, targetInternalID, string(role), actorInternalID, formatTime(at.UTC())); err != nil {
		return domain.WorkspaceMembership{}, fmt.Errorf("assign workspace membership: %w", err)
	}
	if err := recordWorkspaceAuditTx(ctx, tx, actorInternalID, workspaceID, eventWorkspaceMembershipAdded, workspaceMutationDetails{WorkspaceID: workspaceID, MembershipID: membershipID, UserID: targetUserID, Role: string(role), Reason: strings.TrimSpace(reason)}, at); err != nil {
		return domain.WorkspaceMembership{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.WorkspaceMembership{}, fmt.Errorf("commit workspace membership assignment: %w", err)
	}
	return domain.WorkspaceMembership{ID: membershipID, WorkspaceID: workspaceID, UserID: targetUserID, Role: role, Active: true, AssignedBy: actorUserID, AssignedAt: at.UTC()}, nil
}

func (s *Store) ChangeWorkspaceMemberRole(ctx context.Context, actorUserID, workspaceID, targetUserID string, role domain.WorkspaceRole, reason string, at time.Time) (domain.WorkspaceMembership, error) {
	if !role.Valid() {
		return domain.WorkspaceMembership{}, domain.ErrInvalidWorkspaceRole
	}
	if err := validateMutationReason(reason); err != nil {
		return domain.WorkspaceMembership{}, err
	}
	if at.IsZero() {
		return domain.WorkspaceMembership{}, errors.New("workspace membership role-change time is required")
	}
	membershipID, err := domain.NewWorkspaceMembershipID()
	if err != nil {
		return domain.WorkspaceMembership{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.WorkspaceMembership{}, fmt.Errorf("begin workspace membership role change: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	actorInternalID, _, workspaceInternalID, _, err := authorisedWorkspaceMutationTx(ctx, tx, actorUserID, workspaceID, true)
	if err != nil {
		return domain.WorkspaceMembership{}, err
	}
	if err := activeWorkspaceTargetUserTx(ctx, tx, workspaceInternalID, targetUserID); err != nil {
		return domain.WorkspaceMembership{}, err
	}
	var currentID int64
	var currentRole string
	if err := tx.QueryRowContext(ctx, `SELECT m.id,m.role FROM workspace_memberships m JOIN users u ON u.id=m.user_id AND u.active=1 WHERE m.workspace_id=? AND u.public_id=? AND m.active=1`, workspaceInternalID, targetUserID).Scan(&currentID, &currentRole); errors.Is(err, sql.ErrNoRows) {
		return domain.WorkspaceMembership{}, ErrWorkspaceMembershipNotFound
	} else if err != nil {
		return domain.WorkspaceMembership{}, fmt.Errorf("read workspace membership role: %w", err)
	}
	if currentRole == string(role) {
		return domain.WorkspaceMembership{}, ErrDuplicateWorkspaceMember
	}
	if _, err := tx.ExecContext(ctx, `UPDATE workspace_memberships SET active=0,removed_by=?,removed_at=?,removal_reason=? WHERE id=? AND active=1`, actorInternalID, formatTime(at.UTC()), strings.TrimSpace(reason), currentID); err != nil {
		return domain.WorkspaceMembership{}, fmt.Errorf("retire workspace membership role: %w", err)
	}
	targetInternalID, err := targetUserIDInternal(ctx, tx, targetUserID)
	if err != nil {
		return domain.WorkspaceMembership{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO workspace_memberships(public_id,workspace_id,user_id,role,active,assigned_by,assigned_at) VALUES(?,?,?,?,1,?,?)`, membershipID, workspaceInternalID, targetInternalID, string(role), actorInternalID, formatTime(at.UTC())); err != nil {
		return domain.WorkspaceMembership{}, fmt.Errorf("write workspace membership role: %w", err)
	}
	if err := recordWorkspaceAuditTx(ctx, tx, actorInternalID, workspaceID, eventWorkspaceRoleChanged, workspaceMutationDetails{WorkspaceID: workspaceID, MembershipID: membershipID, UserID: targetUserID, Role: string(role), Reason: strings.TrimSpace(reason)}, at); err != nil {
		return domain.WorkspaceMembership{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.WorkspaceMembership{}, fmt.Errorf("commit workspace membership role change: %w", err)
	}
	return domain.WorkspaceMembership{ID: membershipID, WorkspaceID: workspaceID, UserID: targetUserID, Role: role, Active: true, AssignedBy: actorUserID, AssignedAt: at.UTC()}, nil
}

func (s *Store) DeactivateWorkspaceMember(ctx context.Context, actorUserID, workspaceID, targetUserID, reason string, at time.Time) error {
	if err := validateMutationReason(reason); err != nil {
		return err
	}
	if at.IsZero() {
		return errors.New("workspace membership deactivation time is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin workspace membership deactivation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	actorInternalID, _, workspaceInternalID, _, err := authorisedWorkspaceMutationTx(ctx, tx, actorUserID, workspaceID, true)
	if err != nil {
		return err
	}
	var membershipID int64
	var membershipPublicID string
	var targetRole string
	if err := tx.QueryRowContext(ctx, `SELECT m.id,m.public_id,m.role FROM workspace_memberships m JOIN users u ON u.id=m.user_id AND u.active=1 WHERE m.workspace_id=? AND u.public_id=? AND m.active=1`, workspaceInternalID, targetUserID).Scan(&membershipID, &membershipPublicID, &targetRole); errors.Is(err, sql.ErrNoRows) {
		return ErrWorkspaceMembershipNotFound
	} else if err != nil {
		return fmt.Errorf("read workspace membership for deactivation: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE workspace_memberships SET active=0,removed_by=?,removed_at=?,removal_reason=? WHERE id=? AND active=1`, actorInternalID, formatTime(at.UTC()), strings.TrimSpace(reason), membershipID)
	if err != nil {
		return fmt.Errorf("deactivate workspace membership: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("check workspace membership deactivation: %w", err)
	} else if rows != 1 {
		return ErrWorkspaceMembershipNotFound
	}
	if err := recordWorkspaceAuditTx(ctx, tx, actorInternalID, workspaceID, eventWorkspaceMemberRemoved, workspaceMutationDetails{WorkspaceID: workspaceID, MembershipID: membershipPublicID, UserID: targetUserID, Role: targetRole, Reason: strings.TrimSpace(reason)}, at); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit workspace membership deactivation: %w", err)
	}
	return nil
}

func (s *Store) ArchiveWorkspace(ctx context.Context, actorUserID, workspaceID, reason string, at time.Time) error {
	if err := validateMutationReason(reason); err != nil {
		return err
	}
	if at.IsZero() {
		return errors.New("workspace archive time is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin workspace archive: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	actorInternalID, _, workspaceInternalID, _, err := authorisedWorkspaceMutationTx(ctx, tx, actorUserID, workspaceID, false)
	if err != nil {
		return err
	}
	var orgRole string
	if err := tx.QueryRowContext(ctx, `SELECT role FROM organisation_memberships WHERE organisation_id=(SELECT organisation_id FROM workspaces WHERE id=?) AND user_id=? AND active=1`, workspaceInternalID, actorInternalID).Scan(&orgRole); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("resolve workspace archive authority: %w", err)
	}
	var systemAdmin int
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM system_role_assignments WHERE user_id=? AND role='system_admin' AND active=1)`, actorInternalID).Scan(&systemAdmin); err != nil {
		return fmt.Errorf("resolve workspace archive system role: %w", err)
	}
	if systemAdmin != 1 && !domain.OrganisationRole(orgRole).CanAdminister() {
		return ErrUnauthorisedWorkspaceAction
	}
	result, err := tx.ExecContext(ctx, `UPDATE workspaces SET status='archived',archived_at=? WHERE id=? AND status='active'`, formatTime(at.UTC()), workspaceInternalID)
	if err != nil {
		return fmt.Errorf("archive workspace: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("check workspace archive: %w", err)
	} else if rows != 1 {
		return ErrWorkspaceArchived
	}
	rows, err := tx.QueryContext(ctx, `SELECT m.public_id,u.public_id,m.role FROM workspace_memberships m JOIN users u ON u.id=m.user_id WHERE m.workspace_id=? AND m.active=1`, workspaceInternalID)
	if err != nil {
		return fmt.Errorf("read workspace memberships for archive: %w", err)
	}
	type activeMembership struct{ membershipID, userID, role string }
	var memberships []activeMembership
	for rows.Next() {
		var membership activeMembership
		if err := rows.Scan(&membership.membershipID, &membership.userID, &membership.role); err != nil {
			_ = rows.Close()
			return fmt.Errorf("read workspace membership for archive: %w", err)
		}
		memberships = append(memberships, membership)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterate workspace memberships for archive: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close workspace memberships for archive: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE workspace_memberships SET active=0,removed_by=?,removed_at=?,removal_reason=? WHERE workspace_id=? AND active=1`, actorInternalID, formatTime(at.UTC()), strings.TrimSpace(reason), workspaceInternalID); err != nil {
		return fmt.Errorf("deactivate archived workspace memberships: %w", err)
	}
	for _, membership := range memberships {
		if err := recordWorkspaceAuditTx(ctx, tx, actorInternalID, workspaceID, eventWorkspaceMemberRemoved, workspaceMutationDetails{WorkspaceID: workspaceID, MembershipID: membership.membershipID, UserID: membership.userID, Role: membership.role, Reason: strings.TrimSpace(reason)}, at); err != nil {
			return err
		}
	}
	if err := recordWorkspaceAuditTx(ctx, tx, actorInternalID, workspaceID, eventWorkspaceArchived, workspaceMutationDetails{WorkspaceID: workspaceID, Reason: strings.TrimSpace(reason)}, at); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit workspace archive: %w", err)
	}
	return nil
}

func (s *Store) ListWorkspaceMembers(ctx context.Context, requesterUserID, workspaceID string) ([]store.WorkspaceMemberRecord, error) {
	workspaceInternalID, err := s.authorisedWorkspaceRead(ctx, requesterUserID, workspaceID, true)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT m.public_id,w.public_id,u.public_id,u.email,u.display_name,m.role,m.assigned_at
		FROM workspace_memberships m
		JOIN workspaces w ON w.id=m.workspace_id AND w.status='active'
		JOIN organisations o ON o.id=w.organisation_id AND o.status='active'
		JOIN users u ON u.id=m.user_id AND u.active=1
		WHERE m.workspace_id=? AND m.active=1 ORDER BY u.display_name,u.public_id`, workspaceInternalID)
	if err != nil {
		return nil, fmt.Errorf("list workspace members: %w", err)
	}
	defer rows.Close()
	members := make([]store.WorkspaceMemberRecord, 0)
	for rows.Next() {
		var member store.WorkspaceMemberRecord
		var role, assignedAt string
		if err := rows.Scan(&member.MembershipID, &member.WorkspaceID, &member.UserID, &member.Email, &member.DisplayName, &role, &assignedAt); err != nil {
			return nil, fmt.Errorf("read workspace member: %w", err)
		}
		member.Role = domain.WorkspaceRole(role)
		member.AssignedAt = parseTime(assignedAt)
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace members: %w", err)
	}
	return members, nil
}

func (s *Store) ListWorkspaceAudit(ctx context.Context, requesterUserID, workspaceID string, limit int) ([]store.WorkspaceAuditRecord, error) {
	workspaceInternalID, err := s.authorisedWorkspaceRead(ctx, requesterUserID, workspaceID, true)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		return nil, errors.New("workspace audit limit must be between 1 and 100")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT a.id,w.public_id,actor.public_id,a.event,a.details,a.occurred_at
		FROM audit_events a
		JOIN workspaces w ON w.public_id=a.aggregate_id AND w.id=?
		JOIN users actor ON actor.id=a.actor_user_id
		WHERE a.aggregate_type=? AND a.aggregate_id=?
		ORDER BY a.occurred_at DESC,a.id DESC LIMIT ?`, workspaceInternalID, auditWorkspace, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("list workspace audit: %w", err)
	}
	defer rows.Close()
	events := make([]store.WorkspaceAuditRecord, 0)
	for rows.Next() {
		var event store.WorkspaceAuditRecord
		var occurredAt string
		if err := rows.Scan(&event.ID, &event.WorkspaceID, &event.ActorUserID, &event.Event, &event.Details, &occurredAt); err != nil {
			return nil, fmt.Errorf("read workspace audit: %w", err)
		}
		event.OccurredAt = parseTime(occurredAt)
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace audit: %w", err)
	}
	return events, nil
}

func (s *Store) authorisedWorkspaceRead(ctx context.Context, requesterUserID, workspaceID string, adminOnly bool) (int64, error) {
	proof, err := proveWorkspaceScopeQuery(ctx, s.db, requesterUserID, workspaceID, adminOnly)
	if err != nil {
		return 0, s.classifyWorkspaceScopeError(ctx, workspaceID, err)
	}
	return proof.workspaceInternalID, nil
}

func authorisedWorkspaceMutationTx(ctx context.Context, tx *sql.Tx, actorUserID, workspaceID string, allowWorkspaceAdmin bool) (int64, int64, int64, bool, error) {
	proof, err := proveWorkspaceScopeQuery(ctx, tx, actorUserID, workspaceID, true)
	if err != nil {
		return 0, 0, 0, false, classifyWorkspaceScopeErrorTx(ctx, tx, workspaceID, err)
	}
	if !allowWorkspaceAdmin && !proof.scope.SystemAdmin && !proof.scope.OrganisationRole.CanAdminister() {
		return 0, 0, 0, false, ErrUnauthorisedWorkspaceAction
	}
	return proof.userInternalID, proof.organisationInternalID, proof.workspaceInternalID, proof.scope.SystemAdmin, nil
}

func (s *Store) classifyWorkspaceScopeError(ctx context.Context, workspaceID string, err error) error {
	if !errors.Is(err, store.ErrAccessScopeDenied) {
		return err
	}
	var status, organisationStatus string
	if lookupErr := s.db.QueryRowContext(ctx, `SELECT w.status,o.status FROM workspaces w JOIN organisations o ON o.id=w.organisation_id WHERE w.public_id=?`, workspaceID).Scan(&status, &organisationStatus); errors.Is(lookupErr, sql.ErrNoRows) {
		return ErrWorkspaceNotFound
	} else if lookupErr != nil {
		return fmt.Errorf("classify workspace scope: %w", lookupErr)
	}
	if status != string(domain.WorkspaceActive) || organisationStatus != string(domain.OrganisationActive) {
		return ErrWorkspaceArchived
	}
	return ErrUnauthorisedWorkspaceAction
}

func classifyWorkspaceScopeErrorTx(ctx context.Context, tx *sql.Tx, workspaceID string, err error) error {
	if !errors.Is(err, store.ErrAccessScopeDenied) {
		return err
	}
	var status, organisationStatus string
	if lookupErr := tx.QueryRowContext(ctx, `SELECT w.status,o.status FROM workspaces w JOIN organisations o ON o.id=w.organisation_id WHERE w.public_id=?`, workspaceID).Scan(&status, &organisationStatus); errors.Is(lookupErr, sql.ErrNoRows) {
		return ErrWorkspaceNotFound
	} else if lookupErr != nil {
		return fmt.Errorf("classify workspace scope in transaction: %w", lookupErr)
	}
	if status != string(domain.WorkspaceActive) || organisationStatus != string(domain.OrganisationActive) {
		return ErrWorkspaceArchived
	}
	return ErrUnauthorisedWorkspaceAction
}

func activeWorkspaceTargetUserTx(ctx context.Context, tx *sql.Tx, workspaceInternalID int64, targetUserID string) error {
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users u JOIN organisations o ON o.id=(SELECT organisation_id FROM workspaces WHERE id=?) AND o.status='active' JOIN organisation_memberships om ON om.organisation_id=o.id AND om.user_id=u.id AND om.active=1 WHERE u.public_id=? AND u.active=1)`, workspaceInternalID, targetUserID).Scan(&exists); err != nil {
		return fmt.Errorf("resolve workspace member organisation: %w", err)
	}
	if exists != 1 {
		return ErrUnauthorisedWorkspaceAction
	}
	return nil
}

func targetUserIDInternal(ctx context.Context, tx *sql.Tx, targetUserID string) (int64, error) {
	var internalID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE public_id=? AND active=1`, targetUserID).Scan(&internalID); err != nil {
		return 0, fmt.Errorf("resolve workspace member user: %w", err)
	}
	return internalID, nil
}

func recordWorkspaceAuditTx(ctx context.Context, tx *sql.Tx, actorInternalID int64, workspaceID, event string, details workspaceMutationDetails, at time.Time) error {
	payload, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("encode workspace audit: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_events(actor_user_id,aggregate_type,aggregate_id,event,details,occurred_at) VALUES(?,?,?,?,?,?)`, actorInternalID, auditWorkspace, workspaceID, event, string(payload), formatTime(at.UTC())); err != nil {
		return fmt.Errorf("record workspace audit: %w", err)
	}
	var organisationID string
	if err := tx.QueryRowContext(ctx, `SELECT o.public_id FROM workspaces w JOIN organisations o ON o.id=w.organisation_id WHERE w.public_id=?`, workspaceID).Scan(&organisationID); err != nil {
		return fmt.Errorf("resolve workspace event organisation: %w", err)
	}
	if err := appendWorkspaceEventTx(ctx, tx, event, details, organisationID, at); err != nil {
		return fmt.Errorf("record workspace outbox event: %w", err)
	}
	return nil
}

func (s *Store) workspaceAuditCount(ctx context.Context, workspaceID, event string) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE aggregate_type=? AND aggregate_id=? AND event=?`, auditWorkspace, workspaceID, event).Scan(&count); err != nil {
		return 0, fmt.Errorf("count workspace audit: %w", err)
	}
	return count, nil
}
