package sqlite

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Backup creates a coordinated SQLite copy. WAL is checkpointed before
// VACUUM INTO so a live database is never copied as a bare main file.
func (s *Store) Backup(ctx context.Context, destination string) error {
	if strings.TrimSpace(destination) == "" || (!filepath.IsAbs(destination) && filepath.Clean(destination) == ".") {
		return fmt.Errorf("backup destination is required")
	}
	destination = filepath.Clean(destination)
	if _, err := os.Stat(destination); err == nil {
		return fmt.Errorf("backup destination already exists")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect backup destination: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o750); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `PRAGMA wal_checkpoint(FULL)`); err != nil {
		return fmt.Errorf("checkpoint WAL: %w", err)
	}
	quoted := "'" + strings.ReplaceAll(destination, "'", "''") + "'"
	if _, err := s.db.ExecContext(ctx, "VACUUM INTO "+quoted); err != nil { // #nosec G202 -- local path is SQL literal syntax
		return fmt.Errorf("create SQLite backup: %w", err)
	}
	return nil
}
