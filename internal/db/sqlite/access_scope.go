package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Onellan/tockrplatform/internal/domain"
	"github.com/Onellan/tockrplatform/internal/store"
)

var ErrAccessScopeDenied = store.ErrAccessScopeDenied

type scopeQueryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type workspaceScopeProof struct {
	userInternalID         int64
	organisationInternalID int64
	workspaceInternalID    int64
	scope                  store.WorkspaceScope
}

func (s *Store) ProveWorkspaceScope(ctx context.Context, requesterUserID, workspaceID string, adminOnly bool) (store.WorkspaceScope, error) {
	proof, err := proveWorkspaceScopeQuery(ctx, s.db, requesterUserID, workspaceID, adminOnly)
	if err != nil {
		return store.WorkspaceScope{}, err
	}
	return proof.scope, nil
}

func proveWorkspaceScopeQuery(ctx context.Context, query scopeQueryRower, requesterUserID, workspaceID string, adminOnly bool) (workspaceScopeProof, error) {
	var proof workspaceScopeProof
	var organisationRole, workspaceRole string
	var systemAdmin int
	err := query.QueryRowContext(ctx, `SELECT u.id,o.id,w.id,u.public_id,o.public_id,w.public_id,
		COALESCE(om.role,''),COALESCE(wm.role,''),
		EXISTS(SELECT 1 FROM system_role_assignments sra WHERE sra.user_id=u.id AND sra.role='system_admin' AND sra.active=1)
		FROM users u
		JOIN workspaces w ON w.public_id=? AND w.status='active'
		JOIN organisations o ON o.id=w.organisation_id AND o.status='active'
		LEFT JOIN organisation_memberships om ON om.organisation_id=o.id AND om.user_id=u.id AND om.active=1
		LEFT JOIN workspace_memberships wm ON wm.workspace_id=w.id AND wm.user_id=u.id AND wm.active=1
		WHERE u.public_id=? AND u.active=1`, workspaceID, requesterUserID).
		Scan(&proof.userInternalID, &proof.organisationInternalID, &proof.workspaceInternalID, &proof.scope.UserID, &proof.scope.OrganisationID, &proof.scope.WorkspaceID, &organisationRole, &workspaceRole, &systemAdmin)
	if errors.Is(err, sql.ErrNoRows) {
		return workspaceScopeProof{}, store.ErrAccessScopeDenied
	}
	if err != nil {
		return workspaceScopeProof{}, fmt.Errorf("prove workspace access scope: %w", err)
	}
	proof.scope.OrganisationRole = domain.OrganisationRole(organisationRole)
	proof.scope.WorkspaceRole = domain.WorkspaceRole(workspaceRole)
	proof.scope.SystemAdmin = systemAdmin == 1
	if !proof.scope.SystemAdmin && !proof.scope.OrganisationRole.Valid() {
		return workspaceScopeProof{}, store.ErrAccessScopeDenied
	}
	if adminOnly && !proof.scope.SystemAdmin && !proof.scope.OrganisationRole.CanAdminister() && !proof.scope.WorkspaceRole.CanAdminister() {
		return workspaceScopeProof{}, store.ErrAccessScopeDenied
	}
	if !adminOnly && !proof.scope.SystemAdmin && !proof.scope.OrganisationRole.CanAdminister() && !proof.scope.WorkspaceRole.Valid() {
		return workspaceScopeProof{}, store.ErrAccessScopeDenied
	}
	return proof, nil
}
