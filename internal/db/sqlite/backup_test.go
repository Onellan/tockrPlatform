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
	if _, err := source.DB().ExecContext(ctx, `
		INSERT INTO organisations(public_id,name,status,created_at) VALUES('org_backup','Backup Organisation','active','2026-09-26T00:00:00Z');
		INSERT INTO organisation_memberships(public_id,organisation_id,user_id,role,active,assigned_by,assigned_at) VALUES('om_backup',(SELECT id FROM organisations WHERE public_id='org_backup'),(SELECT id FROM users WHERE public_id='usr_backup'),'owner',1,(SELECT id FROM users WHERE public_id='usr_backup'),'2026-09-26T00:00:00Z');
		INSERT INTO audit_events(actor_user_id,actor_user_public_id,aggregate_type,aggregate_id,event,details,occurred_at) VALUES((SELECT id FROM users WHERE public_id='usr_backup'),'usr_backup','Organisation','org_backup','membership.assigned','historical audit','2026-09-26T00:00:00Z');
		INSERT INTO platform_import_runs(manifest_id,manifest_digest,contract_version,source_report_sha256,source_snapshots_json,approval_id,operator_id,key_id,approved_at,status,next_index,applied_count,created_count,reconciled_count,conflict_count,checkpoint_mac,created_at,updated_at,receipt_json) VALUES('aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb','platform.reconciliation.import.v1','cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc','[{"source":"ctrl"}]','approval-backup','operator-backup','key-backup','2026-09-26T00:00:00Z','rolled_back',1,1,0,1,0,'dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd','2026-09-26T00:00:00Z','2026-09-26T00:00:01Z','{"status":"rolled_back","manifest_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}');
		INSERT INTO platform_import_records(manifest_id,record_order,entity_kind,platform_id,record_digest,source_refs_json,outcome,created_by_import,compensated,applied_at) VALUES('aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',0,'user','usr_backup','eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee','[{"source":"ctrl","id":"local-1"}]','rolled_back',1,1,'2026-09-26T00:00:01Z');`); err != nil {
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
	checks := []struct {
		name  string
		query string
		want  int
	}{
		{"membership", `SELECT COUNT(*) FROM organisation_memberships WHERE public_id='om_backup' AND organisation_id=(SELECT id FROM organisations WHERE public_id='org_backup')`, 1},
		{"audit", `SELECT COUNT(*) FROM audit_events WHERE aggregate_id='org_backup' AND event='membership.assigned'`, 1},
		{"rollback receipt", `SELECT COUNT(*) FROM platform_import_runs WHERE manifest_id='aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa' AND status='rolled_back'`, 1},
		{"historical import record", `SELECT COUNT(*) FROM platform_import_records WHERE manifest_id='aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa' AND compensated=1`, 1},
	}
	for _, check := range checks {
		if err := restored.DB().QueryRowContext(ctx, check.query).Scan(&count); err != nil || count != check.want {
			t.Fatalf("restored %s count=%d err=%v", check.name, count, err)
		}
	}
	var integrity string
	if err := restored.DB().QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatalf("integrity=%q err=%v", integrity, err)
	}
}
