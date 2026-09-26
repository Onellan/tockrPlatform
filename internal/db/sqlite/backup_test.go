package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

func TestBackupRestorePreservesCoreState(t *testing.T) {
	ctx := context.Background()
	source, err := Open(ctx, filepath.Join(t.TempDir(), "source.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = source.Close() }()
	if _, err := source.DB().ExecContext(ctx, `INSERT INTO users(public_id,email,display_name,password_hash,active,created_at) VALUES('usr_backup','backup@example.test','Backup User','hash',1,'2026-09-26T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(t.TempDir(), "backup.db")
	if err := source.Backup(ctx, backup); err != nil {
		t.Fatal(err)
	}
	restored, err := Open(ctx, backup)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = restored.Close() }()
	var count int
	if err := restored.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE public_id='usr_backup'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("restored user count=%d err=%v", count, err)
	}
	var integrity string
	if err := restored.DB().QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatalf("integrity=%q err=%v", integrity, err)
	}
}
