package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Onellan/tockrplatform/internal/auth"
	"github.com/Onellan/tockrplatform/internal/domain"
)

func TestUserLifecycleHasStableOpaqueIdentityAndReopens(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "platform.db")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	userID, err := domain.NewUserID()
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.CreateUser(ctx, domain.User{ID: userID, Email: "Owner@Example.test", DisplayName: "Platform Owner", Active: true}, "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != userID || created.Email != "owner@example.test" {
		t.Fatalf("created user = %#v", created)
	}
	if created.PasswordHash != "" {
		t.Fatal("credential hash leaked from create projection")
	}
	stored, err := store.FindUserByEmail(ctx, "OWNER@example.test")
	if err != nil || stored == nil {
		t.Fatalf("find user = %#v, err=%v", stored, err)
	}
	if stored.ID != userID || stored.PasswordHash == "" || stored.PasswordHash == "correct horse battery staple" || !auth.CheckPassword(stored.PasswordHash, "correct horse battery staple") {
		t.Fatalf("stored credential/user identity invalid: %#v", stored)
	}
	if err := store.SetUserActive(ctx, userID, false); err != nil {
		t.Fatal(err)
	}
	inactive, err := store.FindUserByID(ctx, userID)
	if err != nil || inactive == nil || inactive.Active {
		t.Fatalf("inactive lifecycle = %#v, err=%v", inactive, err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	user, err := reopened.FindUserByID(ctx, userID)
	if err != nil || user == nil || user.ID != userID || user.Active {
		t.Fatalf("reopened user = %#v, err=%v", user, err)
	}
	var migrationVersion int
	if err := reopened.DB().QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&migrationVersion); err != nil {
		t.Fatal(err)
	}
	if migrationVersion != 2 {
		t.Fatalf("migration version = %d, want 2", migrationVersion)
	}
}

func TestStoreUsesOneSQLiteConnection(t *testing.T) {
	store, err := Open(context.Background(), filepath.Join(t.TempDir(), "single-connection.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	stats := store.DB().Stats()
	if stats.MaxOpenConnections != 1 {
		t.Fatalf("max open connections = %d, want 1", stats.MaxOpenConnections)
	}
	if stats.MaxIdleClosed != 0 {
		t.Fatalf("unexpected idle connection closures = %d", stats.MaxIdleClosed)
	}
}

func TestSecurityEventsDoNotRequireAUser(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.RecordSecurityEvent(ctx, nil, "failed_login", "credentials rejected", nowForTest()); err != nil {
		t.Fatal(err)
	}
	count, err := store.SecurityEventCount(ctx, "failed_login")
	if err != nil || count != 1 {
		t.Fatalf("failed login audit count = %d, err=%v", count, err)
	}
}

func TestMigrationLedgerRejectsRenamedMigration(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "divergent.db")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB().ExecContext(ctx, `UPDATE schema_migrations SET name='renamed-migration' WHERE version=1`); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if reopened, err := Open(ctx, path); err == nil {
		_ = reopened.Close()
		t.Fatal("divergent migration ledger was accepted")
	}
}

func nowForTest() time.Time { return time.Now().UTC() }
