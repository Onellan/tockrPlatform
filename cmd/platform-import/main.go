// Command platform-import is the protected operator boundary for the
// versioned, signed Platform production reconciliation/import contract.
// It never accepts fixture-only manifests and emits only a redacted receipt.
package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Onellan/tockrplatform/internal/db/sqlite"
	"github.com/Onellan/tockrplatform/internal/platform/reconciliation"
)

func main() {
	operation := flag.String("operation", "dry-run", "operation: dry-run, apply, rollback or status")
	manifestPath := flag.String("manifest", "", "signed production manifest JSON path")
	manifestID := flag.String("manifest-id", "", "manifest ID for status")
	publicKeyPath := flag.String("public-key", "", "hex Ed25519 public-key path")
	secretKeyPath := flag.String("secret-key", "", "hex 32-byte import checkpoint key path")
	databasePath := flag.String("database", "", "Platform SQLite database path")
	operatorID := flag.String("operator", "", "operator ID; must match signed approval")
	approvalID := flag.String("approval", "", "approval ID; must match signed approval")
	keyID := flag.String("key-id", "", "verification key ID; must match signed approval")
	atValue := flag.String("at", "", "RFC3339 execution time")
	limit := flag.Int("limit", 0, "maximum records for apply; zero means all")
	flag.Parse()

	receipt, err := execute(context.Background(), strings.ToLower(strings.TrimSpace(*operation)), *manifestPath, *manifestID, *publicKeyPath, *secretKeyPath, *databasePath, reconciliation.ProductionExecution{OperatorID: *operatorID, ApprovalID: *approvalID, KeyID: *keyID}, *atValue, *limit)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_, _ = os.Stdout.Write(append(encoded, '\n'))
}

func execute(ctx context.Context, operation, manifestPath, manifestID, publicKeyPath, secretKeyPath, databasePath string, execution reconciliation.ProductionExecution, atValue string, limit int) (reconciliation.ProductionImportReceipt, error) {
	if operation == "status" {
		if manifestID == "" || databasePath == "" || secretKeyPath == "" {
			return reconciliation.ProductionImportReceipt{}, errors.New("manifest-id, database and secret-key are required for status")
		}
		secret, err := readHexKey(secretKeyPath, 32)
		if err != nil {
			return reconciliation.ProductionImportReceipt{}, err
		}
		persistence, err := sqlite.OpenWithKey(ctx, databasePath, secret)
		if err != nil {
			return reconciliation.ProductionImportReceipt{}, err
		}
		defer persistence.Close()
		return persistence.ProductionImportStatus(ctx, manifestID)
	}
	if manifestPath == "" || publicKeyPath == "" || databasePath == "" {
		return reconciliation.ProductionImportReceipt{}, errors.New("manifest, public-key and database are required")
	}
	var signed reconciliation.SignedProductionManifest
	if err := decodeFile(manifestPath, &signed); err != nil {
		return reconciliation.ProductionImportReceipt{}, fmt.Errorf("decode manifest: %w", err)
	}
	publicBytes, err := readHexKey(publicKeyPath, ed25519.PublicKeySize)
	if err != nil {
		return reconciliation.ProductionImportReceipt{}, err
	}
	var secret []byte
	if operation != "dry-run" {
		if secretKeyPath == "" {
			return reconciliation.ProductionImportReceipt{}, errors.New("secret-key is required for apply and rollback")
		}
		secret, err = readHexKey(secretKeyPath, 32)
		if err != nil {
			return reconciliation.ProductionImportReceipt{}, err
		}
	}
	var persistence *sqlite.Store
	if operation == "dry-run" {
		persistence, err = sqlite.Open(ctx, databasePath)
	} else {
		persistence, err = sqlite.OpenWithKey(ctx, databasePath, secret)
	}
	if err != nil {
		return reconciliation.ProductionImportReceipt{}, err
	}
	defer persistence.Close()
	switch operation {
	case "dry-run":
		return persistence.DryRunProductionImport(ctx, signed, ed25519.PublicKey(publicBytes), execution)
	case "apply":
		when, err := parseExecutionTime(atValue)
		if err != nil {
			return reconciliation.ProductionImportReceipt{}, err
		}
		return persistence.ApplyProductionImport(ctx, signed, ed25519.PublicKey(publicBytes), execution, when, limit)
	case "rollback":
		when, err := parseExecutionTime(atValue)
		if err != nil {
			return reconciliation.ProductionImportReceipt{}, err
		}
		return persistence.RollbackProductionImport(ctx, signed, ed25519.PublicKey(publicBytes), execution, when)
	default:
		return reconciliation.ProductionImportReceipt{}, errors.New("operation must be dry-run, apply or rollback")
	}
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

func parseExecutionTime(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, errors.New("at is required for apply and rollback")
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse at: %w", err)
	}
	return parsed, nil
}
