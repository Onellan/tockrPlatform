package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Onellan/tockrplatform/internal/domain"
	"github.com/Onellan/tockrplatform/internal/store"
)

func (s *Store) ExecuteMembershipCommand(ctx context.Context, request store.MembershipCommandRequest) (store.MembershipCommandResult, error) {
	if strings.TrimSpace(request.Consumer) == "" || strings.TrimSpace(request.RequestHash) == "" || strings.TrimSpace(request.IdempotencyKey) == "" || request.OccurredAt.IsZero() {
		return store.MembershipCommandResult{}, store.ErrMembershipCommandConflict
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.MembershipCommandResult{}, fmt.Errorf("begin membership command: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var storedHash, storedResult string
	err = tx.QueryRowContext(ctx, `SELECT request_hash,result_json FROM platform_membership_command_results WHERE consumer_key=? AND idempotency_key=?`, request.Consumer, request.IdempotencyKey).Scan(&storedHash, &storedResult)
	if err == nil {
		if storedHash != request.RequestHash {
			return store.MembershipCommandResult{}, store.ErrMembershipCommandReplayMismatch
		}
		if err := validateMembershipCommandActorTx(ctx, tx, request); err != nil {
			return store.MembershipCommandResult{}, err
		}
		var result store.MembershipCommandResult
		if err := json.Unmarshal([]byte(storedResult), &result); err != nil {
			return store.MembershipCommandResult{}, fmt.Errorf("decode stored membership command result: %w", err)
		}
		result.Replay = true
		return result, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return store.MembershipCommandResult{}, fmt.Errorf("read membership command result: %w", err)
	}

	request.Reason = strings.TrimSpace(request.Reason)
	request.OccurredAt = request.OccurredAt.UTC()
	var result store.MembershipCommandResult
	switch request.Scope {
	case "organisation_membership":
		result, err = s.executeOrganisationMembershipCommandTx(ctx, tx, request)
	case "workspace_membership":
		result, err = s.executeWorkspaceMembershipCommandTx(ctx, tx, request)
	default:
		return store.MembershipCommandResult{}, store.ErrMembershipCommandUnsupported
	}
	if err != nil {
		return store.MembershipCommandResult{}, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return store.MembershipCommandResult{}, fmt.Errorf("encode membership command result: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO platform_membership_command_results(consumer_key,idempotency_key,request_hash,result_json,created_at) VALUES(?,?,?,?,?)`, request.Consumer, request.IdempotencyKey, request.RequestHash, string(encoded), formatTime(request.OccurredAt)); err != nil {
		return store.MembershipCommandResult{}, fmt.Errorf("record membership command result: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return store.MembershipCommandResult{}, fmt.Errorf("commit membership command: %w", err)
	}
	return result, nil
}

func validateMembershipCommandActorTx(ctx context.Context, tx *sql.Tx, request store.MembershipCommandRequest) error {
	switch request.Scope {
	case "organisation_membership":
		_, _, _, _, err := authorisedOrganisationMutationTx(ctx, tx, request.ActorUserID, request.OrganisationID)
		return err
	case "workspace_membership":
		_, _, _, _, err := authorisedWorkspaceMutationTx(ctx, tx, request.ActorUserID, request.WorkspaceID, true)
		return err
	default:
		return store.ErrMembershipCommandUnsupported
	}
}

func (s *Store) executeOrganisationMembershipCommandTx(ctx context.Context, tx *sql.Tx, request store.MembershipCommandRequest) (store.MembershipCommandResult, error) {
	actorInternalID, actorRole, organisationInternalID, systemAdmin, err := authorisedOrganisationMutationTx(ctx, tx, request.ActorUserID, request.OrganisationID)
	if err != nil {
		return store.MembershipCommandResult{}, err
	}
	role := domain.OrganisationRole(request.Role)
	switch request.Operation {
	case "add":
		if role != domain.OrganisationAdmin && role != domain.OrganisationMember {
			return store.MembershipCommandResult{}, domain.ErrInvalidOrganisationRole
		}
		if !systemAdmin && actorRole == domain.OrganisationAdmin && role == domain.OrganisationAdmin {
			return store.MembershipCommandResult{}, ErrUnauthorisedOrganisationAction
		}
		targetID, err := activeUserIDTx(ctx, tx, request.UserID)
		if err != nil {
			return store.MembershipCommandResult{}, err
		}
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM organisation_memberships WHERE organisation_id=? AND user_id=? AND active=1`, organisationInternalID, targetID).Scan(&exists); err != nil {
			return store.MembershipCommandResult{}, err
		}
		if exists != 0 {
			return store.MembershipCommandResult{}, ErrDuplicateOrganisationMember
		}
		membershipID, err := domain.NewOrganisationMembershipID()
		if err != nil {
			return store.MembershipCommandResult{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO organisation_memberships(public_id,organisation_id,user_id,role,active,assigned_by,assigned_at,membership_version) VALUES(?,?,?,?,1,?,?,1)`, membershipID, organisationInternalID, targetID, string(role), actorInternalID, formatTime(request.OccurredAt)); err != nil {
			return store.MembershipCommandResult{}, err
		}
		details := organisationMutationDetails{OrganisationID: request.OrganisationID, MembershipID: membershipID, UserID: request.UserID, Role: string(role), Reason: request.Reason}
		if err := recordOrganisationAuditTx(ctx, tx, actorInternalID, request.OrganisationID, eventMembershipAdded, details, request.OccurredAt); err != nil {
			return store.MembershipCommandResult{}, err
		}
		return store.MembershipCommandResult{MembershipID: membershipID, OrganisationID: request.OrganisationID, UserID: request.UserID, Role: string(role), Active: true, MembershipVersion: 1}, nil
	case "change_role":
		if role != domain.OrganisationAdmin && role != domain.OrganisationMember {
			return store.MembershipCommandResult{}, domain.ErrInvalidOrganisationRole
		}
		var currentID, currentVersion int64
		var currentPublicID, currentRole string
		if err := tx.QueryRowContext(ctx, `SELECT m.id,m.public_id,m.membership_version,m.role FROM organisation_memberships m JOIN users u ON u.id=m.user_id AND u.active=1 WHERE m.organisation_id=? AND u.public_id=? AND m.active=1`, organisationInternalID, request.UserID).Scan(&currentID, &currentPublicID, &currentVersion, &currentRole); errors.Is(err, sql.ErrNoRows) {
			return store.MembershipCommandResult{}, ErrMembershipNotFound
		} else if err != nil {
			return store.MembershipCommandResult{}, err
		}
		if request.ExpectedVersion != currentVersion {
			return store.MembershipCommandResult{}, store.ErrMembershipCommandConflict
		}
		if currentRole == string(domain.OrganisationOwner) {
			return store.MembershipCommandResult{}, ErrOwnerMutationNotAuthorised
		}
		if !systemAdmin && actorRole == domain.OrganisationAdmin && currentRole != string(domain.OrganisationMember) {
			return store.MembershipCommandResult{}, ErrUnauthorisedOrganisationAction
		}
		if currentRole == string(role) {
			return store.MembershipCommandResult{}, ErrDuplicateOrganisationMember
		}
		targetID, err := activeUserIDTx(ctx, tx, request.UserID)
		if err != nil {
			return store.MembershipCommandResult{}, err
		}
		membershipID, err := domain.NewOrganisationMembershipID()
		if err != nil {
			return store.MembershipCommandResult{}, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE organisation_memberships SET active=0,removed_by=?,removed_at=?,removal_reason=? WHERE id=? AND active=1`, actorInternalID, formatTime(request.OccurredAt), request.Reason, currentID); err != nil {
			return store.MembershipCommandResult{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO organisation_memberships(public_id,organisation_id,user_id,role,active,assigned_by,assigned_at,membership_version) VALUES(?,?,?,?,1,?,?,?)`, membershipID, organisationInternalID, targetID, string(role), actorInternalID, formatTime(request.OccurredAt), currentVersion+1); err != nil {
			return store.MembershipCommandResult{}, err
		}
		details := organisationMutationDetails{OrganisationID: request.OrganisationID, MembershipID: membershipID, UserID: request.UserID, Role: string(role), Reason: request.Reason}
		if err := recordOrganisationAuditTx(ctx, tx, actorInternalID, request.OrganisationID, eventMembershipRoleChanged, details, request.OccurredAt); err != nil {
			return store.MembershipCommandResult{}, err
		}
		return store.MembershipCommandResult{MembershipID: membershipID, OrganisationID: request.OrganisationID, UserID: request.UserID, Role: string(role), Active: true, MembershipVersion: currentVersion + 1}, nil
	case "deactivate":
		var currentID, currentVersion int64
		var currentPublicID, currentRole string
		if err := tx.QueryRowContext(ctx, `SELECT m.id,m.public_id,m.membership_version,m.role FROM organisation_memberships m JOIN users u ON u.id=m.user_id AND u.active=1 WHERE m.organisation_id=? AND u.public_id=? AND m.active=1`, organisationInternalID, request.UserID).Scan(&currentID, &currentPublicID, &currentVersion, &currentRole); errors.Is(err, sql.ErrNoRows) {
			return store.MembershipCommandResult{}, ErrMembershipNotFound
		} else if err != nil {
			return store.MembershipCommandResult{}, err
		}
		if request.ExpectedVersion != currentVersion {
			return store.MembershipCommandResult{}, store.ErrMembershipCommandConflict
		}
		if currentRole == string(domain.OrganisationOwner) {
			return store.MembershipCommandResult{}, ErrOwnerMutationNotAuthorised
		}
		if !systemAdmin && actorRole == domain.OrganisationAdmin && currentRole != string(domain.OrganisationMember) {
			return store.MembershipCommandResult{}, ErrUnauthorisedOrganisationAction
		}
		if _, err := tx.ExecContext(ctx, `UPDATE organisation_memberships SET active=0,removed_by=?,removed_at=?,removal_reason=? WHERE id=? AND active=1`, actorInternalID, formatTime(request.OccurredAt), request.Reason, currentID); err != nil {
			return store.MembershipCommandResult{}, err
		}
		details := organisationMutationDetails{OrganisationID: request.OrganisationID, MembershipID: currentPublicID, UserID: request.UserID, Role: currentRole, Reason: request.Reason}
		if err := recordOrganisationAuditTx(ctx, tx, actorInternalID, request.OrganisationID, eventMembershipDeactivated, details, request.OccurredAt); err != nil {
			return store.MembershipCommandResult{}, err
		}
		return store.MembershipCommandResult{MembershipID: currentPublicID, OrganisationID: request.OrganisationID, UserID: request.UserID, Role: currentRole, Active: false, MembershipVersion: currentVersion}, nil
	default:
		return store.MembershipCommandResult{}, store.ErrMembershipCommandUnsupported
	}
}

func (s *Store) executeWorkspaceMembershipCommandTx(ctx context.Context, tx *sql.Tx, request store.MembershipCommandRequest) (store.MembershipCommandResult, error) {
	actorInternalID, _, workspaceInternalID, _, err := authorisedWorkspaceMutationTx(ctx, tx, request.ActorUserID, request.WorkspaceID, true)
	if err != nil {
		return store.MembershipCommandResult{}, err
	}
	role := domain.WorkspaceRole(request.Role)
	switch request.Operation {
	case "add":
		if !role.Valid() {
			return store.MembershipCommandResult{}, domain.ErrInvalidWorkspaceRole
		}
		if err := activeWorkspaceTargetUserTx(ctx, tx, workspaceInternalID, request.UserID); err != nil {
			return store.MembershipCommandResult{}, err
		}
		var active int
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_memberships wm JOIN users u ON u.id=wm.user_id AND u.active=1 WHERE wm.workspace_id=? AND u.public_id=? AND wm.active=1)`, workspaceInternalID, request.UserID).Scan(&active); err != nil {
			return store.MembershipCommandResult{}, err
		}
		if active == 1 {
			return store.MembershipCommandResult{}, ErrDuplicateWorkspaceMember
		}
		targetID, err := targetUserIDInternal(ctx, tx, request.UserID)
		if err != nil {
			return store.MembershipCommandResult{}, err
		}
		membershipID, err := domain.NewWorkspaceMembershipID()
		if err != nil {
			return store.MembershipCommandResult{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO workspace_memberships(public_id,workspace_id,user_id,role,active,assigned_by,assigned_at,membership_version) VALUES(?,?,?,?,1,?,?,1)`, membershipID, workspaceInternalID, targetID, string(role), actorInternalID, formatTime(request.OccurredAt)); err != nil {
			return store.MembershipCommandResult{}, err
		}
		details := workspaceMutationDetails{WorkspaceID: request.WorkspaceID, MembershipID: membershipID, UserID: request.UserID, Role: string(role), Reason: request.Reason}
		if err := recordWorkspaceAuditTx(ctx, tx, actorInternalID, request.WorkspaceID, eventWorkspaceMembershipAdded, details, request.OccurredAt); err != nil {
			return store.MembershipCommandResult{}, err
		}
		return store.MembershipCommandResult{MembershipID: membershipID, WorkspaceID: request.WorkspaceID, UserID: request.UserID, Role: string(role), Active: true, MembershipVersion: 1}, nil
	case "change_role":
		if !role.Valid() {
			return store.MembershipCommandResult{}, domain.ErrInvalidWorkspaceRole
		}
		var currentID, currentVersion int64
		var currentPublicID, currentRole string
		if err := tx.QueryRowContext(ctx, `SELECT m.id,m.public_id,m.membership_version,m.role FROM workspace_memberships m JOIN users u ON u.id=m.user_id AND u.active=1 WHERE m.workspace_id=? AND u.public_id=? AND m.active=1`, workspaceInternalID, request.UserID).Scan(&currentID, &currentPublicID, &currentVersion, &currentRole); errors.Is(err, sql.ErrNoRows) {
			return store.MembershipCommandResult{}, ErrWorkspaceMembershipNotFound
		} else if err != nil {
			return store.MembershipCommandResult{}, err
		}
		if request.ExpectedVersion != currentVersion {
			return store.MembershipCommandResult{}, store.ErrMembershipCommandConflict
		}
		if currentRole == string(role) {
			return store.MembershipCommandResult{}, ErrDuplicateWorkspaceMember
		}
		targetID, err := targetUserIDInternal(ctx, tx, request.UserID)
		if err != nil {
			return store.MembershipCommandResult{}, err
		}
		membershipID, err := domain.NewWorkspaceMembershipID()
		if err != nil {
			return store.MembershipCommandResult{}, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE workspace_memberships SET active=0,removed_by=?,removed_at=?,removal_reason=? WHERE id=? AND active=1`, actorInternalID, formatTime(request.OccurredAt), request.Reason, currentID); err != nil {
			return store.MembershipCommandResult{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO workspace_memberships(public_id,workspace_id,user_id,role,active,assigned_by,assigned_at,membership_version) VALUES(?,?,?,?,1,?,?,?)`, membershipID, workspaceInternalID, targetID, string(role), actorInternalID, formatTime(request.OccurredAt), currentVersion+1); err != nil {
			return store.MembershipCommandResult{}, err
		}
		details := workspaceMutationDetails{WorkspaceID: request.WorkspaceID, MembershipID: membershipID, UserID: request.UserID, Role: string(role), Reason: request.Reason}
		if err := recordWorkspaceAuditTx(ctx, tx, actorInternalID, request.WorkspaceID, eventWorkspaceRoleChanged, details, request.OccurredAt); err != nil {
			return store.MembershipCommandResult{}, err
		}
		return store.MembershipCommandResult{MembershipID: membershipID, WorkspaceID: request.WorkspaceID, UserID: request.UserID, Role: string(role), Active: true, MembershipVersion: currentVersion + 1}, nil
	case "deactivate":
		var currentID, currentVersion int64
		var currentPublicID, currentRole string
		if err := tx.QueryRowContext(ctx, `SELECT m.id,m.public_id,m.membership_version,m.role FROM workspace_memberships m JOIN users u ON u.id=m.user_id AND u.active=1 WHERE m.workspace_id=? AND u.public_id=? AND m.active=1`, workspaceInternalID, request.UserID).Scan(&currentID, &currentPublicID, &currentVersion, &currentRole); errors.Is(err, sql.ErrNoRows) {
			return store.MembershipCommandResult{}, ErrWorkspaceMembershipNotFound
		} else if err != nil {
			return store.MembershipCommandResult{}, err
		}
		if request.ExpectedVersion != currentVersion {
			return store.MembershipCommandResult{}, store.ErrMembershipCommandConflict
		}
		if _, err := tx.ExecContext(ctx, `UPDATE workspace_memberships SET active=0,removed_by=?,removed_at=?,removal_reason=? WHERE id=? AND active=1`, actorInternalID, formatTime(request.OccurredAt), request.Reason, currentID); err != nil {
			return store.MembershipCommandResult{}, err
		}
		details := workspaceMutationDetails{WorkspaceID: request.WorkspaceID, MembershipID: currentPublicID, UserID: request.UserID, Role: currentRole, Reason: request.Reason}
		if err := recordWorkspaceAuditTx(ctx, tx, actorInternalID, request.WorkspaceID, eventWorkspaceMemberRemoved, details, request.OccurredAt); err != nil {
			return store.MembershipCommandResult{}, err
		}
		return store.MembershipCommandResult{MembershipID: currentPublicID, WorkspaceID: request.WorkspaceID, UserID: request.UserID, Role: currentRole, Active: false, MembershipVersion: currentVersion}, nil
	default:
		return store.MembershipCommandResult{}, store.ErrMembershipCommandUnsupported
	}
}

var _ store.MembershipCommandStore = (*Store)(nil)
