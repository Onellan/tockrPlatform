package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/Onellan/tockrplatform/internal/auth"
	"github.com/Onellan/tockrplatform/internal/domain"
)

var testMFAKey = []byte("01234567890123456789012345678901")

func newMFAStore(t *testing.T) (*Store, domain.User) {
	t.Helper()
	ctx := context.Background()
	store, err := OpenWithKey(ctx, filepath.Join(t.TempDir(), "mfa.db"), testMFAKey)
	if err != nil {
		t.Fatal(err)
	}
	userID, err := domain.NewUserID()
	if err != nil {
		t.Fatal(err)
	}
	user, err := store.CreateUser(ctx, domain.User{ID: userID, Email: "mfa@example.test", DisplayName: "MFA User", Active: true}, "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, user
}

func TestMFAEnrollmentVerificationAndOneTimeRecovery(t *testing.T) {
	ctx := context.Background()
	store, user := newMFAStore(t)
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	now := time.Now().UTC()
	token, err := store.CreateMFAEnrollment(ctx, user.ID, secret, now.Add(10*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	recoveryCodes, err := auth.GenerateRecoveryCodes(10)
	if err != nil {
		t.Fatal(err)
	}
	hashes := make([]string, 0, len(recoveryCodes))
	for _, code := range recoveryCodes {
		hash, err := auth.HashPassword(auth.NormalizeRecoveryCode(code))
		if err != nil {
			t.Fatal(err)
		}
		hashes = append(hashes, hash)
	}
	verificationAt := time.Now().UTC()
	code, err := auth.TOTPCode(secret, verificationAt)
	if err != nil {
		t.Fatal(err)
	}
	complete, err := store.CompleteMFAEnrollment(ctx, user.ID, token, code, hashes, verificationAt)
	if err != nil || !complete {
		t.Fatalf("complete MFA = %v, err=%v", complete, err)
	}
	valid, err := store.VerifyMFA(ctx, user.ID, code)
	if err != nil || !valid {
		t.Fatalf("verify MFA = %v, err=%v", valid, err)
	}
	valid, err = store.VerifyMFA(ctx, user.ID, code)
	if err != nil || valid {
		t.Fatalf("replayed MFA = %v, err=%v", valid, err)
	}
	used, err := store.UseRecoveryCode(ctx, user.ID, recoveryCodes[0], now.Add(time.Second))
	if err != nil || !used {
		t.Fatalf("use recovery code = %v, err=%v", used, err)
	}
	used, err = store.UseRecoveryCode(ctx, user.ID, recoveryCodes[0], now.Add(2*time.Second))
	if err != nil || used {
		t.Fatalf("replayed recovery code = %v, err=%v", used, err)
	}
	events, err := store.SecurityEventCount(ctx, "mfa_recovery_used")
	if err != nil || events != 1 {
		t.Fatalf("recovery audit count = %d, err=%v", events, err)
	}
	projected, err := store.AuthenticatedSession(ctx, "not-a-token", now)
	if projected.User.PasswordHash != "" || err == nil {
		t.Fatalf("invalid projection = %#v, err=%v", projected, err)
	}
}

func TestMFARequiresConfiguredEncryptionKey(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "no-key.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	userID, err := domain.NewUserID()
	if err != nil {
		t.Fatal(err)
	}
	user, err := store.CreateUser(ctx, domain.User{ID: userID, Email: "nokey@example.test", DisplayName: "No Key", Active: true}, "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateMFAEnrollment(ctx, user.ID, "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", time.Now().UTC().Add(time.Minute)); err == nil {
		t.Fatal("MFA enrollment succeeded without encryption key")
	}
}

func TestMFAFreshUpgradeAndReopenMigrationEvidence(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "migration.db")
	store, err := OpenWithKey(ctx, path, testMFAKey)
	if err != nil {
		t.Fatal(err)
	}
	var freshVersion int
	if err := store.DB().QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&freshVersion); err != nil {
		t.Fatal(err)
	}
	if freshVersion != 13 {
		t.Fatalf("fresh migration version = %d, want 13", freshVersion)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenWithKey(ctx, path, testMFAKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}

	legacyPath := filepath.Join(t.TempDir(), "upgrade.db")
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
	upgraded, err := OpenWithKey(ctx, legacyPath, testMFAKey)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Close()
	var upgradedVersion int
	if err := upgraded.DB().QueryRowContext(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&upgradedVersion); err != nil {
		t.Fatal(err)
	}
	if upgradedVersion != 13 {
		t.Fatalf("upgraded migration version = %d, want 13", upgradedVersion)
	}
}
