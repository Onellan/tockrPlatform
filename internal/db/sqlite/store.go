package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("sqlite path is required")
	}
	if path != ":memory:" && !strings.HasPrefix(path, "file:") {
		path = filepath.Clean(path)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// Platform deliberately runs one SQLite connection until a separately
	// authorised and measured pool upgrade. This prevents callers from
	// accidentally relying on multi-connection semantics.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	store := &Store{db: db}
	if err := store.configure(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) configure(ctx context.Context) error {
	for _, statement := range []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
	} {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("configure sqlite (%s): %w", statement, err)
		}
	}
	return nil
}

func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, checksum TEXT NOT NULL, applied_at TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}
	migrations := supportedMigrations()
	rows, err := s.db.QueryContext(ctx, `SELECT version,name,checksum FROM schema_migrations ORDER BY version`)
	if err != nil {
		return fmt.Errorf("read migration ledger: %w", err)
	}
	defer rows.Close()
	applied := 0
	for rows.Next() {
		var version int
		var name, checksum string
		if err := rows.Scan(&version, &name, &checksum); err != nil {
			return fmt.Errorf("read migration row: %w", err)
		}
		if applied >= len(migrations) || version != migrations[applied].version || name != migrations[applied].name || checksum != migrationChecksum(migrations[applied]) {
			return fmt.Errorf("migration ledger diverges at version %d", version)
		}
		applied++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate migration ledger: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close migration ledger: %w", err)
	}
	for _, migration := range migrations[applied:] {
		if err := s.applyMigration(ctx, migration); err != nil {
			return err
		}
	}
	return nil
}

type migration struct {
	version    int
	name       string
	statements []string
}

func supportedMigrations() []migration {
	return []migration{{
		version: 1,
		name:    "identity-authentication-foundation",
		statements: []string{
			`CREATE TABLE users (
				id INTEGER PRIMARY KEY,
				public_id TEXT NOT NULL UNIQUE,
				email TEXT NOT NULL UNIQUE COLLATE NOCASE,
				display_name TEXT NOT NULL,
				password_hash TEXT NOT NULL,
				active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
				created_at TEXT NOT NULL,
				last_login_at TEXT
			)`,
			`CREATE TABLE sessions (
				id INTEGER PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				token_hash TEXT NOT NULL UNIQUE,
				csrf_token_hash TEXT NOT NULL,
				created_at TEXT NOT NULL,
				expires_at TEXT NOT NULL,
				revoked_at TEXT
			)`,
			`CREATE INDEX sessions_active_token_idx ON sessions(token_hash, expires_at, revoked_at)`,
			`CREATE TABLE security_events (
				id INTEGER PRIMARY KEY,
				user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
				event TEXT NOT NULL,
				details TEXT NOT NULL DEFAULT '',
				occurred_at TEXT NOT NULL
			)`,
			`CREATE INDEX security_events_user_time_idx ON security_events(user_id, occurred_at)`,
		},
	}}
}

func migrationChecksum(value migration) string {
	hash := sha256.Sum256([]byte(strings.Join(value.statements, "\n")))
	return hex.EncodeToString(hash[:])
}

func (s *Store) applyMigration(ctx context.Context, value migration) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %d: %w", value.version, err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, statement := range value.statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply migration %d: %w", value.version, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version,name,checksum,applied_at) VALUES(?,?,?,?)`, value.version, value.name, migrationChecksum(value), time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("record migration %d: %w", value.version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %d: %w", value.version, err)
	}
	return nil
}
