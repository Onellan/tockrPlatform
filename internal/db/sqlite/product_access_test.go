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

func TestUserProductAssignmentAndEffectiveAccessRequireEveryPredicate(t *testing.T) {
	ctx := context.Background()
	persistence, users := newOrganisationStore(t, 5)
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	organisationID, err := domain.NewOrganisationID()
	if err != nil {
		t.Fatal(err)
	}
	organisation, _, err := persistence.CreateOrganisation(ctx, users[0].ID, domain.Organisation{ID: organisationID, Name: "Access Organisation"}, "create access organisation", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[1].ID, domain.OrganisationAdmin, "appoint access admin", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[2].ID, domain.OrganisationMember, "add access member", now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	workspace, _, err := persistence.CreateWorkspace(ctx, users[0].ID, organisation.ID, domain.Workspace{ID: "wsp_access", Name: "Access Workspace"}, "create access workspace", now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.AddWorkspaceMember(ctx, users[0].ID, workspace.ID, users[2].ID, domain.WorkspaceMember, "grant access workspace", now.Add(4*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.EntitleOrganisation(ctx, users[0].ID, organisation.ID, "product.tockrctrl", "enable CTRL", now.Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}

	if _, err := persistence.ProveProductAccess(ctx, users[0].ID, organisation.ID, "product.tockrctrl", workspace.ID); !errors.Is(err, store.ErrProductAccessDenied) {
		t.Fatalf("owner without assignment access = %v, want denied", err)
	}
	assignment, err := persistence.AssignUserProduct(ctx, users[1].ID, organisation.ID, users[2].ID, "product.tockrctrl", "assign CTRL member", now.Add(6*time.Minute))
	if err != nil || !assignment.Active {
		t.Fatalf("effective assignment = %#v, err=%v", assignment, err)
	}
	access, err := persistence.ProveProductAccess(ctx, users[2].ID, organisation.ID, "product.tockrctrl", workspace.ID)
	if err != nil || access.UserID != users[2].ID || access.WorkspaceID != workspace.ID || access.OrganisationRole != domain.OrganisationMember || access.WorkspaceRole != domain.WorkspaceMember {
		t.Fatalf("all-predicate access = %#v, err=%v", access, err)
	}
	if _, err := persistence.ProveProductAccess(ctx, users[2].ID, organisation.ID, "product.tockrctrl", "wsp_wrong"); !errors.Is(err, store.ErrProductAccessDenied) {
		t.Fatalf("workspace mismatch access = %v, want denied", err)
	}
	if _, err := persistence.AssignUserProduct(ctx, users[3].ID, organisation.ID, users[3].ID, "product.tockrctrl", "outsider assignment", now.Add(7*time.Minute)); !errors.Is(err, ErrUnauthorisedOrganisationAction) {
		t.Fatalf("outsider assignment = %v, want unauthorised", err)
	}
	if _, err := persistence.AssignUserProduct(ctx, users[1].ID, organisation.ID, users[3].ID, "product.tockrctrl", "non-member assignment", now.Add(8*time.Minute)); !errors.Is(err, ErrUnauthorisedOrganisationAction) {
		t.Fatalf("non-member assignment = %v, want unauthorised", err)
	}

	assignmentOnly, err := persistence.AssignUserProduct(ctx, users[1].ID, organisation.ID, users[2].ID, "product.tockrims", "stage IMS assignment", now.Add(9*time.Minute))
	if err != nil || !assignmentOnly.Active {
		t.Fatalf("assignment without entitlement = %#v, err=%v", assignmentOnly, err)
	}
	if _, err := persistence.ProveProductAccess(ctx, users[2].ID, organisation.ID, "product.tockrims", workspace.ID); !errors.Is(err, store.ErrProductAccessDenied) {
		t.Fatalf("assignment-only access = %v, want denied", err)
	}
	assignments, err := persistence.ListUserProductAssignments(ctx, users[0].ID, organisation.ID)
	if err != nil || len(assignments) != 2 || !assignments[0].Active || !assignments[1].Active {
		t.Fatalf("assignment read model = %#v, err=%v", assignments, err)
	}
	if err := persistence.RevokeUserProduct(ctx, users[1].ID, organisation.ID, assignment.ID, "revoke CTRL assignment", now.Add(10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.ProveProductAccess(ctx, users[2].ID, organisation.ID, "product.tockrctrl", workspace.ID); !errors.Is(err, store.ErrProductAccessDenied) {
		t.Fatalf("revoked assignment access = %v, want denied", err)
	}
	if err := persistence.RevokeUserProduct(ctx, users[1].ID, organisation.ID, assignment.ID, "repeat revoke", now.Add(11*time.Minute)); !errors.Is(err, ErrProductAssignmentInactive) {
		t.Fatalf("repeat assignment revoke = %v, want inactive", err)
	}

	secondAssignment, err := persistence.AssignUserProduct(ctx, users[0].ID, organisation.ID, users[2].ID, "product.tockrctrl", "restore CTRL assignment", now.Add(12*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if err := persistence.RevokeOrganisationEntitlement(ctx, users[0].ID, organisation.ID, mustCurrentEntitlementID(t, persistence, organisation.ID, "product.tockrctrl"), "revoke CTRL entitlement", now.Add(13*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.ProveProductAccess(ctx, users[2].ID, organisation.ID, "product.tockrctrl", workspace.ID); !errors.Is(err, store.ErrProductAccessDenied) {
		t.Fatalf("revoked entitlement access = %v, want denied", err)
	}
	if err := persistence.DeactivateOrganisationMember(ctx, users[0].ID, organisation.ID, users[2].ID, "revoke access membership", now.Add(14*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.ProveProductAccess(ctx, users[2].ID, organisation.ID, "product.tockrctrl", workspace.ID); !errors.Is(err, store.ErrProductAccessDenied) {
		t.Fatalf("revoked membership access = %v, want denied", err)
	}
	if secondAssignment.ID == assignment.ID {
		t.Fatal("assignment regrant did not preserve history")
	}
	for _, event := range []string{eventAssignmentGranted, eventAssignmentRevoked} {
		count, err := persistence.organisationAuditCount(ctx, organisation.ID, event)
		if err != nil || count == 0 {
			t.Fatalf("assignment audit %q count = %d, err=%v", event, count, err)
		}
	}

	if err := seedSystemAdministrator(ctx, persistence, users[4].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.ProveProductAccess(ctx, users[4].ID, organisation.ID, "product.tockrctrl", workspace.ID); !errors.Is(err, store.ErrProductAccessDenied) {
		t.Fatalf("system administrator without membership access = %v, want denied", err)
	}
}

func TestUserProductAssignmentDuplicateIsSerialized(t *testing.T) {
	ctx := context.Background()
	persistence, users := newOrganisationStore(t, 4)
	now := time.Now().UTC().Truncate(time.Second)
	organisationID, err := domain.NewOrganisationID()
	if err != nil {
		t.Fatal(err)
	}
	organisation, _, err := persistence.CreateOrganisation(ctx, users[0].ID, domain.Organisation{ID: organisationID, Name: "Concurrent Assignment Organisation"}, "create concurrent assignment organisation", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[1].ID, domain.OrganisationAdmin, "appoint concurrent assignment admin", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.EntitleOrganisation(ctx, users[0].ID, organisation.ID, "product.tockrctrl", "enable concurrent CTRL", now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	group.Add(2)
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			defer group.Done()
			_, callErr := persistence.AssignUserProduct(ctx, users[1].ID, organisation.ID, users[1].ID, "product.tockrctrl", "concurrent assignment", now.Add(3*time.Minute))
			results <- callErr
		}()
	}
	group.Wait()
	close(results)
	successes, duplicates := 0, 0
	for callErr := range results {
		if callErr == nil {
			successes++
		} else if errors.Is(callErr, ErrDuplicateUserProductAssignment) {
			duplicates++
		} else {
			t.Fatalf("concurrent assignment error = %v", callErr)
		}
	}
	if successes != 1 || duplicates != 1 {
		t.Fatalf("concurrent assignment outcomes = successes:%d duplicates:%d, want one each", successes, duplicates)
	}
}

func mustCurrentEntitlementID(t *testing.T, persistence *Store, organisationID, productKey string) string {
	t.Helper()
	entitlements, err := persistence.ListOrganisationProductEntitlements(context.Background(), findOrganisationOwner(t, persistence, organisationID), organisationID)
	if err != nil {
		t.Fatal(err)
	}
	for _, entitlement := range entitlements {
		if entitlement.ProductKey == productKey && entitlement.Active {
			return entitlement.ID
		}
	}
	t.Fatalf("active entitlement %q not found", productKey)
	return ""
}

func findOrganisationOwner(t *testing.T, persistence *Store, organisationID string) string {
	t.Helper()
	var userID string
	if err := persistence.DB().QueryRowContext(context.Background(), `SELECT u.public_id FROM users u JOIN organisation_memberships m ON m.user_id=u.id JOIN organisations o ON o.id=m.organisation_id WHERE o.public_id=? AND m.role='owner' AND m.active=1`, organisationID).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	return userID
}
