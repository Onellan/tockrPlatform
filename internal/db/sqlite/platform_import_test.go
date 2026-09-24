package sqlite

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Onellan/tockrplatform/internal/platform/reconciliation"
)

func productionTestManifest(t *testing.T) (reconciliation.SignedProductionManifest, ed25519.PublicKey, reconciliation.ProductionExecution) {
	t.Helper()
	reportSHA := strings.Repeat("b", 64)
	manifest, err := reconciliation.NewProductionManifest(reportSHA, []reconciliation.ProductionSourceSnapshot{{Source: reconciliation.SourceCTRL, SourceSHA256: strings.Repeat("a", 40), ReportSHA256: reportSHA}}, []reconciliation.ProductionManifestRecord{{
		Entity: reconciliation.ProductionEntityUser, PlatformID: "usr_imported", MatchKeySHA256: strings.Repeat("c", 64), CreatePolicy: "create_or_reconcile", SourceRefs: []reconciliation.SourceRef{{Source: reconciliation.SourceCTRL, SourceVersion: strings.Repeat("d", 40), SourceID: "ctrl-user-1"}}, Payload: reconciliation.ProductionPayload{User: &reconciliation.UserImportPayload{ID: "usr_imported", Email: "imported@example.test", DisplayName: "Imported", Active: true, CreatedAt: "2026-09-24T12:00:00Z"}},
	}}, reconciliation.ProductionApproval{ApprovalID: "approval-1", OperatorID: "operator-1", ApprovedAt: "2026-09-24T12:00:00Z", Reason: "approved production rehearsal", Scope: reconciliation.ProductionScope, KeyID: "key-1"})
	if err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := reconciliation.SignProductionManifest(manifest, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	return signed, publicKey, reconciliation.ProductionExecution{OperatorID: "operator-1", ApprovalID: "approval-1", KeyID: "key-1"}
}

func TestProductionImportIsDurableIdempotentAndCompensatesOnlyCreatedRows(t *testing.T) {
	ctx := context.Background()
	store, err := OpenWithKey(ctx, ":memory:", []byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	signed, publicKey, execution := productionTestManifest(t)
	at := time.Date(2026, 9, 24, 12, 1, 0, 0, time.UTC)
	dryRun, err := store.DryRunProductionImport(ctx, signed, publicKey, execution)
	if err != nil || dryRun.Status != reconciliation.ProductionImportDryRun {
		t.Fatalf("dry run = %#v, err=%v", dryRun, err)
	}
	first, err := store.ApplyProductionImport(ctx, signed, publicKey, execution, at, 1)
	if err != nil || first.Status != reconciliation.ProductionImportCompleted || first.Created != 1 {
		t.Fatalf("first import = %#v, err=%v", first, err)
	}
	second, err := store.ApplyProductionImport(ctx, signed, publicKey, execution, at.Add(time.Minute), 0)
	if err != nil || second.Status != reconciliation.ProductionImportIdempotent || second.Skipped != 1 {
		t.Fatalf("idempotent import = %#v, err=%v", second, err)
	}
	var count int
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE public_id='usr_imported'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("imported user count = %d, err=%v", count, err)
	}
	rolledBack, err := store.RollbackProductionImport(ctx, signed, publicKey, execution, at.Add(2*time.Minute))
	if err != nil || rolledBack.Status != reconciliation.ProductionImportRolledBack {
		t.Fatalf("rollback = %#v, err=%v", rolledBack, err)
	}
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE public_id='usr_imported'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("rolled-back user count = %d, err=%v", count, err)
	}
	var outbox int
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM platform_outbox WHERE aggregate_id='usr_imported'`).Scan(&outbox); err != nil || outbox != 1 {
		t.Fatalf("outbox history count = %d, err=%v", outbox, err)
	}
}

func TestProductionImportRequiresMatchingExecutionAuthorization(t *testing.T) {
	ctx := context.Background()
	store, err := OpenWithKey(ctx, ":memory:", []byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	signed, publicKey, execution := productionTestManifest(t)
	execution.OperatorID = "wrong-operator"
	if _, err := store.DryRunProductionImport(ctx, signed, publicKey, execution); !errors.Is(err, reconciliation.ErrProductionExecution) {
		t.Fatalf("execution error = %v, want authorization failure", err)
	}
}
