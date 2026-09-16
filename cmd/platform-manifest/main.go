// Command platform-manifest creates or verifies a signed fixture-only
// reconciliation manifest. It has no database, network or CTRL/IMS source
// connector and never performs an import itself.
package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Onellan/tockrplatform/internal/platform/reconciliation"
)

func main() {
	operation := flag.String("operation", "sign", "operation: sign or verify")
	reportPath := flag.String("report", "", "inventory report JSON path for sign")
	manifestPath := flag.String("manifest", "", "signed manifest JSON path for verify")
	keyPath := flag.String("key", "", "hex Ed25519 private-key path for sign or public-key path for verify")
	operatorID := flag.String("operator", "", "explicit fixture approval operator ID for sign")
	reason := flag.String("reason", "", "explicit fixture approval reason for sign")
	approvedAt := flag.String("approved-at", "", "RFC3339 approval time for sign")
	outputPath := flag.String("output", "", "output JSON path; stdout when omitted")
	flag.Parse()

	var output []byte
	var err error
	switch strings.ToLower(strings.TrimSpace(*operation)) {
	case "sign":
		output, err = sign(*reportPath, *keyPath, *operatorID, *reason, *approvedAt)
	case "verify":
		output, err = verify(*manifestPath, *keyPath)
	default:
		err = errors.New("operation must be sign or verify")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *outputPath == "" {
		_, err = os.Stdout.Write(output)
	} else {
		err = os.WriteFile(*outputPath, output, 0o600)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func sign(reportPath, keyPath, operatorID, reason, approvedAt string) ([]byte, error) {
	if reportPath == "" || keyPath == "" {
		return nil, errors.New("sign requires -report and -key")
	}
	var report reconciliation.Report
	if err := decodeFile(reportPath, &report); err != nil {
		return nil, fmt.Errorf("decode report: %w", err)
	}
	manifest, err := reconciliation.BuildManifest(report)
	if err != nil {
		return nil, fmt.Errorf("build manifest: %w", err)
	}
	when, err := time.Parse(time.RFC3339Nano, approvedAt)
	if err != nil {
		return nil, fmt.Errorf("parse approved-at: %w", err)
	}
	manifest, err = reconciliation.ApproveManifest(manifest, operatorID, reason, when)
	if err != nil {
		return nil, fmt.Errorf("approve manifest: %w", err)
	}
	key, err := readHexKey(keyPath, ed25519.PrivateKeySize)
	if err != nil {
		return nil, err
	}
	signed, err := reconciliation.SignManifest(manifest, ed25519.PrivateKey(key))
	if err != nil {
		return nil, fmt.Errorf("sign manifest: %w", err)
	}
	return json.MarshalIndent(signed, "", "  ")
}

func verify(manifestPath, keyPath string) ([]byte, error) {
	if manifestPath == "" || keyPath == "" {
		return nil, errors.New("verify requires -manifest and -key")
	}
	var signed reconciliation.SignedManifest
	if err := decodeFile(manifestPath, &signed); err != nil {
		return nil, fmt.Errorf("decode signed manifest: %w", err)
	}
	key, err := readHexKey(keyPath, ed25519.PublicKeySize)
	if err != nil {
		return nil, err
	}
	if err := reconciliation.VerifySignedManifest(signed, ed25519.PublicKey(key)); err != nil {
		return nil, fmt.Errorf("verify manifest: %w", err)
	}
	return json.MarshalIndent(map[string]string{
		"status":      "verified",
		"manifest_id": signed.Manifest.ManifestID,
		"scope":       signed.Manifest.Scope,
	}, "", "  ")
}

func decodeFile(path string, destination any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	return decoder.Decode(destination)
}

func readHexKey(path string, length int) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key: %w", err)
	}
	key, err := hex.DecodeString(strings.TrimSpace(string(data)))
	if err != nil || len(key) != length {
		return nil, fmt.Errorf("key must be %d hex bytes", length)
	}
	return key, nil
}
