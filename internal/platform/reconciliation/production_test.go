package reconciliation

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"strings"
	"testing"
)

func TestProductionManifestIsSeparateSignedBoundary(t *testing.T) {
	reportSHA := strings.Repeat("b", 64)
	manifest, err := NewProductionManifest(reportSHA, []ProductionSourceSnapshot{{Source: SourceCTRL, SourceSHA256: strings.Repeat("a", 40), ReportSHA256: reportSHA}}, []ProductionManifestRecord{{
		Entity: ProductionEntityUser, PlatformID: "usr_imported", MatchKeySHA256: strings.Repeat("c", 64), CreatePolicy: "create_or_reconcile", SourceRefs: []SourceRef{{Source: SourceCTRL, SourceVersion: strings.Repeat("d", 40), SourceID: "ctrl-user-1"}}, Payload: ProductionPayload{User: &UserImportPayload{ID: "usr_imported", Email: "imported@example.test", DisplayName: "Imported", Active: true, CreatedAt: "2026-09-24T12:00:00Z"}},
	}}, ProductionApproval{ApprovalID: "approval-1", OperatorID: "operator-1", ApprovedAt: "2026-09-24T12:00:00Z", Reason: "approved production rehearsal", Scope: ProductionScope, KeyID: "key-1"})
	if err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := SignProductionManifest(manifest, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifySignedProductionManifest(signed, publicKey); err != nil {
		t.Fatal(err)
	}
	if err := ValidateProductionExecution(manifest, ProductionExecution{OperatorID: "operator-1", ApprovalID: "approval-1", KeyID: "key-1"}); err != nil {
		t.Fatal(err)
	}
	signed.Manifest.Records[0].Payload.User.DisplayName = "tampered"
	if err := VerifySignedProductionManifest(signed, publicKey); !errors.Is(err, ErrProductionSignature) && !errors.Is(err, ErrProductionManifestIntegrity) && !errors.Is(err, ErrProductionRecord) {
		t.Fatalf("tampered production manifest error = %v", err)
	}
}
