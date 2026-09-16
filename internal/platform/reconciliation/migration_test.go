package reconciliation

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
	"time"
)

func signedFixtureManifest(t *testing.T) (SignedManifest, ed25519.PublicKey) {
	t.Helper()
	report := Build(Inventory{Records: []SourceRecord{
		{Source: SourceCTRL, SourceVersion: testVersion, Entity: EntityUser, SourceID: "ctrl-user-1", MatchKey: "user-1"},
		{Source: SourceCTRL, SourceVersion: testVersion, Entity: EntityUser, SourceID: "ctrl-user-2", MatchKey: "user-2"},
	}})
	manifest, err := BuildManifest(report)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err = ApproveManifest(manifest, "fixture-operator", "approved for disposable rehearsal", time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := SignManifest(manifest, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	return signed, publicKey
}

func TestBuildManifestBlocksUnresolvedRecords(t *testing.T) {
	report := Build(Inventory{Records: []SourceRecord{
		{Source: SourceCTRL, SourceVersion: testVersion, Entity: EntityUser, SourceID: "ctrl-user-1", MatchKey: "user-1"},
		{Source: SourceCTRL, SourceVersion: testVersion, Entity: EntityWorkspace, SourceID: "ctrl-workspace-1", MatchKey: "workspace-1", ParentSourceID: "ctrl-org-missing", ParentMatchKey: "org-missing"},
	}})
	if _, err := BuildManifest(report); !errors.Is(err, ErrManifestBlocked) {
		t.Fatalf("BuildManifest error = %v, want blocked", err)
	}
}

func TestSignedManifestRequiresApprovalAndRejectsTampering(t *testing.T) {
	report := Build(Inventory{Records: []SourceRecord{{
		Source: SourceCTRL, SourceVersion: testVersion, Entity: EntityUser, SourceID: "ctrl-user-1", MatchKey: "user-1",
	}}})
	manifest, err := BuildManifest(report)
	if err != nil {
		t.Fatal(err)
	}
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SignManifest(manifest, privateKey); !errors.Is(err, ErrManifestApprovalRequired) {
		t.Fatalf("unsigned manifest error = %v, want approval required", err)
	}
	manifest, err = ApproveManifest(manifest, "operator-1", "fixture rehearsal", time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := SignManifest(manifest, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifySignedManifest(signed, publicKey); err != nil {
		t.Fatal(err)
	}
	signed.Manifest.Records[0].PlatformID = "usr_tampered"
	if err := VerifySignedManifest(signed, publicKey); !errors.Is(err, ErrManifestIntegrity) {
		t.Fatalf("tampered manifest error = %v, want integrity failure", err)
	}
}

func TestFixtureImportIsDeterministicIdempotentResumableAndRollbackSafe(t *testing.T) {
	signed, publicKey := signedFixtureManifest(t)
	importer, err := NewFixtureImporter([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := importer.Apply(signed, publicKey, time.Date(2026, 9, 16, 12, 1, 0, 0, time.UTC), 1)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != ImportStatusPaused || first.Applied != 1 || len(importer.Snapshot()) != 1 {
		t.Fatalf("first import = %#v, snapshot = %#v", first, importer.Snapshot())
	}
	second, err := importer.Apply(signed, publicKey, time.Date(2026, 9, 16, 12, 2, 0, 0, time.UTC), 0)
	if err != nil {
		t.Fatal(err)
	}
	if second.Status != ImportStatusCompleted || second.Applied != 1 || len(importer.Snapshot()) != 2 {
		t.Fatalf("resumed import = %#v, snapshot = %#v", second, importer.Snapshot())
	}
	idempotent, err := importer.Apply(signed, publicKey, time.Date(2026, 9, 16, 12, 3, 0, 0, time.UTC), 0)
	if err != nil {
		t.Fatal(err)
	}
	if idempotent.Status != ImportStatusIdempotent || idempotent.Applied != 0 || idempotent.Skipped != 2 {
		t.Fatalf("idempotent import = %#v", idempotent)
	}
	rolledBack, err := importer.Rollback(signed, publicKey, time.Date(2026, 9, 16, 12, 4, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if rolledBack.Status != ImportStatusRolledBack || rolledBack.Skipped != 2 || len(importer.Snapshot()) != 0 {
		t.Fatalf("rollback = %#v, snapshot = %#v", rolledBack, importer.Snapshot())
	}
	if len(importer.Audit()) != 3 {
		t.Fatalf("audit entries = %#v, want two apply and one rollback", importer.Audit())
	}
	secondRollback, err := importer.Rollback(signed, publicKey, time.Date(2026, 9, 16, 12, 5, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if secondRollback.Status != ImportStatusRolledBack || len(importer.Audit()) != 3 {
		t.Fatalf("second rollback = %#v, audit = %#v", secondRollback, importer.Audit())
	}
}

func TestFixtureCheckpointIntegrityFailsClosed(t *testing.T) {
	signed, publicKey := signedFixtureManifest(t)
	importer, err := NewFixtureImporter([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := importer.Apply(signed, publicKey, time.Date(2026, 9, 16, 12, 1, 0, 0, time.UTC), 1); err != nil {
		t.Fatal(err)
	}
	importer.checkpoint.NextIndex++
	if _, err := importer.Apply(signed, publicKey, time.Date(2026, 9, 16, 12, 2, 0, 0, time.UTC), 0); !errors.Is(err, ErrCheckpointIntegrity) {
		t.Fatalf("tampered checkpoint error = %v, want integrity failure", err)
	}
}
