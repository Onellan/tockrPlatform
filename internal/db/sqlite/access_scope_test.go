package sqlite

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
	"github.com/Onellan/tockrplatform/internal/store"
)

func TestWorkspaceScopeGuardRequiresActiveParentAndHonoursAdminBoundary(t *testing.T) {
	ctx := context.Background()
	persistence, users := newOrganisationStore(t, 5)
	now := time.Now().UTC().Truncate(time.Second)
	organisationID, err := domain.NewOrganisationID()
	if err != nil {
		t.Fatal(err)
	}
	organisation, _, err := persistence.CreateOrganisation(ctx, users[0].ID, domain.Organisation{ID: organisationID, Name: "Scope Organisation", Status: domain.OrganisationActive}, "create scope organisation", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[1].ID, domain.OrganisationMember, "add scope member", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[2].ID, domain.OrganisationMember, "add scope outsider", now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	workspace, _, err := persistence.CreateWorkspace(ctx, users[0].ID, organisation.ID, domain.Workspace{Name: "Scoped Workspace"}, "create scoped workspace", now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.AddWorkspaceMember(ctx, users[0].ID, workspace.ID, users[1].ID, domain.WorkspaceMember, "grant scoped member", now.Add(4*time.Minute)); err != nil {
		t.Fatal(err)
	}

	ownerScope, err := persistence.ProveWorkspaceScope(ctx, users[0].ID, workspace.ID, false)
	if err != nil || ownerScope.OrganisationRole != domain.OrganisationOwner || ownerScope.WorkspaceRole != domain.WorkspaceAdmin {
		t.Fatalf("owner scope = %#v, err=%v", ownerScope, err)
	}
	if _, err := persistence.ProveWorkspaceScope(ctx, users[0].ID, workspace.ID, true); err != nil {
		t.Fatalf("owner admin scope = %v", err)
	}
	memberScope, err := persistence.ProveWorkspaceScope(ctx, users[1].ID, workspace.ID, false)
	if err != nil || memberScope.WorkspaceRole != domain.WorkspaceMember {
		t.Fatalf("member scope = %#v, err=%v", memberScope, err)
	}
	if _, err := persistence.ProveWorkspaceScope(ctx, users[1].ID, workspace.ID, true); !errors.Is(err, store.ErrAccessScopeDenied) {
		t.Fatalf("member admin scope = %v, want denied", err)
	}
	if _, err := persistence.ProveWorkspaceScope(ctx, users[2].ID, workspace.ID, false); !errors.Is(err, store.ErrAccessScopeDenied) {
		t.Fatalf("organisation-only scope = %v, want denied", err)
	}
	if _, err := persistence.ProveWorkspaceScope(ctx, users[1].ID, "wsp_tampered", false); !errors.Is(err, store.ErrAccessScopeDenied) {
		t.Fatalf("tampered scope = %v, want denied", err)
	}

	if err := persistence.DeactivateOrganisationMember(ctx, users[0].ID, organisation.ID, users[1].ID, "revoke scope parent", now.Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.ProveWorkspaceScope(ctx, users[1].ID, workspace.ID, false); !errors.Is(err, store.ErrAccessScopeDenied) {
		t.Fatalf("revoked parent scope = %v, want denied", err)
	}
	if err := persistence.SetUserActive(ctx, users[4].ID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.ProveWorkspaceScope(ctx, users[4].ID, workspace.ID, false); !errors.Is(err, store.ErrAccessScopeDenied) {
		t.Fatalf("inactive user scope = %v, want denied", err)
	}
	if err := persistence.SetUserActive(ctx, users[4].ID, true); err != nil {
		t.Fatal(err)
	}
	if err := seedSystemAdministrator(ctx, persistence, users[4].ID); err != nil {
		t.Fatal(err)
	}
	systemScope, err := persistence.ProveWorkspaceScope(ctx, users[4].ID, workspace.ID, true)
	if err != nil || !systemScope.SystemAdmin {
		t.Fatalf("system scope = %#v, err=%v", systemScope, err)
	}

	if err := persistence.ArchiveWorkspace(ctx, users[0].ID, workspace.ID, "archive scoped workspace", now.Add(6*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.ProveWorkspaceScope(ctx, users[0].ID, workspace.ID, false); !errors.Is(err, store.ErrAccessScopeDenied) {
		t.Fatalf("archived scope = %v, want denied", err)
	}
}

func TestWorkspaceScopeGuardConcurrentRevocationFailsClosedAfterCommit(t *testing.T) {
	ctx := context.Background()
	persistence, users := newOrganisationStore(t, 2)
	now := time.Now().UTC().Truncate(time.Second)
	organisationID, err := domain.NewOrganisationID()
	if err != nil {
		t.Fatal(err)
	}
	organisation, _, err := persistence.CreateOrganisation(ctx, users[0].ID, domain.Organisation{ID: organisationID, Name: "Concurrent Scope Organisation", Status: domain.OrganisationActive}, "create concurrent scope organisation", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[1].ID, domain.OrganisationMember, "add concurrent scope member", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	workspace, _, err := persistence.CreateWorkspace(ctx, users[0].ID, organisation.ID, domain.Workspace{Name: "Concurrent Scope Workspace"}, "create concurrent scope workspace", now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.AddWorkspaceMember(ctx, users[0].ID, workspace.ID, users[1].ID, domain.WorkspaceMember, "grant concurrent scope access", now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}

	var group sync.WaitGroup
	group.Add(1)
	go func() {
		defer group.Done()
		if err := persistence.DeactivateOrganisationMember(ctx, users[0].ID, organisation.ID, users[1].ID, "concurrent scope revocation", now.Add(4*time.Minute)); err != nil {
			t.Errorf("concurrent scope revocation = %v", err)
		}
	}()
	for i := 0; i < 25; i++ {
		if _, err := persistence.ProveWorkspaceScope(ctx, users[1].ID, workspace.ID, false); err != nil && !errors.Is(err, store.ErrAccessScopeDenied) {
			t.Fatalf("concurrent scope proof = %v", err)
		}
	}
	group.Wait()
	if _, err := persistence.ProveWorkspaceScope(ctx, users[1].ID, workspace.ID, false); !errors.Is(err, store.ErrAccessScopeDenied) {
		t.Fatalf("scope after concurrent revocation = %v, want denied", err)
	}
}
