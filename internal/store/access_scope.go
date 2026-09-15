package store

import (
	"context"
	"errors"

	"github.com/Onellan/tockrplatform/internal/domain"
)

var ErrAccessScopeDenied = errors.New("access scope is not authorised")

// WorkspaceScope is the minimum Platform proof needed by a protected
// Workspace caller. Empty roles are intentional for system administrators and
// Organisation administrators who do not need a Workspace membership row to
// administer an Organisation-owned Workspace.
type WorkspaceScope struct {
	UserID           string
	OrganisationID   string
	WorkspaceID      string
	OrganisationRole domain.OrganisationRole
	WorkspaceRole    domain.WorkspaceRole
	SystemAdmin      bool
}

type AccessScopeStore interface {
	ProveWorkspaceScope(context.Context, string, string, bool) (WorkspaceScope, error)
}
