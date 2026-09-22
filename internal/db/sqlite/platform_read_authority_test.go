package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
	"github.com/Onellan/tockrplatform/internal/platform/readauthority"
	"github.com/Onellan/tockrplatform/internal/store"
)

func TestReadAuthoritySnapshotFreshSeedIsCompleteDeterministicAndPaged(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "read-authority.db")
	persistence, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer persistence.Close()

	created := time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC)
	snapshot, err := persistence.CreateReadAuthoritySnapshot(ctx, readauthority.ConsumerCTRL, []string{
		string(readauthority.EntityUser), string(readauthority.EntityOrganisation), string(readauthority.EntityOrganisationMembership),
		string(readauthority.EntityWorkspace), string(readauthority.EntityWorkspaceMembership), string(readauthority.EntityProduct),
		string(readauthority.EntityOrganisationEntitlement), string(readauthority.EntityUserProductAssignment),
	}, time.Hour, 500, created)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.SourceCursor != "cur_0" || snapshot.RecordCount != 2 || !snapshot.Complete || snapshot.ContractVersion != readauthority.VersionV2 {
		t.Fatalf("fresh snapshot = %#v", snapshot)
	}

	page, err := persistence.ListReadAuthoritySnapshotRecords(ctx, readauthority.ConsumerCTRL, snapshot.SnapshotID, string(readauthority.EntityProduct), "", 1, created.Add(time.Minute))
	if err != nil || len(page.Records) != 1 || page.Complete || page.NextCursor == "" {
		t.Fatalf("first product page = %#v, err=%v", page, err)
	}
	if page.Records[0].ProvenanceKind != string(readauthority.ProvenanceMigrationSeed) || page.Records[0].MigrationVersion != 6 || page.Records[0].MigrationName != "product-catalogue-organisation-entitlements" || page.Records[0].SourceEventID != "" {
		t.Fatalf("seed provenance = %#v", page.Records[0])
	}
	last, err := persistence.ListReadAuthoritySnapshotRecords(ctx, readauthority.ConsumerCTRL, snapshot.SnapshotID, string(readauthority.EntityProduct), page.NextCursor, 1, created.Add(time.Minute))
	if err != nil || len(last.Records) != 1 || !last.Complete || last.NextCursor != "" || last.Records[0].ID == page.Records[0].ID {
		t.Fatalf("second product page = %#v, err=%v", last, err)
	}
	if _, err := persistence.GetReadAuthoritySnapshot(ctx, readauthority.ConsumerIMS, snapshot.SnapshotID, created.Add(time.Minute)); !errors.Is(err, store.ErrReadAuthoritySnapshotNotFound) {
		t.Fatalf("cross-consumer snapshot read = %v", err)
	}
}

func TestReadAuthoritySnapshotMaterializesCurrentFactsWithEventProvenance(t *testing.T) {
	ctx := context.Background()
	persistence, users := newOrganisationStore(t, 3)
	now := time.Date(2026, 9, 17, 11, 0, 0, 0, time.UTC)
	organisation, _, err := persistence.CreateOrganisation(ctx, users[0].ID, domain.Organisation{Name: "Snapshot Organisation"}, "snapshot organisation", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[1].ID, domain.OrganisationMember, "snapshot member", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	workspace, _, err := persistence.CreateWorkspace(ctx, users[0].ID, organisation.ID, domain.Workspace{Name: "Snapshot Workspace"}, "snapshot workspace", now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.EntitleOrganisation(ctx, users[0].ID, organisation.ID, readauthority.ProductCTRL, "snapshot entitlement", now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	assignment, err := persistence.AssignUserProduct(ctx, users[0].ID, organisation.ID, users[1].ID, readauthority.ProductCTRL, "snapshot assignment", now.Add(4*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if assignment.ID == "" || workspace.ID == "" {
		t.Fatal("fixture did not create access records")
	}

	snapshot, err := persistence.CreateReadAuthoritySnapshot(ctx, readauthority.ConsumerIMS, []string{
		string(readauthority.EntityUser), string(readauthority.EntityOrganisation), string(readauthority.EntityOrganisationMembership),
		string(readauthority.EntityWorkspace), string(readauthority.EntityWorkspaceMembership), string(readauthority.EntityProduct),
		string(readauthority.EntityOrganisationEntitlement), string(readauthority.EntityUserProductAssignment),
	}, time.Hour, 500, now.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.RecordCount < 10 || snapshot.SourceCursor == "cur_0" {
		t.Fatalf("populated snapshot = %#v", snapshot)
	}
	page, err := persistence.ListReadAuthoritySnapshotRecords(ctx, readauthority.ConsumerIMS, snapshot.SnapshotID, "", "", 500, now.Add(6*time.Minute))
	if err != nil || len(page.Records) != snapshot.RecordCount || !page.Complete {
		t.Fatalf("populated snapshot page = %#v, err=%v", page, err)
	}
	seenEvent, seenSeed := false, false
	for _, record := range page.Records {
		if record.ProvenanceKind == string(readauthority.ProvenanceEvent) {
			seenEvent = true
		}
		if record.ProvenanceKind == string(readauthority.ProvenanceMigrationSeed) {
			seenSeed = true
		}
		if record.EntityKind == string(readauthority.EntityUser) && record.SourceEventID == "" {
			t.Fatalf("user lacks event provenance: %#v", record)
		}
	}
	if !seenEvent || !seenSeed {
		t.Fatalf("expected event and migration provenance: event=%v seed=%v", seenEvent, seenSeed)
	}

	if _, err := persistence.DB().ExecContext(ctx, `UPDATE products SET status='retired',retired_at=? WHERE product_key=?`, formatTime(now.Add(7*time.Minute)), readauthority.ProductIMS); err != nil {
		t.Fatal(err)
	}
	unchanged, err := persistence.GetReadAuthoritySnapshot(ctx, readauthority.ConsumerIMS, snapshot.SnapshotID, now.Add(8*time.Minute))
	if err != nil || unchanged.Checksum != snapshot.Checksum || unchanged.RecordCount != snapshot.RecordCount {
		t.Fatalf("snapshot changed after source mutation = %#v, err=%v", unchanged, err)
	}
}

func TestReadAuthoritySnapshotIntegrityExpiryCleanupAndMissingProvenanceFailClosed(t *testing.T) {
	ctx := context.Background()
	persistence, err := Open(ctx, filepath.Join(t.TempDir(), "integrity.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer persistence.Close()
	created := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	snapshot, err := persistence.CreateReadAuthoritySnapshot(ctx, readauthority.ConsumerCTRL, []string{string(readauthority.EntityProduct)}, time.Hour, 1, created)
	if err != nil {
		t.Fatal(err)
	}
	var originalHash, originalJSON string
	if err := persistence.DB().QueryRowContext(ctx, `SELECT record_hash,record_json FROM platform_read_authority_snapshot_records WHERE snapshot_id=? AND ordinal=0`, snapshot.SnapshotID).Scan(&originalHash, &originalJSON); err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.DB().ExecContext(ctx, `UPDATE platform_read_authority_snapshot_records SET record_hash=? WHERE snapshot_id=? AND ordinal=0`, strings.Repeat("0", 64), snapshot.SnapshotID); err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.GetReadAuthoritySnapshot(ctx, readauthority.ConsumerCTRL, snapshot.SnapshotID, created.Add(time.Minute)); !errors.Is(err, store.ErrReadAuthorityChecksumMismatch) {
		t.Fatalf("record hash corruption = %v", err)
	}
	if _, err := persistence.DB().ExecContext(ctx, `UPDATE platform_read_authority_snapshot_records SET record_hash=?,record_json=? WHERE snapshot_id=? AND ordinal=0`, originalHash, originalJSON+"{}", snapshot.SnapshotID); err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.GetReadAuthoritySnapshot(ctx, readauthority.ConsumerCTRL, snapshot.SnapshotID, created.Add(time.Minute)); !errors.Is(err, store.ErrReadAuthorityChecksumMismatch) {
		t.Fatalf("trailing record corruption = %v", err)
	}

	expiring, err := persistence.CreateReadAuthoritySnapshot(ctx, readauthority.ConsumerCTRL, []string{string(readauthority.EntityProduct)}, time.Hour, 500, created.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.GetReadAuthoritySnapshot(ctx, readauthority.ConsumerCTRL, expiring.SnapshotID, expiring.ExpiresAt); !errors.Is(err, store.ErrReadAuthoritySnapshotExpired) {
		t.Fatalf("expired snapshot = %v", err)
	}
	removed, err := persistence.CleanupReadAuthoritySnapshots(ctx, readauthority.ConsumerCTRL, created.Add(time.Hour), 10)
	if err != nil || removed != 1 {
		t.Fatalf("cleanup = %d, err=%v", removed, err)
	}
	removed, err = persistence.CleanupReadAuthoritySnapshots(ctx, readauthority.ConsumerCTRL, expiring.ExpiresAt, 10)
	if err != nil || removed != 1 {
		t.Fatalf("second cleanup = %d, err=%v", removed, err)
	}
	if _, err := persistence.GetReadAuthoritySnapshot(ctx, readauthority.ConsumerCTRL, expiring.SnapshotID, created.Add(time.Hour)); !errors.Is(err, store.ErrReadAuthoritySnapshotNotFound) {
		t.Fatalf("cleaned snapshot = %v", err)
	}

	if _, err := persistence.DB().ExecContext(ctx, `INSERT INTO users(public_id,email,display_name,password_hash,active,created_at) VALUES(?,?,?,?,?,?)`, "usr_missing_provenance", "missing@example.test", "Missing", "not-a-password-hash", 1, formatTime(created)); err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.CreateReadAuthoritySnapshot(ctx, readauthority.ConsumerCTRL, []string{string(readauthority.EntityUser)}, time.Hour, 500, created.Add(2*time.Hour)); err == nil {
		t.Fatal("snapshot accepted a row without committed provenance")
	}
}

func TestReadAuthoritySnapshotMigrationFreshUpgradeReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "migration.db")
	first, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	assertReadAuthorityTables(t, first)
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	assertReadAuthorityTables(t, second)
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
}

func assertReadAuthorityTables(t *testing.T, persistence *Store) {
	t.Helper()
	var version int
	if err := persistence.DB().QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 12 {
		t.Fatalf("read-authority migration version = %d, want 12", version)
	}
	for _, table := range []string{"platform_read_authority_snapshots", "platform_read_authority_snapshot_records"} {
		var count int
		if err := persistence.DB().QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("table %s exists = %d, want 1", table, count)
		}
	}
}
