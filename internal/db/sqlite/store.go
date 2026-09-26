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
	db        *sql.DB
	secretKey []byte
}

func Open(ctx context.Context, path string) (*Store, error) {
	return open(ctx, path, nil)
}

func OpenWithKey(ctx context.Context, path string, secretKey []byte) (*Store, error) {
	if len(secretKey) != 32 {
		return nil, errors.New("Platform MFA key must be exactly 32 bytes")
	}
	return open(ctx, path, secretKey)
}

func open(ctx context.Context, path string, secretKey []byte) (*Store, error) {
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
	store := &Store{db: db, secretKey: append([]byte(nil), secretKey...)}
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
	return []migration{
		{
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
		},
		{
			version: 2,
			name:    "sessions-mfa-recovery",
			statements: []string{
				`ALTER TABLE users ADD COLUMN mfa_enabled INTEGER NOT NULL DEFAULT 0 CHECK(mfa_enabled IN (0,1))`,
				`ALTER TABLE users ADD COLUMN mfa_secret_ciphertext BLOB`,
				`ALTER TABLE users ADD COLUMN mfa_last_step INTEGER NOT NULL DEFAULT -1`,
				`CREATE TABLE mfa_enrollments (
					id INTEGER PRIMARY KEY,
					user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					token_hash TEXT NOT NULL UNIQUE,
					secret_ciphertext BLOB NOT NULL,
					expires_at TEXT NOT NULL,
					attempts INTEGER NOT NULL DEFAULT 0,
					created_at TEXT NOT NULL,
					used_at TEXT
				)`,
				`CREATE INDEX mfa_enrollments_expiry_idx ON mfa_enrollments(expires_at, used_at)`,
				`CREATE TABLE mfa_recovery_codes (
					id INTEGER PRIMARY KEY,
					user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					code_hash TEXT NOT NULL,
					created_at TEXT NOT NULL,
					used_at TEXT
				)`,
				`CREATE INDEX mfa_recovery_active_idx ON mfa_recovery_codes(user_id, used_at)`,
				`CREATE INDEX sessions_expiry_idx ON sessions(expires_at, revoked_at)`,
			},
		},
		{
			version: 3,
			name:    "organisation-authority",
			statements: []string{
				`CREATE TABLE organisations (
					id INTEGER PRIMARY KEY,
					public_id TEXT NOT NULL UNIQUE,
					name TEXT NOT NULL,
					status TEXT NOT NULL CHECK(status IN ('active','archived')),
					created_at TEXT NOT NULL,
					archived_at TEXT
				)`,
				`CREATE TABLE organisation_memberships (
					id INTEGER PRIMARY KEY,
					public_id TEXT NOT NULL UNIQUE,
					organisation_id INTEGER NOT NULL REFERENCES organisations(id),
					user_id INTEGER NOT NULL REFERENCES users(id),
					role TEXT NOT NULL CHECK(role IN ('owner','admin','member')),
					active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
					assigned_by INTEGER NOT NULL REFERENCES users(id),
					assigned_at TEXT NOT NULL,
					removed_by INTEGER REFERENCES users(id),
					removed_at TEXT,
					removal_reason TEXT NOT NULL DEFAULT ''
				)`,
				`CREATE UNIQUE INDEX organisation_memberships_current_idx ON organisation_memberships(organisation_id,user_id) WHERE active=1`,
				`CREATE UNIQUE INDEX organisation_memberships_owner_idx ON organisation_memberships(organisation_id) WHERE active=1 AND role='owner'`,
				`CREATE INDEX organisation_memberships_scope_idx ON organisation_memberships(organisation_id,user_id,active,role)`,
				`CREATE TABLE audit_events (
					id INTEGER PRIMARY KEY,
					actor_user_id INTEGER NOT NULL REFERENCES users(id),
					aggregate_type TEXT NOT NULL,
					aggregate_id TEXT NOT NULL,
					event TEXT NOT NULL,
					details TEXT NOT NULL DEFAULT '',
					occurred_at TEXT NOT NULL
				)`,
				`CREATE INDEX audit_events_aggregate_time_idx ON audit_events(aggregate_type,aggregate_id,occurred_at,id)`,
			},
		},
		{
			version: 4,
			name:    "organisation-administration-authority",
			statements: []string{
				`CREATE TABLE system_role_assignments (
					id INTEGER PRIMARY KEY,
					user_id INTEGER NOT NULL REFERENCES users(id),
					role TEXT NOT NULL CHECK(role='system_admin'),
					active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
					assigned_by INTEGER NOT NULL REFERENCES users(id),
					assigned_at TEXT NOT NULL,
					revoked_by INTEGER REFERENCES users(id),
					revoked_at TEXT,
					revocation_reason TEXT NOT NULL DEFAULT ''
				)`,
				`CREATE UNIQUE INDEX system_role_assignments_current_idx ON system_role_assignments(user_id,role) WHERE active=1`,
				`CREATE INDEX system_role_assignments_active_idx ON system_role_assignments(user_id,active,role)`,
			},
		},
		{
			version: 5,
			name:    "workspace-authority",
			statements: []string{
				`CREATE TABLE workspaces (
					id INTEGER PRIMARY KEY,
					public_id TEXT NOT NULL UNIQUE,
					organisation_id INTEGER NOT NULL REFERENCES organisations(id),
					name TEXT NOT NULL,
					status TEXT NOT NULL CHECK(status IN ('active','archived')),
					created_at TEXT NOT NULL,
					archived_at TEXT
				)`,
				`CREATE INDEX workspaces_organisation_status_idx ON workspaces(organisation_id,status,public_id)`,
				`CREATE TABLE workspace_memberships (
					id INTEGER PRIMARY KEY,
					public_id TEXT NOT NULL UNIQUE,
					workspace_id INTEGER NOT NULL REFERENCES workspaces(id),
					user_id INTEGER NOT NULL REFERENCES users(id),
					role TEXT NOT NULL CHECK(role IN ('admin','member','viewer')),
					active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1)),
					assigned_by INTEGER NOT NULL REFERENCES users(id),
					assigned_at TEXT NOT NULL,
					removed_by INTEGER REFERENCES users(id),
					removed_at TEXT,
					removal_reason TEXT NOT NULL DEFAULT ''
				)`,
				`CREATE UNIQUE INDEX workspace_memberships_current_idx ON workspace_memberships(workspace_id,user_id) WHERE active=1`,
				`CREATE INDEX workspace_memberships_scope_idx ON workspace_memberships(workspace_id,user_id,active,role)`,
			},
		},
		{
			version: 6,
			name:    "product-catalogue-organisation-entitlements",
			statements: []string{
				`CREATE TABLE products (
					product_key TEXT PRIMARY KEY CHECK(length(product_key)>8 AND substr(product_key,1,8)='product.'),
					display_name TEXT NOT NULL CHECK(length(trim(display_name))>0 AND length(display_name)<=200),
					status TEXT NOT NULL CHECK(status IN ('active','retired')),
					created_at TEXT NOT NULL,
					retired_at TEXT,
					CHECK((status='active' AND retired_at IS NULL) OR (status='retired' AND retired_at IS NOT NULL))
				)`,
				`INSERT INTO products(product_key,display_name,status,created_at) VALUES
					('product.tockrctrl','TockrCTRL','active','2026-09-16T00:00:00Z'),
					('product.tockrims','TockrIMS','active','2026-09-16T00:00:00Z')`,
				`CREATE TABLE organisation_product_entitlements (
					id INTEGER PRIMARY KEY,
					public_id TEXT NOT NULL UNIQUE,
					organisation_id INTEGER NOT NULL REFERENCES organisations(id),
					product_key TEXT NOT NULL REFERENCES products(product_key),
					status TEXT NOT NULL CHECK(status IN ('active','revoked')),
					granted_by INTEGER NOT NULL REFERENCES users(id),
					granted_at TEXT NOT NULL,
					revoked_by INTEGER REFERENCES users(id),
					revoked_at TEXT,
					revocation_reason TEXT NOT NULL DEFAULT '',
					CHECK((status='active' AND revoked_by IS NULL AND revoked_at IS NULL AND revocation_reason='') OR
						(status='revoked' AND revoked_by IS NOT NULL AND revoked_at IS NOT NULL AND length(trim(revocation_reason))>0))
				)`,
				`CREATE UNIQUE INDEX organisation_product_entitlements_current_idx ON organisation_product_entitlements(organisation_id,product_key) WHERE status='active'`,
				`CREATE INDEX organisation_product_entitlements_scope_idx ON organisation_product_entitlements(organisation_id,product_key,status,granted_at,id)`,
			},
		},
		{
			version: 7,
			name:    "user-product-assignments",
			statements: []string{
				`CREATE TABLE user_product_assignments (
					id INTEGER PRIMARY KEY,
					public_id TEXT NOT NULL UNIQUE,
					user_id INTEGER NOT NULL REFERENCES users(id),
					organisation_id INTEGER NOT NULL REFERENCES organisations(id),
					product_key TEXT NOT NULL REFERENCES products(product_key),
					status TEXT NOT NULL CHECK(status IN ('active','revoked')),
					assigned_by INTEGER NOT NULL REFERENCES users(id),
					assigned_at TEXT NOT NULL,
					revoked_by INTEGER REFERENCES users(id),
					revoked_at TEXT,
					revocation_reason TEXT NOT NULL DEFAULT '',
					CHECK((status='active' AND revoked_by IS NULL AND revoked_at IS NULL AND revocation_reason='') OR
						(status='revoked' AND revoked_by IS NOT NULL AND revoked_at IS NOT NULL AND length(trim(revocation_reason))>0))
				)`,
				`CREATE UNIQUE INDEX user_product_assignments_current_idx ON user_product_assignments(user_id,organisation_id,product_key) WHERE status='active'`,
				`CREATE INDEX user_product_assignments_scope_idx ON user_product_assignments(organisation_id,user_id,product_key,status,assigned_at,id)`,
			},
		},
		{
			version: 8,
			name:    "platform-event-outbox",
			statements: []string{
				`CREATE TABLE platform_outbox (
					id INTEGER PRIMARY KEY,
					event_id TEXT NOT NULL UNIQUE,
					event_type TEXT NOT NULL,
					aggregate_type TEXT NOT NULL CHECK(aggregate_type IN ('User','Organisation','Workspace','Product','Access')),
					aggregate_id TEXT NOT NULL,
					sequence INTEGER NOT NULL CHECK(sequence>0),
					schema_version INTEGER NOT NULL CHECK(schema_version=1),
					occurred_at TEXT NOT NULL,
					payload TEXT NOT NULL CHECK(length(payload)>1 AND length(payload)<=4096),
					published_at TEXT,
					UNIQUE(aggregate_type,aggregate_id,sequence)
				)`,
				`CREATE INDEX platform_outbox_pending_idx ON platform_outbox(published_at,id)`,
				`CREATE INDEX platform_outbox_aggregate_idx ON platform_outbox(aggregate_type,aggregate_id,sequence)`,
			},
		},
		{
			version: 9,
			name:    "platform-projection-inbox",
			statements: []string{
				`CREATE TABLE platform_projection_inbox (
					id INTEGER PRIMARY KEY,
					consumer_key TEXT NOT NULL CHECK(length(trim(consumer_key))>0 AND length(consumer_key)<=100),
					event_id TEXT NOT NULL,
					event_type TEXT NOT NULL CHECK(length(event_type)>0 AND length(event_type)<=200),
					aggregate_type TEXT NOT NULL CHECK(length(aggregate_type)>0 AND length(aggregate_type)<=50),
					aggregate_id TEXT NOT NULL CHECK(length(aggregate_id)>0 AND length(aggregate_id)<=200),
					sequence INTEGER NOT NULL CHECK(sequence>0),
					schema_version INTEGER NOT NULL CHECK(schema_version>0),
					occurred_at TEXT NOT NULL,
					payload TEXT NOT NULL CHECK(length(payload)>1 AND length(payload)<=4096),
					state TEXT NOT NULL CHECK(state IN ('pending','applied','gap','stale','blocked')),
					reason TEXT NOT NULL DEFAULT '' CHECK(length(reason)<=500),
					received_at TEXT NOT NULL,
					applied_at TEXT,
					UNIQUE(consumer_key,event_id),
					CHECK((state='applied' AND applied_at IS NOT NULL) OR (state<>'applied' AND applied_at IS NULL))
				)`,
				`CREATE INDEX platform_projection_inbox_order_idx ON platform_projection_inbox(consumer_key,aggregate_type,aggregate_id,sequence,id)`,
				`CREATE INDEX platform_projection_inbox_state_idx ON platform_projection_inbox(consumer_key,state,id)`,
				`CREATE TABLE platform_projection_checkpoints (
					id INTEGER PRIMARY KEY,
					consumer_key TEXT NOT NULL CHECK(length(trim(consumer_key))>0 AND length(consumer_key)<=100),
					aggregate_type TEXT NOT NULL CHECK(length(aggregate_type)>0 AND length(aggregate_type)<=50),
					aggregate_id TEXT NOT NULL CHECK(length(aggregate_id)>0 AND length(aggregate_id)<=200),
					last_sequence INTEGER NOT NULL CHECK(last_sequence>=0),
					last_event_id TEXT NOT NULL DEFAULT '',
					state TEXT NOT NULL CHECK(state IN ('current','stale','gap','blocked','unavailable')),
					reason TEXT NOT NULL DEFAULT '' CHECK(length(reason)<=500),
					updated_at TEXT NOT NULL,
					UNIQUE(consumer_key,aggregate_type,aggregate_id)
				)`,
				`CREATE INDEX platform_projection_checkpoint_state_idx ON platform_projection_checkpoints(consumer_key,state,updated_at)`,
			},
		},
		{
			version: 10,
			name:    "platform-read-authority-snapshots",
			statements: []string{
				`CREATE TABLE platform_read_authority_snapshots (
					id INTEGER PRIMARY KEY,
					snapshot_id TEXT NOT NULL UNIQUE CHECK(length(snapshot_id)>5 AND substr(snapshot_id,1,5)='snap_'),
					consumer_key TEXT NOT NULL CHECK(length(trim(consumer_key))>0 AND length(consumer_key)<=100),
					contract_version TEXT NOT NULL CHECK(contract_version='platform.read-authority.v2'),
					source_cursor TEXT NOT NULL CHECK(length(source_cursor)>4 AND substr(source_cursor,1,4)='cur_'),
					checksum_algorithm TEXT NOT NULL CHECK(checksum_algorithm='sha256'),
					checksum TEXT NOT NULL CHECK(length(checksum)=64),
					created_at TEXT NOT NULL,
					expires_at TEXT NOT NULL,
					record_count INTEGER NOT NULL CHECK(record_count>=0 AND record_count<=10000),
					page_size INTEGER NOT NULL CHECK(page_size>0 AND page_size<=500),
					complete INTEGER NOT NULL CHECK(complete IN (0,1)),
					finalized_at TEXT,
					CHECK((complete=1 AND finalized_at IS NOT NULL) OR (complete=0 AND finalized_at IS NULL)),
					CHECK(expires_at>created_at)
				)`,
				`CREATE INDEX platform_read_authority_snapshots_consumer_idx ON platform_read_authority_snapshots(consumer_key,created_at,snapshot_id)`,
				`CREATE TABLE platform_read_authority_snapshot_records (
					id INTEGER PRIMARY KEY,
					snapshot_id TEXT NOT NULL REFERENCES platform_read_authority_snapshots(snapshot_id) ON DELETE CASCADE,
					ordinal INTEGER NOT NULL CHECK(ordinal>=0 AND ordinal<10000),
					entity_kind TEXT NOT NULL,
					record_id TEXT NOT NULL,
					record_hash TEXT NOT NULL CHECK(length(record_hash)=64),
					record_json TEXT NOT NULL CHECK(length(record_json)>1 AND length(record_json)<=4096),
					UNIQUE(snapshot_id,ordinal),
					UNIQUE(snapshot_id,entity_kind,record_id),
					UNIQUE(snapshot_id,record_hash)
				)`,
				`CREATE INDEX platform_read_authority_snapshot_records_page_idx ON platform_read_authority_snapshot_records(snapshot_id,ordinal)`,
				`CREATE INDEX platform_read_authority_snapshot_records_kind_idx ON platform_read_authority_snapshot_records(snapshot_id,entity_kind,ordinal)`,
			},
		},
		{
			version: 11,
			name:    "platform-read-authority-nonce-replay",
			statements: []string{
				`CREATE TABLE platform_read_authority_nonces (
					id INTEGER PRIMARY KEY,
					consumer_key TEXT NOT NULL CHECK(length(trim(consumer_key))>0 AND length(consumer_key)<=100),
					nonce TEXT NOT NULL CHECK(length(nonce)>0 AND length(nonce)<=128),
					reserved_at TEXT NOT NULL,
					expires_at TEXT NOT NULL CHECK(expires_at>reserved_at),
					UNIQUE(consumer_key,nonce)
				)`,
				`CREATE INDEX platform_read_authority_nonces_expiry_idx ON platform_read_authority_nonces(expires_at)`,
			},
		},
		{
			version: 12,
			name:    "membership-command-idempotency-and-versions",
			statements: []string{
				`ALTER TABLE organisation_memberships ADD COLUMN membership_version INTEGER NOT NULL DEFAULT 1 CHECK(membership_version>0)`,
				`ALTER TABLE workspace_memberships ADD COLUMN membership_version INTEGER NOT NULL DEFAULT 1 CHECK(membership_version>0)`,
				`CREATE TABLE platform_membership_command_results (
					id INTEGER PRIMARY KEY,
					consumer_key TEXT NOT NULL CHECK(length(trim(consumer_key))>0 AND length(consumer_key)<=100),
					idempotency_key TEXT NOT NULL CHECK(length(trim(idempotency_key))>0 AND length(idempotency_key)<=128),
					request_hash TEXT NOT NULL CHECK(length(request_hash)=64),
					result_json TEXT NOT NULL CHECK(length(result_json)>1 AND length(result_json)<=4096),
					created_at TEXT NOT NULL,
					UNIQUE(consumer_key,idempotency_key)
				)`,
				`CREATE INDEX platform_membership_command_results_created_idx ON platform_membership_command_results(created_at)`,
			},
		},
		{
			version: 13,
			name:    "platform-production-import-ledger",
			statements: []string{
				`CREATE TABLE platform_import_runs (
					id INTEGER PRIMARY KEY,
					manifest_id TEXT NOT NULL UNIQUE CHECK(length(manifest_id)=64),
					manifest_digest TEXT NOT NULL CHECK(length(manifest_digest)=64),
					contract_version TEXT NOT NULL CHECK(contract_version='platform.reconciliation.import.v1'),
					source_report_sha256 TEXT NOT NULL CHECK(length(source_report_sha256)=64),
					source_snapshots_json TEXT NOT NULL CHECK(length(source_snapshots_json)>1 AND length(source_snapshots_json)<=4096),
					approval_id TEXT NOT NULL CHECK(length(trim(approval_id))>0 AND length(approval_id)<=200),
					operator_id TEXT NOT NULL CHECK(length(trim(operator_id))>0 AND length(operator_id)<=200),
					key_id TEXT NOT NULL CHECK(length(trim(key_id))>0 AND length(key_id)<=200),
					approved_at TEXT NOT NULL,
					status TEXT NOT NULL CHECK(status IN ('running','paused','completed','rolled_back')),
					next_index INTEGER NOT NULL CHECK(next_index>=0),
					applied_count INTEGER NOT NULL CHECK(applied_count>=0),
					created_count INTEGER NOT NULL CHECK(created_count>=0),
					reconciled_count INTEGER NOT NULL CHECK(reconciled_count>=0),
					conflict_count INTEGER NOT NULL CHECK(conflict_count>=0),
					checkpoint_mac TEXT NOT NULL CHECK(length(checkpoint_mac)=64),
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL,
					receipt_json TEXT NOT NULL CHECK(length(receipt_json)>1 AND length(receipt_json)<=8192)
				)`,
				`CREATE TABLE platform_import_records (
					id INTEGER PRIMARY KEY,
					manifest_id TEXT NOT NULL REFERENCES platform_import_runs(manifest_id) ON DELETE CASCADE,
					record_order INTEGER NOT NULL CHECK(record_order>=0),
					entity_kind TEXT NOT NULL CHECK(length(trim(entity_kind))>0 AND length(entity_kind)<=100),
					platform_id TEXT NOT NULL CHECK(length(trim(platform_id))>0 AND length(platform_id)<=200),
					record_digest TEXT NOT NULL CHECK(length(record_digest)=64),
					source_refs_json TEXT NOT NULL CHECK(length(source_refs_json)>1 AND length(source_refs_json)<=4096),
					outcome TEXT NOT NULL CHECK(outcome IN ('created','reconciled','skipped','conflict','rolled_back')),
					created_by_import INTEGER NOT NULL CHECK(created_by_import IN (0,1)),
					compensated INTEGER NOT NULL DEFAULT 0 CHECK(compensated IN (0,1)),
					applied_at TEXT NOT NULL,
					UNIQUE(manifest_id,record_order),
					UNIQUE(manifest_id,platform_id)
				)`,
				`CREATE INDEX platform_import_records_manifest_order_idx ON platform_import_records(manifest_id,record_order)`,
				`CREATE INDEX platform_import_records_created_idx ON platform_import_records(manifest_id,created_by_import,compensated)`,
				`CREATE TABLE platform_import_compensations (
					id INTEGER PRIMARY KEY,
					manifest_id TEXT NOT NULL REFERENCES platform_import_runs(manifest_id) ON DELETE CASCADE,
					record_order INTEGER NOT NULL,
					entity_kind TEXT NOT NULL,
					platform_id TEXT NOT NULL,
					status TEXT NOT NULL CHECK(status IN ('compensated','skipped','blocked')),
					reason TEXT NOT NULL DEFAULT '' CHECK(length(reason)<=500),
					occurred_at TEXT NOT NULL,
					UNIQUE(manifest_id,record_order)
				)`,
			},
		},
		{
			version: 14,
			name:    "audit-actor-tombstone-provenance",
			statements: []string{
				`ALTER TABLE audit_events RENAME TO audit_events_legacy`,
				`CREATE TABLE audit_events (
					id INTEGER PRIMARY KEY,
					actor_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
					actor_user_public_id TEXT NOT NULL DEFAULT '',
					aggregate_type TEXT NOT NULL,
					aggregate_id TEXT NOT NULL,
					event TEXT NOT NULL,
					details TEXT NOT NULL DEFAULT '',
					occurred_at TEXT NOT NULL
				)`,
				`INSERT INTO audit_events(id,actor_user_id,actor_user_public_id,aggregate_type,aggregate_id,event,details,occurred_at)
				 SELECT a.id,a.actor_user_id,COALESCE(u.public_id,''),a.aggregate_type,a.aggregate_id,a.event,a.details,a.occurred_at
				 FROM audit_events_legacy a LEFT JOIN users u ON u.id=a.actor_user_id`,
				`DROP TABLE audit_events_legacy`,
				`CREATE INDEX audit_events_aggregate_time_idx ON audit_events(aggregate_type,aggregate_id,occurred_at,id)`,
			},
		},
	}
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
