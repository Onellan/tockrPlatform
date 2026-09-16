package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Onellan/tockrplatform/internal/platform/reconciliation"
)

func TestSignAndVerifyFixtureManifestCommand(t *testing.T) {
	report := reconciliation.Build(reconciliation.Inventory{Records: []reconciliation.SourceRecord{{
		Source: reconciliation.SourceCTRL, SourceVersion: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Entity: reconciliation.EntityUser, SourceID: "ctrl-user-1", MatchKey: "user-1",
	}}})
	reportBytes, err := reconciliation.MarshalReport(report)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	reportPath := filepath.Join(directory, "report.json")
	privatePath := filepath.Join(directory, "private.key")
	publicPath := filepath.Join(directory, "public.key")
	manifestPath := filepath.Join(directory, "manifest.json")
	if err := os.WriteFile(reportPath, reportBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(privatePath, []byte(hex.EncodeToString(privateKey)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publicPath, []byte(hex.EncodeToString(publicKey)), 0o600); err != nil {
		t.Fatal(err)
	}
	signed, err := sign(reportPath, privatePath, "fixture-operator", "command acceptance", "2026-09-16T12:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, signed, 0o600); err != nil {
		t.Fatal(err)
	}
	verified, err := verify(manifestPath, publicPath)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]string
	if err := json.Unmarshal(verified, &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "verified" || result["scope"] != reconciliation.FixtureOnlyScope || result["manifest_id"] == "" {
		t.Fatalf("verification result = %#v", result)
	}
}
