package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
)

func TestOrganisationLifecycleIsTransactionalAuditableAndHistoryPreserving(t *testing.T) {
	ctx := context.Background()
	store, users := newOrganisationStore(t, 4)
	now := time.Date(2026, 9, 16, 8, 0, 0, 0, time.UTC)
	organisationID, err := domain.NewOrganisationID()
	if err != nil {
		t.Fatal(err)
	}
	organisation, ownerMembership, err := store.CreateOrganisation(ctx, users[0].ID, domain.Organisation{ID: organisationID, Name: "North Star", Status: domain.OrganisationActive}, "initial owner", now)
	if err != nil {
		t.Fatal(err)
	}
	if organisation.Status != domain.OrganisationActive || ownerMembership.Role != domain.OrganisationOwner || !ownerMembership.Active {
		t.Fatalf("created organisation = %#v, membership = %#v", organisation, ownerMembership)
	}

	adminMembership, err := store.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[1].ID, domain.OrganisationAdmin, "appoint administration", now.Add(time.Minute))
	if err != nil || adminMembership.Role != domain.OrganisationAdmin {
		t.Fatalf("add admin = %#v, err=%v", adminMembership, err)
	}
	memberMembership, err := store.AddOrganisationMember(ctx, users[1].ID, organisation.ID, users[2].ID, domain.OrganisationMember, "invite member", now.Add(2*time.Minute))
	if err != nil || memberMembership.Role != domain.OrganisationMember {
		t.Fatalf("add member = %#v, err=%v", memberMembership, err)
	}
	if _, err := store.AddOrganisationMember(ctx, users[1].ID, organisation.ID, users[3].ID, domain.OrganisationAdmin, "unauthorised promotion", now.Add(3*time.Minute)); !errors.Is(err, ErrUnauthorisedOrganisationAction) {
		t.Fatalf("admin promotion error = %v, want unauthorised", err)
	}

	promoted, err := store.ChangeOrganisationMemberRole(ctx, users[0].ID, organisation.ID, users[2].ID, domain.OrganisationAdmin, "promote member", now.Add(4*time.Minute))
	if err != nil || promoted.Role != domain.OrganisationAdmin || promoted.ID == memberMembership.ID {
		t.Fatalf("promoted membership = %#v, old=%#v, err=%v", promoted, memberMembership, err)
	}
	if _, err := store.GetOrganisationMembership(ctx, users[2].ID, organisation.ID, users[2].ID); err != nil {
		t.Fatalf("promoted member read = %v", err)
	}
	if err := store.DeactivateOrganisationMember(ctx, users[0].ID, organisation.ID, users[1].ID, "remove administrator", now.Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetOrganisationMembership(ctx, users[1].ID, organisation.ID, users[1].ID); !errors.Is(err, ErrMembershipNotFound) {
		t.Fatalf("inactive membership read = %v, want not found", err)
	}
	if _, err := store.GetOrganisation(ctx, users[1].ID, organisation.ID); !errors.Is(err, ErrUnauthorisedOrganisationAction) {
		t.Fatalf("inactive organisation read = %v, want unauthorised", err)
	}

	var historicalRoles []string
	rows, err := store.DB().QueryContext(ctx, `SELECT role FROM organisation_memberships WHERE organisation_id=(SELECT id FROM organisations WHERE public_id=?) AND user_id=(SELECT id FROM users WHERE public_id=?) ORDER BY id`, organisation.ID, users[2].ID)
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
	if len(historicalRoles) != 2 || historicalRoles[0] != string(domain.OrganisationMember) || historicalRoles[1] != string(domain.OrganisationAdmin) {
		t.Fatalf("promoted member history = %v", historicalRoles)
	}
	var retiredRole string
	if err := store.DB().QueryRowContext(ctx, `SELECT role FROM organisation_memberships WHERE public_id=?`, memberMembership.ID).Scan(&retiredRole); err != nil {
		t.Fatal(err)
	}
	if retiredRole != string(domain.OrganisationMember) {
		t.Fatalf("retired membership role = %q", retiredRole)
	}

	if err := store.ArchiveOrganisation(ctx, users[0].ID, organisation.ID, "close organisation", now.Add(6*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetOrganisation(ctx, users[0].ID, organisation.ID); !errors.Is(err, ErrUnauthorisedOrganisationAction) {
		t.Fatalf("archived organisation read = %v, want unauthorised", err)
	}
	for _, event := range []string{"organisation_created", "membership_added", "membership_role_changed", "membership_deactivated", "organisation_archived"} {
		count, err := store.OrganisationAuditCount(ctx, organisation.ID, event)
		if err != nil || count == 0 {
			t.Fatalf("audit event %q count = %d, err=%v", event, count, err)
		}
	}
}

func TestOrganisationScopeAndOwnerProtectionFailClosed(t *testing.T) {
	ctx := context.Background()
	store, users := newOrganisationStore(t, 3)
	now := time.Now().UTC().Truncate(time.Second)
	firstID, err := domain.NewOrganisationID()
	if err != nil {
		t.Fatal(err)
	}
	first, _, err := store.CreateOrganisation(ctx, users[0].ID, domain.Organisation{ID: firstID, Name: "First", Status: domain.OrganisationActive}, "create first", now)
	if err != nil {
		t.Fatal(err)
	}
	secondID, err := domain.NewOrganisationID()
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := store.CreateOrganisation(ctx, users[1].ID, domain.Organisation{ID: secondID, Name: "Second", Status: domain.OrganisationActive}, "create second", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetOrganisation(ctx, users[0].ID, second.ID); !errors.Is(err, ErrUnauthorisedOrganisationAction) {
		t.Fatalf("cross-organisation read = %v, want unauthorised", err)
	}
	if _, err := store.AddOrganisationMember(ctx, users[0].ID, second.ID, users[2].ID, domain.OrganisationMember, "cross-organisation write", now); !errors.Is(err, ErrUnauthorisedOrganisationAction) {
		t.Fatalf("cross-organisation write = %v, want unauthorised", err)
	}
	if err := store.DeactivateOrganisationMember(ctx, users[0].ID, first.ID, users[0].ID, "remove owner", now); !errors.Is(err, ErrOwnerMutationNotAuthorised) {
		t.Fatalf("owner deactivation = %v, want protected", err)
	}
	if _, err := store.ChangeOrganisationMemberRole(ctx, users[0].ID, first.ID, users[0].ID, domain.OrganisationMember, "demote owner", now); !errors.Is(err, ErrOwnerMutationNotAuthorised) {
		t.Fatalf("owner role change = %v, want protected", err)
	}
}

func TestOrganisationFreshUpgradeReopenAndDivergenceMigration(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "organisation.db")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	var version int
	if err := store.DB().QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 3 {
		t.Fatalf("fresh schema version = %d, want 3", version)
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
	for _, statement := range supportedMigrations()[0].statements {
		if _, err := legacy.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := legacy.ExecContext(ctx, `INSERT INTO schema_migrations(version,name,checksum,applied_at) VALUES(1,?,?,?)`, supportedMigrations()[0].name, migrationChecksum(supportedMigrations()[0]), formatTime(time.Now().UTC())); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}
	upgraded, err := Open(ctx, legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Close()
	if err := upgraded.DB().QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 3 {
		t.Fatalf("upgraded schema version = %d, want 3", version)
	}
	if _, err := upgraded.DB().ExecContext(ctx, `UPDATE schema_migrations SET name='changed' WHERE version=3`); err != nil {
		t.Fatal(err)
	}
	if err := upgraded.Close(); err != nil {
		t.Fatal(err)
	}
	if reopened, err := Open(ctx, legacyPath); err == nil {
		_ = reopened.Close()
		t.Fatal("divergent organisation migration ledger was accepted")
	}
}

func newOrganisationStore(t *testing.T, count int) (*Store, []domain.User) {
	t.Helper()
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "platform.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	users := make([]domain.User, 0, count)
	for i := 0; i < count; i++ {
		id, err := domain.NewUserID()
		if err != nil {
			t.Fatal(err)
		}
		user, err := store.CreateUser(ctx, domain.User{ID: id, Email: "org-user-" + string(rune('a'+i)) + "@example.test", DisplayName: "Organisation User", Active: true}, "correct horse battery staple")
		if err != nil {
			t.Fatal(err)
		}
		users = append(users, user)
	}
	return store, users
}
