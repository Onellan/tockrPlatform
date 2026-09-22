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

const membershipCommandResultRetention = 24 * time.Hour

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

func (s *Store) CleanupMembershipCommandResults(ctx context.Context, before time.Time, limit int) (int64, error) {
	minimumAge := time.Now().UTC().Add(-membershipCommandResultRetention)
	if before.IsZero() || before.After(minimumAge) || limit < 1 || limit > 1000 {
		return 0, store.ErrInvalidReadAuthorityRequest
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM platform_membership_command_results
		WHERE id IN (SELECT id FROM platform_membership_command_results
		WHERE created_at<=? ORDER BY created_at,id LIMIT ?)`, formatTime(before.UTC()), limit)
	if err != nil {
		return 0, fmt.Errorf("cleanup membership command results: %w", err)
	}
	removed, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count cleaned membership command results: %w", err)
	}
	return removed, nil
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
	role := domain.OrganisationRole(request.Role)
	var membership domain.OrganisationMembership
	var err error
	version := request.ExpectedVersion
	switch request.Operation {
	case "add":
		membership, err = s.addOrganisationMemberTx(ctx, tx, request.ActorUserID, request.OrganisationID, request.UserID, role, request.Reason, request.OccurredAt)
		version = 1
	case "change_role":
		membership, err = s.changeOrganisationMemberRoleTx(ctx, tx, request.ActorUserID, request.OrganisationID, request.UserID, role, request.Reason, request.OccurredAt, &request.ExpectedVersion)
		version = request.ExpectedVersion + 1
	case "deactivate":
		membership, err = s.deactivateOrganisationMemberTx(ctx, tx, request.ActorUserID, request.OrganisationID, request.UserID, request.Reason, request.OccurredAt, &request.ExpectedVersion)
	default:
		return store.MembershipCommandResult{}, store.ErrMembershipCommandUnsupported
	}
	if err != nil {
		return store.MembershipCommandResult{}, err
	}
	return store.MembershipCommandResult{MembershipID: membership.ID, OrganisationID: membership.OrganisationID, UserID: membership.UserID, Role: string(membership.Role), Active: membership.Active, MembershipVersion: version}, nil
}

func (s *Store) executeWorkspaceMembershipCommandTx(ctx context.Context, tx *sql.Tx, request store.MembershipCommandRequest) (store.MembershipCommandResult, error) {
	role := domain.WorkspaceRole(request.Role)
	var membership domain.WorkspaceMembership
	var err error
	version := request.ExpectedVersion
	switch request.Operation {
	case "add":
		membership, err = s.addWorkspaceMemberTx(ctx, tx, request.ActorUserID, request.WorkspaceID, request.UserID, role, request.Reason, request.OccurredAt)
		version = 1
	case "change_role":
		membership, err = s.changeWorkspaceMemberRoleTx(ctx, tx, request.ActorUserID, request.WorkspaceID, request.UserID, role, request.Reason, request.OccurredAt, &request.ExpectedVersion)
		version = request.ExpectedVersion + 1
	case "deactivate":
		membership, err = s.deactivateWorkspaceMemberTx(ctx, tx, request.ActorUserID, request.WorkspaceID, request.UserID, request.Reason, request.OccurredAt, &request.ExpectedVersion)
	default:
		return store.MembershipCommandResult{}, store.ErrMembershipCommandUnsupported
	}
	if err != nil {
		return store.MembershipCommandResult{}, err
	}
	return store.MembershipCommandResult{MembershipID: membership.ID, WorkspaceID: membership.WorkspaceID, UserID: membership.UserID, Role: string(membership.Role), Active: membership.Active, MembershipVersion: version}, nil
}

var _ store.MembershipCommandStore = (*Store)(nil)
