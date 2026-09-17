package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
	_ "modernc.org/sqlite"
)

func TestWorkspaceLifecycleIsOrganisationOwnedHistoricalAndFailClosed(t *testing.T) {
	ctx := context.Background()
	store, users := newOrganisationStore(t, 5)
	now := time.Date(2026, 9, 16, 9, 0, 0, 0, time.UTC)
	organisationID, err := domain.NewOrganisationID()
	if err != nil {
		t.Fatal(err)
	}
	organisation, _, err := store.CreateOrganisation(ctx, users[0].ID, domain.Organisation{ID: organisationID, Name: "Workspace Organisation", Status: domain.OrganisationActive}, "create workspace organisation", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[1].ID, domain.OrganisationMember, "add workspace member", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[2].ID, domain.OrganisationMember, "add second workspace member", now.Add(90*time.Second)); err != nil {
		t.Fatal(err)
	}
	workspaceID, err := domain.NewWorkspaceID()
	if err != nil {
		t.Fatal(err)
	}
	workspace, adminMembership, err := store.CreateWorkspace(ctx, users[0].ID, organisation.ID, domain.Workspace{ID: workspaceID, Name: "Operations", Status: domain.WorkspaceActive}, "create operations workspace", now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if workspace.ID != workspaceID || workspace.OrganisationID != organisation.ID || adminMembership.Role != domain.WorkspaceAdmin || !adminMembership.Active {
		t.Fatalf("created workspace = %#v, admin membership = %#v", workspace, adminMembership)
	}

	if _, err := store.GetWorkspace(ctx, users[1].ID, workspace.ID); !errors.Is(err, ErrUnauthorisedWorkspaceAction) {
		t.Fatalf("organisation-only user workspace read = %v, want unauthorised", err)
	}
	memberMembership, err := store.AddWorkspaceMember(ctx, users[0].ID, workspace.ID, users[1].ID, domain.WorkspaceMember, "grant workspace member access", now.Add(3*time.Minute))
	if err != nil || memberMembership.Role != domain.WorkspaceMember {
		t.Fatalf("add workspace member = %#v, err=%v", memberMembership, err)
	}
	read, err := store.GetWorkspace(ctx, users[1].ID, workspace.ID)
	if err != nil || read.Name != "Operations" {
		t.Fatalf("workspace member read = %#v, err=%v", read, err)
	}
	if _, err := store.ListWorkspaceMembers(ctx, users[1].ID, workspace.ID); !errors.Is(err, ErrUnauthorisedWorkspaceAction) {
		t.Fatalf("workspace member enumeration = %v, want unauthorised", err)
	}

	workspaceAdmin, err := store.ChangeWorkspaceMemberRole(ctx, users[0].ID, workspace.ID, users[1].ID, domain.WorkspaceAdmin, "grant workspace administration", now.Add(4*time.Minute))
	if err != nil || workspaceAdmin.Role != domain.WorkspaceAdmin {
		t.Fatalf("promote workspace administrator = %#v, err=%v", workspaceAdmin, err)
	}
	viewerMembership, err := store.AddWorkspaceMember(ctx, users[1].ID, workspace.ID, users[2].ID, domain.WorkspaceViewer, "grant viewer access", now.Add(5*time.Minute))
	if err != nil || viewerMembership.Role != domain.WorkspaceViewer {
		t.Fatalf("workspace admin add viewer = %#v, err=%v", viewerMembership, err)
	}
	changed, err := store.ChangeWorkspaceMemberRole(ctx, users[0].ID, workspace.ID, users[2].ID, domain.WorkspaceMember, "grant member access", now.Add(6*time.Minute))
	if err != nil || changed.Role != domain.WorkspaceMember || changed.ID == viewerMembership.ID {
		t.Fatalf("workspace role change = %#v, old=%#v, err=%v", changed, viewerMembership, err)
	}
	members, err := store.ListWorkspaceMembers(ctx, users[0].ID, workspace.ID)
	if err != nil || len(members) != 3 {
		t.Fatalf("workspace member list = %#v, err=%v", members, err)
	}
	if _, err := store.GetWorkspaceMembership(ctx, users[2].ID, workspace.ID, users[2].ID); err != nil {
		t.Fatalf("workspace self membership read = %v", err)
	}
	if _, err := store.GetWorkspaceMembership(ctx, users[2].ID, workspace.ID, users[1].ID); !errors.Is(err, ErrUnauthorisedWorkspaceAction) {
		t.Fatalf("workspace non-admin membership read = %v, want unauthorised", err)
	}
	visible, err := store.ListOrganisationWorkspaces(ctx, users[1].ID, organisation.ID)
	if err != nil || len(visible) != 1 || visible[0].ID != workspace.ID {
		t.Fatalf("member workspace list = %#v, err=%v", visible, err)
	}
	if err := store.DeactivateOrganisationMember(ctx, users[0].ID, organisation.ID, users[1].ID, "revoke parent organisation access", now.Add(6*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetWorkspace(ctx, users[1].ID, workspace.ID); !errors.Is(err, ErrUnauthorisedWorkspaceAction) {
		t.Fatalf("stale workspace membership after organisation removal = %v, want unauthorised", err)
	}

	otherID, err := domain.NewOrganisationID()
	if err != nil {
		t.Fatal(err)
	}
	other, _, err := store.CreateOrganisation(ctx, users[3].ID, domain.Organisation{ID: otherID, Name: "Other Organisation", Status: domain.OrganisationActive}, "create other organisation", now)
	if err != nil {
		t.Fatal(err)
	}
	otherWorkspace, _, err := store.CreateWorkspace(ctx, users[3].ID, other.ID, domain.Workspace{Name: "Other Workspace"}, "create other workspace", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetWorkspace(ctx, users[0].ID, otherWorkspace.ID); !errors.Is(err, ErrUnauthorisedWorkspaceAction) {
		t.Fatalf("cross-organisation workspace read = %v, want unauthorised", err)
	}
	if _, err := store.AddWorkspaceMember(ctx, users[0].ID, workspace.ID, users[3].ID, domain.WorkspaceViewer, "cross-organisation target", now.Add(7*time.Minute)); !errors.Is(err, ErrUnauthorisedWorkspaceAction) {
		t.Fatalf("cross-organisation membership assignment = %v, want unauthorised", err)
	}
	if err := store.SetUserActive(ctx, users[4].ID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddWorkspaceMember(ctx, users[0].ID, workspace.ID, users[4].ID, domain.WorkspaceViewer, "inactive target", now.Add(7*time.Minute)); !errors.Is(err, ErrUnauthorisedWorkspaceAction) {
		t.Fatalf("inactive workspace target = %v, want unauthorised", err)
	}

	if _, err := store.ListOrganisationWorkspaces(ctx, users[1].ID, organisation.ID); !errors.Is(err, ErrUnauthorisedOrganisationAction) {
		t.Fatalf("workspace list after organisation removal = %v, want unauthorised", err)
	}
	if err := store.ArchiveWorkspace(ctx, users[1].ID, workspace.ID, "member cannot archive", now.Add(7*time.Minute)); !errors.Is(err, ErrUnauthorisedWorkspaceAction) {
		t.Fatalf("member archive = %v, want unauthorised", err)
	}
	if err := store.ArchiveWorkspace(ctx, users[0].ID, workspace.ID, "retire workspace", now.Add(8*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetWorkspace(ctx, users[0].ID, workspace.ID); !errors.Is(err, ErrWorkspaceArchived) {
		t.Fatalf("archived workspace read = %v, want archived", err)
	}
	if workspaces, err := store.ListOrganisationWorkspaces(ctx, users[0].ID, organisation.ID); err != nil || len(workspaces) != 0 {
		t.Fatalf("archived workspace listing = %#v, err=%v", workspaces, err)
	}

	var historicalRoles []string
	rows, err := store.DB().QueryContext(ctx, `SELECT role FROM workspace_memberships WHERE workspace_id=(SELECT id FROM workspaces WHERE public_id=?) AND user_id=(SELECT id FROM users WHERE public_id=?) ORDER BY id`, workspace.ID, users[2].ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			t.Fatal(err)
		}
		historicalRoles = append(historicalRoles, role)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(historicalRoles) != 2 || historicalRoles[0] != string(domain.WorkspaceViewer) || historicalRoles[1] != string(domain.WorkspaceMember) {
		t.Fatalf("workspace role history = %v", historicalRoles)
	}
	for _, event := range []string{eventWorkspaceCreated, eventWorkspaceMembershipAdded, eventWorkspaceRoleChanged, eventWorkspaceMemberRemoved, eventWorkspaceArchived} {
		count, err := store.workspaceAuditCount(ctx, workspace.ID, event)
		if err != nil || count == 0 {
			t.Fatalf("workspace audit event %q count = %d, err=%v", event, count, err)
		}
	}
}

func TestWorkspaceFreshUpgradeReopenAndDivergenceMigration(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "workspace.db")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	var version int
	if err := store.DB().QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 10 {
		t.Fatalf("fresh schema version = %d, want 10", version)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}

	legacyPath := filepath.Join(t.TempDir(), "legacy.db")
	legacy, err := sql.Open("sqlite", legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.ExecContext(ctx, `CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, checksum TEXT NOT NULL, applied_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	for _, migration := range supportedMigrations()[:4] {
		for _, statement := range migration.statements {
			if _, err := legacy.ExecContext(ctx, statement); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := legacy.ExecContext(ctx, `INSERT INTO schema_migrations(version,name,checksum,applied_at) VALUES(?,?,?,?)`, migration.version, migration.name, migrationChecksum(migration), formatTime(time.Now().UTC())); err != nil {
			t.Fatal(err)
		}
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	upgraded, err := Open(ctx, legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := upgraded.DB().QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 10 {
		t.Fatalf("upgraded schema version = %d, want 10", version)
	}
	if _, err := upgraded.DB().ExecContext(ctx, `UPDATE schema_migrations SET name='changed' WHERE version=5`); err != nil {
		t.Fatal(err)
	}
	if err := upgraded.Close(); err != nil {
		t.Fatal(err)
	}
	if reopened, err := Open(ctx, legacyPath); err == nil {
		_ = reopened.Close()
		t.Fatal("divergent workspace migration ledger was accepted")
	}
}

func TestWorkspaceConcurrentMembershipAssignmentPreservesOneActiveRow(t *testing.T) {
	ctx := context.Background()
	store, users := newOrganisationStore(t, 3)
	now := time.Now().UTC().Truncate(time.Second)
	organisationID, err := domain.NewOrganisationID()
	if err != nil {
		t.Fatal(err)
	}
	organisation, _, err := store.CreateOrganisation(ctx, users[0].ID, domain.Organisation{ID: organisationID, Name: "Concurrent Workspace Organisation", Status: domain.OrganisationActive}, "create concurrent organisation", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[1].ID, domain.OrganisationMember, "add concurrent target", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	workspace, _, err := store.CreateWorkspace(ctx, users[0].ID, organisation.ID, domain.Workspace{Name: "Concurrent Workspace"}, "create concurrent workspace", now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	results := make(chan error, 2)
	var group sync.WaitGroup
	group.Add(2)
	for _, role := range []domain.WorkspaceRole{domain.WorkspaceMember, domain.WorkspaceViewer} {
		go func(role domain.WorkspaceRole) {
			defer group.Done()
			_, callErr := store.AddWorkspaceMember(ctx, users[0].ID, workspace.ID, users[1].ID, role, "concurrent assignment", now.Add(3*time.Minute))
			results <- callErr
		}(role)
	}
	group.Wait()
	close(results)
	var successes, duplicates int
	for callErr := range results {
		switch {
		case callErr == nil:
			successes++
		case errors.Is(callErr, ErrDuplicateWorkspaceMember):
			duplicates++
		default:
			t.Fatalf("concurrent workspace assignment error = %v", callErr)
		}
	}
	if successes != 1 || duplicates != 1 {
		t.Fatalf("concurrent assignment outcomes = successes:%d duplicates:%d, want one each", successes, duplicates)
	}
	var active int
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM workspace_memberships WHERE workspace_id=(SELECT id FROM workspaces WHERE public_id=?) AND user_id=(SELECT id FROM users WHERE public_id=?) AND active=1`, workspace.ID, users[1].ID).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if active != 1 {
		t.Fatalf("active concurrent workspace memberships = %d, want 1", active)
	}
}
