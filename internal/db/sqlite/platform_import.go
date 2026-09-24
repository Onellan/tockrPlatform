package sqlite

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Onellan/tockrplatform/internal/auth"
	"github.com/Onellan/tockrplatform/internal/events"
	"github.com/Onellan/tockrplatform/internal/platform/reconciliation"
	"github.com/Onellan/tockrplatform/internal/store"
)

var (
	ErrProductionImportSecretRequired = errors.New("production import requires an OpenWithKey store")
	ErrProductionImportNotFound       = errors.New("production import run not found")
	ErrProductionImportConflict       = errors.New("production import manifest conflicts with an existing run")
	ErrProductionImportRecord         = errors.New("production import record conflicts with canonical state")
	ErrProductionImportRollback       = errors.New("production import rollback is blocked")
	ErrProductionImportCheckpoint     = errors.New("production import checkpoint integrity is invalid")
)

type productionImportOutcome struct {
	Outcome     string
	Created     bool
	CanonicalID string
}

// DryRunProductionImport verifies the signed production boundary and checks
// the manifest's dependency ordering without writing the database.
func (s *Store) DryRunProductionImport(ctx context.Context, signed reconciliation.SignedProductionManifest, publicKey ed25519.PublicKey, execution reconciliation.ProductionExecution) (reconciliation.ProductionImportReceipt, error) {
	if err := s.validateProductionBoundary(signed, publicKey, execution); err != nil {
		return reconciliation.ProductionImportReceipt{}, err
	}
	if err := s.validateProductionDependencies(ctx, signed.Manifest); err != nil {
		return reconciliation.ProductionImportReceipt{}, err
	}
	return reconciliation.ProductionImportReceipt{
		ManifestID: signed.Manifest.ManifestID, ManifestSHA256: reconciliation.ProductionManifestDigest(signed.Manifest),
		SourceReportSHA256: signed.Manifest.SourceReportSHA256, OperatorID: execution.OperatorID,
		ApprovalID: execution.ApprovalID, KeyID: execution.KeyID, Status: reconciliation.ProductionImportDryRun,
		NextIndex: len(signed.Manifest.Records), Skipped: len(signed.Manifest.Records), OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
	}, nil
}

// ProductionImportStatus returns the last durable, redacted receipt after
// verifying the checkpoint MAC. It does not expose manifest payloads or keys.
func (s *Store) ProductionImportStatus(ctx context.Context, manifestID string) (reconciliation.ProductionImportReceipt, error) {
	if len(s.secretKey) < 32 {
		return reconciliation.ProductionImportReceipt{}, ErrProductionImportSecretRequired
	}
	var digest, sourceReport, status, mac, receiptJSON string
	var next, applied, created, reconciled, conflicts int
	err := s.db.QueryRowContext(ctx, `SELECT manifest_digest,source_report_sha256,status,next_index,applied_count,created_count,reconciled_count,conflict_count,checkpoint_mac,receipt_json FROM platform_import_runs WHERE manifest_id=?`, strings.TrimSpace(manifestID)).Scan(&digest, &sourceReport, &status, &next, &applied, &created, &reconciled, &conflicts, &mac, &receiptJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return reconciliation.ProductionImportReceipt{}, ErrProductionImportNotFound
	}
	if err != nil {
		return reconciliation.ProductionImportReceipt{}, err
	}
	if !hmac.Equal([]byte(mac), []byte(s.productionCheckpointMAC(manifestID, digest, status, next, applied, created, reconciled, conflicts))) {
		return reconciliation.ProductionImportReceipt{}, ErrProductionImportCheckpoint
	}
	var receipt reconciliation.ProductionImportReceipt
	if err := json.Unmarshal([]byte(receiptJSON), &receipt); err != nil {
		return reconciliation.ProductionImportReceipt{}, ErrProductionImportCheckpoint
	}
	if receipt.ManifestID != manifestID || receipt.ManifestSHA256 != digest || receipt.SourceReportSHA256 != sourceReport {
		return reconciliation.ProductionImportReceipt{}, ErrProductionImportCheckpoint
	}
	return receipt, nil
}

// ApplyProductionImport applies a bounded number of records. Each record and
// its durable checkpoint are committed together, so process failure resumes
// from the last committed record without replaying canonical writes.
func (s *Store) ApplyProductionImport(ctx context.Context, signed reconciliation.SignedProductionManifest, publicKey ed25519.PublicKey, execution reconciliation.ProductionExecution, at time.Time, limit int) (reconciliation.ProductionImportReceipt, error) {
	if at.IsZero() {
		return reconciliation.ProductionImportReceipt{}, reconciliation.ErrInvalidExecutionTime
	}
	if err := s.validateProductionBoundary(signed, publicKey, execution); err != nil {
		return reconciliation.ProductionImportReceipt{}, err
	}
	if err := s.validateProductionDependencies(ctx, signed.Manifest); err != nil {
		return reconciliation.ProductionImportReceipt{}, err
	}
	if len(s.secretKey) < 32 {
		return reconciliation.ProductionImportReceipt{}, ErrProductionImportSecretRequired
	}

	run, err := s.loadOrCreateProductionRun(ctx, signed.Manifest, execution, at)
	if err != nil {
		return reconciliation.ProductionImportReceipt{}, err
	}
	if run.Status == "rolled_back" {
		return reconciliation.ProductionImportReceipt{}, reconciliation.ErrImportAlreadyReversed
	}
	if run.Status == "completed" {
		run.Status = reconciliation.ProductionImportIdempotent
		run.Skipped = len(signed.Manifest.Records)
		run.OccurredAt = at.UTC().Format(time.RFC3339Nano)
		return productionReceiptFromRun(run, signed.Manifest.SourceReportSHA256), nil
	}
	if limit <= 0 || limit > len(signed.Manifest.Records)-run.NextIndex {
		limit = len(signed.Manifest.Records) - run.NextIndex
	}
	for count := 0; count < limit && run.NextIndex < len(signed.Manifest.Records); count++ {
		if err := s.applyProductionRecord(ctx, signed.Manifest, execution, at, run.NextIndex); err != nil {
			return reconciliation.ProductionImportReceipt{}, err
		}
		record := signed.Manifest.Records[run.NextIndex]
		outcome, created, err := s.productionRecordOutcome(ctx, signed.Manifest.ManifestID, run.NextIndex)
		if err != nil {
			return reconciliation.ProductionImportReceipt{}, err
		}
		run.NextIndex++
		run.Applied++
		if created {
			run.Created++
		} else {
			run.Reconciled++
		}
		if outcome == "reconciled" {
			run.Skipped++
		}
		run.LastPlatformID = record.PlatformID
	}
	if run.NextIndex == len(signed.Manifest.Records) {
		run.Status = reconciliation.ProductionImportCompleted
	} else {
		run.Status = reconciliation.ProductionImportPaused
	}
	run.OccurredAt = at.UTC().Format(time.RFC3339Nano)
	return s.persistProductionRunReceipt(ctx, signed.Manifest, execution, run, at)
}

// RollbackProductionImport compensates only records created by this manifest,
// in reverse dependency order. Audit, outbox and import-ledger history remain.
func (s *Store) RollbackProductionImport(ctx context.Context, signed reconciliation.SignedProductionManifest, publicKey ed25519.PublicKey, execution reconciliation.ProductionExecution, at time.Time) (reconciliation.ProductionImportReceipt, error) {
	if at.IsZero() {
		return reconciliation.ProductionImportReceipt{}, reconciliation.ErrInvalidExecutionTime
	}
	if err := s.validateProductionBoundary(signed, publicKey, execution); err != nil {
		return reconciliation.ProductionImportReceipt{}, err
	}
	if len(s.secretKey) < 32 {
		return reconciliation.ProductionImportReceipt{}, ErrProductionImportSecretRequired
	}
	run, err := s.loadProductionRun(ctx, signed.Manifest, execution)
	if err != nil {
		return reconciliation.ProductionImportReceipt{}, err
	}
	if run.Status == "rolled_back" {
		run.Status = reconciliation.ProductionImportRolledBack
		run.OccurredAt = at.UTC().Format(time.RFC3339Nano)
		return productionReceiptFromRun(run, signed.Manifest.SourceReportSHA256), nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT record_order,entity_kind,platform_id FROM platform_import_records WHERE manifest_id=? AND created_by_import=1 AND compensated=0 ORDER BY record_order DESC`, signed.Manifest.ManifestID)
	if err != nil {
		return reconciliation.ProductionImportReceipt{}, fmt.Errorf("list production compensation records: %w", err)
	}
	type compensation struct {
		order      int
		entity, id string
	}
	items := make([]compensation, 0)
	for rows.Next() {
		var item compensation
		if err := rows.Scan(&item.order, &item.entity, &item.id); err != nil {
			rows.Close()
			return reconciliation.ProductionImportReceipt{}, err
		}
		items = append(items, item)
	}
	if err := rows.Close(); err != nil {
		return reconciliation.ProductionImportReceipt{}, err
	}
	for _, item := range items {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return reconciliation.ProductionImportReceipt{}, err
		}
		if err := deleteImportedRecordTx(ctx, tx, item.entity, item.id); err != nil {
			_ = tx.Rollback()
			return reconciliation.ProductionImportReceipt{}, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE platform_import_records SET compensated=1,outcome='rolled_back' WHERE manifest_id=? AND record_order=? AND created_by_import=1 AND compensated=0`, signed.Manifest.ManifestID, item.order); err != nil {
			_ = tx.Rollback()
			return reconciliation.ProductionImportReceipt{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO platform_import_compensations(manifest_id,record_order,entity_kind,platform_id,status,occurred_at) VALUES(?,?,?,?,'compensated',?)`, signed.Manifest.ManifestID, item.order, item.entity, item.id, formatTime(at.UTC())); err != nil {
			_ = tx.Rollback()
			return reconciliation.ProductionImportReceipt{}, err
		}
		if err := tx.Commit(); err != nil {
			return reconciliation.ProductionImportReceipt{}, err
		}
	}
	run.Status = "rolled_back"
	run.OccurredAt = at.UTC().Format(time.RFC3339Nano)
	return s.persistProductionRunReceipt(ctx, signed.Manifest, execution, run, at)
}

func (s *Store) validateProductionBoundary(signed reconciliation.SignedProductionManifest, publicKey ed25519.PublicKey, execution reconciliation.ProductionExecution) error {
	if err := reconciliation.VerifySignedProductionManifest(signed, publicKey); err != nil {
		return err
	}
	return reconciliation.ValidateProductionExecution(signed.Manifest, execution)
}

func (s *Store) validateProductionDependencies(ctx context.Context, manifest reconciliation.ProductionManifest) error {
	seen := make(map[string]struct{}, len(manifest.Records))
	for _, record := range manifest.Records {
		if _, exists := seen[record.PlatformID]; exists {
			return reconciliation.ErrProductionRecord
		}
		seen[record.PlatformID] = struct{}{}
		if record.Entity == reconciliation.ProductionEntityWorkspace {
			if record.Payload.Workspace == nil {
				return reconciliation.ErrProductionRecord
			}
			if _, exists := seen[record.Payload.Workspace.OrganisationID]; !exists {
				return fmt.Errorf("%w: workspace parent is not ordered before child", reconciliation.ErrProductionRecord)
			}
		}
		if record.Entity == reconciliation.ProductionEntityOrganisationMembership {
			if record.Payload.OrganisationMembership == nil {
				return reconciliation.ErrProductionRecord
			}
			p := record.Payload.OrganisationMembership
			if _, exists := seen[p.OrganisationID]; !exists {
				return fmt.Errorf("%w: organisation membership parent is not ordered before child", reconciliation.ErrProductionRecord)
			}
			if _, exists := seen[p.UserID]; !exists {
				return fmt.Errorf("%w: membership user is not ordered before child", reconciliation.ErrProductionRecord)
			}
		}
		if record.Entity == reconciliation.ProductionEntityWorkspaceMembership {
			if record.Payload.WorkspaceMembership == nil {
				return reconciliation.ErrProductionRecord
			}
			p := record.Payload.WorkspaceMembership
			if _, exists := seen[p.WorkspaceID]; !exists {
				return fmt.Errorf("%w: workspace membership parent is not ordered before child", reconciliation.ErrProductionRecord)
			}
			if _, exists := seen[p.UserID]; !exists {
				return fmt.Errorf("%w: membership user is not ordered before child", reconciliation.ErrProductionRecord)
			}
		}
		if record.Entity == reconciliation.ProductionEntityOrganisationEntitlement {
			if record.Payload.OrganisationEntitlement == nil {
				return reconciliation.ErrProductionRecord
			}
			p := record.Payload.OrganisationEntitlement
			if _, exists := seen[p.OrganisationID]; !exists {
				return fmt.Errorf("%w: entitlement organisation is not ordered before child", reconciliation.ErrProductionRecord)
			}
			if _, exists := seen[p.ProductKey]; !exists {
				return fmt.Errorf("%w: entitlement product is not ordered before child", reconciliation.ErrProductionRecord)
			}
		}
		if record.Entity == reconciliation.ProductionEntityUserAssignment {
			if record.Payload.UserAssignment == nil {
				return reconciliation.ErrProductionRecord
			}
			p := record.Payload.UserAssignment
			if _, exists := seen[p.UserID]; !exists {
				return fmt.Errorf("%w: assignment user is not ordered before child", reconciliation.ErrProductionRecord)
			}
			if _, exists := seen[p.OrganisationID]; !exists {
				return fmt.Errorf("%w: assignment organisation is not ordered before child", reconciliation.ErrProductionRecord)
			}
			if _, exists := seen[p.ProductKey]; !exists {
				return fmt.Errorf("%w: assignment product is not ordered before child", reconciliation.ErrProductionRecord)
			}
		}
	}
	return nil
}

type productionRunState struct {
	ManifestID, ManifestDigest, SourceReportSHA256, ApprovalID, OperatorID, KeyID, ApprovedAt string
	Status                                                                                    string
	NextIndex, Applied, Skipped, Created, Reconciled, Conflicts                               int
	LastPlatformID                                                                            string
	OccurredAt                                                                                string
}

func (s *Store) loadOrCreateProductionRun(ctx context.Context, manifest reconciliation.ProductionManifest, execution reconciliation.ProductionExecution, at time.Time) (productionRunState, error) {
	run, err := s.loadProductionRun(ctx, manifest, execution)
	if err == nil {
		return run, nil
	}
	if !errors.Is(err, ErrProductionImportNotFound) {
		return productionRunState{}, err
	}
	now := formatTime(at.UTC())
	digest := reconciliation.ProductionManifestDigest(manifest)
	snapshots, _ := json.Marshal(manifest.SourceSnapshots)
	receipt := reconciliation.ProductionImportReceipt{ManifestID: manifest.ManifestID, ManifestSHA256: digest, SourceReportSHA256: manifest.SourceReportSHA256, OperatorID: execution.OperatorID, ApprovalID: execution.ApprovalID, KeyID: execution.KeyID, Status: "running", OccurredAt: now}
	receiptJSON, _ := json.Marshal(receipt)
	mac := s.productionCheckpointMAC(manifest.ManifestID, digest, "running", 0, 0, 0, 0, 0)
	if _, err := s.db.ExecContext(ctx, `INSERT INTO platform_import_runs(manifest_id,manifest_digest,contract_version,source_report_sha256,source_snapshots_json,approval_id,operator_id,key_id,approved_at,status,next_index,applied_count,created_count,reconciled_count,conflict_count,checkpoint_mac,created_at,updated_at,receipt_json) VALUES(?,?,?, ?,?,?,?,?,?,'running',0,0,0,0,0,?,?,?,?)`, manifest.ManifestID, digest, reconciliation.ProductionManifestType, manifest.SourceReportSHA256, string(snapshots), execution.ApprovalID, execution.OperatorID, execution.KeyID, manifest.Approval.ApprovedAt, mac, now, now, string(receiptJSON)); err != nil {
		return productionRunState{}, fmt.Errorf("create production import run: %w", err)
	}
	return productionRunState{ManifestID: manifest.ManifestID, ManifestDigest: digest, SourceReportSHA256: manifest.SourceReportSHA256, ApprovalID: execution.ApprovalID, OperatorID: execution.OperatorID, KeyID: execution.KeyID, ApprovedAt: manifest.Approval.ApprovedAt, Status: "running"}, nil
}

func (s *Store) loadProductionRun(ctx context.Context, manifest reconciliation.ProductionManifest, execution reconciliation.ProductionExecution) (productionRunState, error) {
	var run productionRunState
	var mac string
	err := s.db.QueryRowContext(ctx, `SELECT manifest_id,manifest_digest,source_report_sha256,approval_id,operator_id,key_id,approved_at,status,next_index,applied_count,created_count,reconciled_count,conflict_count,checkpoint_mac,updated_at FROM platform_import_runs WHERE manifest_id=?`, manifest.ManifestID).
		Scan(&run.ManifestID, &run.ManifestDigest, &run.SourceReportSHA256, &run.ApprovalID, &run.OperatorID, &run.KeyID, &run.ApprovedAt, &run.Status, &run.NextIndex, &run.Applied, &run.Created, &run.Reconciled, &run.Conflicts, &mac, &run.OccurredAt)
	if errors.Is(err, sql.ErrNoRows) {
		return productionRunState{}, ErrProductionImportNotFound
	}
	if err != nil {
		return productionRunState{}, fmt.Errorf("load production import run: %w", err)
	}
	if run.ManifestDigest != reconciliation.ProductionManifestDigest(manifest) || run.SourceReportSHA256 != manifest.SourceReportSHA256 || run.ApprovalID != execution.ApprovalID || run.OperatorID != execution.OperatorID || run.KeyID != execution.KeyID || !hmac.Equal([]byte(mac), []byte(s.productionCheckpointMAC(run.ManifestID, run.ManifestDigest, run.Status, run.NextIndex, run.Applied, run.Created, run.Reconciled, run.Conflicts))) {
		return productionRunState{}, ErrProductionImportCheckpoint
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM platform_import_records WHERE manifest_id=? AND outcome='reconciled'`, manifest.ManifestID).Scan(&run.Skipped); err != nil {
		return productionRunState{}, fmt.Errorf("count reconciled production import records: %w", err)
	}
	return run, nil
}

func (s *Store) applyProductionRecord(ctx context.Context, manifest reconciliation.ProductionManifest, execution reconciliation.ProductionExecution, at time.Time, index int) error {
	record := manifest.Records[index]
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin production record: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	outcome, err := applyProductionRecordTx(ctx, tx, record)
	if err != nil {
		return err
	}
	recordJSON, _ := json.Marshal(record)
	refsJSON, _ := json.Marshal(record.SourceRefs)
	digest := digestBytes(recordJSON)
	if _, err := tx.ExecContext(ctx, `INSERT INTO platform_import_records(manifest_id,record_order,entity_kind,platform_id,record_digest,source_refs_json,outcome,created_by_import,applied_at) VALUES(?,?,?,?,?,?,?, ?,?)`, manifest.ManifestID, index, string(record.Entity), record.PlatformID, digest, string(refsJSON), outcome.Outcome, boolInt(outcome.Created), formatTime(at.UTC())); err != nil {
		return fmt.Errorf("record production import outcome: %w", err)
	}
	run, err := loadProductionRunTx(ctx, tx, manifest.ManifestID)
	if err != nil {
		return err
	}
	run.NextIndex = index + 1
	run.Applied++
	if outcome.Created {
		run.Created++
	} else {
		run.Reconciled++
	}
	run.LastPlatformID = record.PlatformID
	if run.NextIndex == len(manifest.Records) {
		run.Status = reconciliation.ProductionImportCompleted
	} else {
		run.Status = reconciliation.ProductionImportPaused
	}
	receipt := reconciliation.ProductionImportReceipt{ManifestID: manifest.ManifestID, ManifestSHA256: run.ManifestDigest, SourceReportSHA256: manifest.SourceReportSHA256, OperatorID: execution.OperatorID, ApprovalID: execution.ApprovalID, KeyID: execution.KeyID, Status: run.Status, NextIndex: run.NextIndex, Applied: run.Applied, Created: run.Created, Reconciled: run.Reconciled, Conflicts: run.Conflicts, OccurredAt: formatTime(at.UTC())}
	receiptJSON, _ := json.Marshal(receipt)
	mac := s.productionCheckpointMAC(run.ManifestID, run.ManifestDigest, run.Status, run.NextIndex, run.Applied, run.Created, run.Reconciled, run.Conflicts)
	if _, err := tx.ExecContext(ctx, `UPDATE platform_import_runs SET status=?,next_index=?,applied_count=?,created_count=?,reconciled_count=?,conflict_count=?,checkpoint_mac=?,updated_at=?,receipt_json=? WHERE manifest_id=?`, run.Status, run.NextIndex, run.Applied, run.Created, run.Reconciled, run.Conflicts, mac, formatTime(at.UTC()), string(receiptJSON), manifest.ManifestID); err != nil {
		return fmt.Errorf("update production import checkpoint: %w", err)
	}
	return tx.Commit()
}

func (s *Store) productionRecordOutcome(ctx context.Context, manifestID string, index int) (string, bool, error) {
	var outcome string
	var created int
	err := s.db.QueryRowContext(ctx, `SELECT outcome,created_by_import FROM platform_import_records WHERE manifest_id=? AND record_order=?`, manifestID, index).Scan(&outcome, &created)
	return outcome, created == 1, err
}

func loadProductionRunTx(ctx context.Context, tx *sql.Tx, manifestID string) (productionRunState, error) {
	var run productionRunState
	err := tx.QueryRowContext(ctx, `SELECT manifest_id,manifest_digest,source_report_sha256,approval_id,operator_id,key_id,approved_at,status,next_index,applied_count,created_count,reconciled_count,conflict_count,updated_at FROM platform_import_runs WHERE manifest_id=?`, manifestID).
		Scan(&run.ManifestID, &run.ManifestDigest, &run.SourceReportSHA256, &run.ApprovalID, &run.OperatorID, &run.KeyID, &run.ApprovedAt, &run.Status, &run.NextIndex, &run.Applied, &run.Created, &run.Reconciled, &run.Conflicts, &run.OccurredAt)
	if err != nil {
		return productionRunState{}, fmt.Errorf("load production import run in transaction: %w", err)
	}
	return run, nil
}

func (s *Store) persistProductionRunReceipt(ctx context.Context, manifest reconciliation.ProductionManifest, execution reconciliation.ProductionExecution, run productionRunState, at time.Time) (reconciliation.ProductionImportReceipt, error) {
	receipt := reconciliation.ProductionImportReceipt{ManifestID: manifest.ManifestID, ManifestSHA256: run.ManifestDigest, SourceReportSHA256: manifest.SourceReportSHA256, OperatorID: execution.OperatorID, ApprovalID: execution.ApprovalID, KeyID: execution.KeyID, Status: run.Status, NextIndex: run.NextIndex, Applied: run.Applied, Skipped: run.Skipped, Created: run.Created, Reconciled: run.Reconciled, Conflicts: run.Conflicts, OccurredAt: at.UTC().Format(time.RFC3339Nano)}
	receiptJSON, err := json.Marshal(receipt)
	if err != nil {
		return reconciliation.ProductionImportReceipt{}, err
	}
	mac := s.productionCheckpointMAC(run.ManifestID, run.ManifestDigest, run.Status, run.NextIndex, run.Applied, run.Created, run.Reconciled, run.Conflicts)
	if _, err := s.db.ExecContext(ctx, `UPDATE platform_import_runs SET status=?,next_index=?,applied_count=?,created_count=?,reconciled_count=?,conflict_count=?,checkpoint_mac=?,updated_at=?,receipt_json=? WHERE manifest_id=?`, run.Status, run.NextIndex, run.Applied, run.Created, run.Reconciled, run.Conflicts, mac, formatTime(at.UTC()), string(receiptJSON), manifest.ManifestID); err != nil {
		return reconciliation.ProductionImportReceipt{}, fmt.Errorf("persist production import receipt: %w", err)
	}
	return receipt, nil
}

func productionReceiptFromRun(run productionRunState, sourceReport string) reconciliation.ProductionImportReceipt {
	return reconciliation.ProductionImportReceipt{ManifestID: run.ManifestID, ManifestSHA256: run.ManifestDigest, SourceReportSHA256: sourceReport, OperatorID: run.OperatorID, ApprovalID: run.ApprovalID, KeyID: run.KeyID, Status: run.Status, NextIndex: run.NextIndex, Applied: run.Applied, Skipped: run.Skipped, Created: run.Created, Reconciled: run.Reconciled, Conflicts: run.Conflicts, OccurredAt: run.OccurredAt}
}

func (s *Store) productionCheckpointMAC(manifestID, digest, status string, next, applied, created, reconciled, conflicts int) string {
	mac := hmac.New(sha256.New, s.secretKey)
	_, _ = fmt.Fprintf(mac, "%s\x00%s\x00%s\x00%d\x00%d\x00%d\x00%d\x00%d", manifestID, digest, status, next, applied, created, reconciled, conflicts)
	return hex.EncodeToString(mac.Sum(nil))
}

func applyProductionRecordTx(ctx context.Context, tx *sql.Tx, record reconciliation.ProductionManifestRecord) (productionImportOutcome, error) {
	switch record.Entity {
	case reconciliation.ProductionEntityUser:
		return applyImportedUserTx(ctx, tx, record)
	case reconciliation.ProductionEntityOrganisation:
		return applyImportedOrganisationTx(ctx, tx, record)
	case reconciliation.ProductionEntityWorkspace:
		return applyImportedWorkspaceTx(ctx, tx, record)
	case reconciliation.ProductionEntityOrganisationMembership:
		return applyImportedOrganisationMembershipTx(ctx, tx, record)
	case reconciliation.ProductionEntityWorkspaceMembership:
		return applyImportedWorkspaceMembershipTx(ctx, tx, record)
	case reconciliation.ProductionEntityProduct:
		return applyImportedProductTx(ctx, tx, record)
	case reconciliation.ProductionEntityOrganisationEntitlement:
		return applyImportedEntitlementTx(ctx, tx, record)
	case reconciliation.ProductionEntityUserAssignment:
		return applyImportedAssignmentTx(ctx, tx, record)
	default:
		return productionImportOutcome{}, reconciliation.ErrProductionRecord
	}
}

func applyImportedUserTx(ctx context.Context, tx *sql.Tx, record reconciliation.ProductionManifestRecord) (productionImportOutcome, error) {
	p := record.Payload.User
	if p == nil || p.ID != record.PlatformID || !strings.HasPrefix(p.ID, "usr_") || strings.TrimSpace(p.Email) == "" || strings.TrimSpace(p.DisplayName) == "" {
		return productionImportOutcome{}, reconciliation.ErrProductionRecord
	}
	if _, err := parseImportTimeValue(p.CreatedAt); err != nil {
		return productionImportOutcome{}, err
	}
	var id int64
	var email, name, created string
	var active int
	err := tx.QueryRowContext(ctx, `SELECT id,email,display_name,active,created_at FROM users WHERE public_id=?`, p.ID).Scan(&id, &email, &name, &active, &created)
	if errors.Is(err, sql.ErrNoRows) {
		if record.CreatePolicy == "reconcile_only" {
			return productionImportOutcome{}, ErrProductionImportRecord
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO users(public_id,email,display_name,password_hash,active,created_at) VALUES(?,?,?,?,?,?)`, p.ID, strings.ToLower(strings.TrimSpace(p.Email)), strings.TrimSpace(p.DisplayName), auth.DummyPasswordHash, boolInt(p.Active), p.CreatedAt); err != nil {
			return productionImportOutcome{}, fmt.Errorf("create imported user: %w", err)
		}
		if err := appendPlatformEventTx(ctx, tx, events.EventUserCreated, p.ID, parseImportTime(p.CreatedAt), map[string]any{"user_id": p.ID, "active": p.Active}); err != nil {
			return productionImportOutcome{}, err
		}
		return productionImportOutcome{Outcome: "created", Created: true, CanonicalID: p.ID}, nil
	}
	if err != nil {
		return productionImportOutcome{}, err
	}
	if email != strings.ToLower(strings.TrimSpace(p.Email)) || name != strings.TrimSpace(p.DisplayName) || active != boolInt(p.Active) || created != p.CreatedAt {
		return productionImportOutcome{}, ErrProductionImportRecord
	}
	return productionImportOutcome{Outcome: "reconciled", CanonicalID: p.ID}, nil
}

func applyImportedOrganisationTx(ctx context.Context, tx *sql.Tx, record reconciliation.ProductionManifestRecord) (productionImportOutcome, error) {
	p := record.Payload.Organisation
	if p == nil || p.ID != record.PlatformID {
		return productionImportOutcome{}, reconciliation.ErrProductionRecord
	}
	if _, err := parseImportTimeValue(p.CreatedAt); err != nil {
		return productionImportOutcome{}, err
	}
	var currentName, currentStatus, currentCreated string
	var currentArchived sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT name,status,created_at,archived_at FROM organisations WHERE public_id=?`, p.ID).Scan(&currentName, &currentStatus, &currentCreated, &currentArchived)
	if errors.Is(err, sql.ErrNoRows) {
		if record.CreatePolicy == "reconcile_only" {
			return productionImportOutcome{}, ErrProductionImportRecord
		}
		actor, err := importActorTx(ctx, tx, p.History.ActorUserID)
		if err != nil {
			return productionImportOutcome{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO organisations(public_id,name,status,created_at,archived_at) VALUES(?,?,?,?,?)`, p.ID, strings.TrimSpace(p.Name), p.Status, p.CreatedAt, nullableString(p.ArchivedAt)); err != nil {
			return productionImportOutcome{}, err
		}
		if err := recordOrganisationAuditTx(ctx, tx, actor, p.ID, eventOrganisationCreated, organisationMutationDetails{OrganisationID: p.ID, Reason: p.History.Reason}, parseImportTime(p.CreatedAt)); err != nil {
			return productionImportOutcome{}, err
		}
		return productionImportOutcome{Outcome: "created", Created: true, CanonicalID: p.ID}, nil
	}
	if err != nil {
		return productionImportOutcome{}, err
	}
	if currentName != p.Name || currentStatus != p.Status || currentCreated != p.CreatedAt || nullableValue(currentArchived) != optionalStringValue(p.ArchivedAt) {
		return productionImportOutcome{}, ErrProductionImportRecord
	}
	return productionImportOutcome{Outcome: "reconciled", CanonicalID: p.ID}, nil
}

func applyImportedWorkspaceTx(ctx context.Context, tx *sql.Tx, record reconciliation.ProductionManifestRecord) (productionImportOutcome, error) {
	p := record.Payload.Workspace
	if p == nil || p.ID != record.PlatformID {
		return productionImportOutcome{}, reconciliation.ErrProductionRecord
	}
	if _, err := parseImportTimeValue(p.CreatedAt); err != nil {
		return productionImportOutcome{}, err
	}
	var currentOrg, currentName, currentStatus, currentCreated string
	var currentArchived sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT o.public_id,w.name,w.status,w.created_at,w.archived_at FROM workspaces w JOIN organisations o ON o.id=w.organisation_id WHERE w.public_id=?`, p.ID).Scan(&currentOrg, &currentName, &currentStatus, &currentCreated, &currentArchived)
	if errors.Is(err, sql.ErrNoRows) {
		if record.CreatePolicy == "reconcile_only" {
			return productionImportOutcome{}, ErrProductionImportRecord
		}
		actor, err := importActorTx(ctx, tx, p.History.ActorUserID)
		if err != nil {
			return productionImportOutcome{}, err
		}
		var organisationID int64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM organisations WHERE public_id=?`, p.OrganisationID).Scan(&organisationID); err != nil {
			return productionImportOutcome{}, fmt.Errorf("resolve imported workspace organisation: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO workspaces(public_id,organisation_id,name,status,created_at,archived_at) VALUES(?,?,?,?,?,?)`, p.ID, organisationID, strings.TrimSpace(p.Name), p.Status, p.CreatedAt, nullableString(p.ArchivedAt)); err != nil {
			return productionImportOutcome{}, err
		}
		if err := recordWorkspaceAuditTx(ctx, tx, actor, p.ID, eventWorkspaceCreated, workspaceMutationDetails{WorkspaceID: p.ID, Reason: p.History.Reason}, parseImportTime(p.CreatedAt)); err != nil {
			return productionImportOutcome{}, err
		}
		return productionImportOutcome{Outcome: "created", Created: true, CanonicalID: p.ID}, nil
	}
	if err != nil {
		return productionImportOutcome{}, err
	}
	if currentOrg != p.OrganisationID || currentName != p.Name || currentStatus != p.Status || currentCreated != p.CreatedAt || nullableValue(currentArchived) != optionalStringValue(p.ArchivedAt) {
		return productionImportOutcome{}, ErrProductionImportRecord
	}
	return productionImportOutcome{Outcome: "reconciled", CanonicalID: p.ID}, nil
}

func applyImportedOrganisationMembershipTx(ctx context.Context, tx *sql.Tx, record reconciliation.ProductionManifestRecord) (productionImportOutcome, error) {
	p := record.Payload.OrganisationMembership
	if p == nil || p.ID != record.PlatformID || p.OrganisationID == "" || p.UserID == "" {
		return productionImportOutcome{}, reconciliation.ErrProductionRecord
	}
	return applyImportedMembershipTx(ctx, tx, record, p.ID, p.OrganisationID, "", p.UserID, p.Role, p.Active, p.AssignedBy, p.AssignedAt, p.RemovedBy, p.RemovedAt, p.RemovalReason, true)
}

func applyImportedWorkspaceMembershipTx(ctx context.Context, tx *sql.Tx, record reconciliation.ProductionManifestRecord) (productionImportOutcome, error) {
	p := record.Payload.WorkspaceMembership
	if p == nil || p.ID != record.PlatformID || p.WorkspaceID == "" || p.UserID == "" {
		return productionImportOutcome{}, reconciliation.ErrProductionRecord
	}
	return applyImportedMembershipTx(ctx, tx, record, p.ID, "", p.WorkspaceID, p.UserID, p.Role, p.Active, p.AssignedBy, p.AssignedAt, p.RemovedBy, p.RemovedAt, p.RemovalReason, false)
}

func applyImportedMembershipTx(ctx context.Context, tx *sql.Tx, record reconciliation.ProductionManifestRecord, publicID, organisationID, workspaceID, userID, role string, active bool, assignedBy, assignedAt, removedBy string, removedAt *string, removalReason string, organisation bool) (productionImportOutcome, error) {
	if strings.TrimSpace(assignedBy) == "" || strings.TrimSpace(assignedAt) == "" || strings.TrimSpace(role) == "" {
		return productionImportOutcome{}, reconciliation.ErrProductionRecord
	}
	assignedTime, err := parseImportTimeValue(assignedAt)
	if err != nil {
		return productionImportOutcome{}, err
	}
	if organisation {
		var currentOrg, currentUser, currentRole, currentAssignedBy, currentAssignedAt, currentRemovedBy, currentRemovedAt, currentReason string
		var currentActive int
		err = tx.QueryRowContext(ctx, `SELECT o.public_id,u.public_id,m.role,m.active,assigned.public_id,m.assigned_at,COALESCE(removed.public_id,''),COALESCE(m.removed_at,''),m.removal_reason FROM organisation_memberships m JOIN organisations o ON o.id=m.organisation_id JOIN users u ON u.id=m.user_id JOIN users assigned ON assigned.id=m.assigned_by LEFT JOIN users removed ON removed.id=m.removed_by WHERE m.public_id=?`, publicID).Scan(&currentOrg, &currentUser, &currentRole, &currentActive, &currentAssignedBy, &currentAssignedAt, &currentRemovedBy, &currentRemovedAt, &currentReason)
		if err == nil {
			if currentOrg != organisationID || currentUser != userID || currentRole != role || currentActive != boolInt(active) || currentAssignedBy != assignedBy || currentAssignedAt != assignedAt || currentRemovedBy != strings.TrimSpace(removedBy) || currentRemovedAt != optionalStringValue(removedAt) || currentReason != removalReason {
				return productionImportOutcome{}, ErrProductionImportRecord
			}
			return productionImportOutcome{Outcome: "reconciled", CanonicalID: publicID}, nil
		}
	} else {
		var currentWorkspace, currentUser, currentRole, currentAssignedBy, currentAssignedAt, currentRemovedBy, currentRemovedAt, currentReason string
		var currentActive int
		err = tx.QueryRowContext(ctx, `SELECT w.public_id,u.public_id,m.role,m.active,assigned.public_id,m.assigned_at,COALESCE(removed.public_id,''),COALESCE(m.removed_at,''),m.removal_reason FROM workspace_memberships m JOIN workspaces w ON w.id=m.workspace_id JOIN users u ON u.id=m.user_id JOIN users assigned ON assigned.id=m.assigned_by LEFT JOIN users removed ON removed.id=m.removed_by WHERE m.public_id=?`, publicID).Scan(&currentWorkspace, &currentUser, &currentRole, &currentActive, &currentAssignedBy, &currentAssignedAt, &currentRemovedBy, &currentRemovedAt, &currentReason)
		if err == nil {
			if currentWorkspace != workspaceID || currentUser != userID || currentRole != role || currentActive != boolInt(active) || currentAssignedBy != assignedBy || currentAssignedAt != assignedAt || currentRemovedBy != strings.TrimSpace(removedBy) || currentRemovedAt != optionalStringValue(removedAt) || currentReason != removalReason {
				return productionImportOutcome{}, ErrProductionImportRecord
			}
			return productionImportOutcome{Outcome: "reconciled", CanonicalID: publicID}, nil
		}
	}
	if !errors.Is(err, sql.ErrNoRows) && err != nil {
		return productionImportOutcome{}, err
	}
	if record.CreatePolicy == "reconcile_only" {
		return productionImportOutcome{}, ErrProductionImportRecord
	}
	assignedActor, err := importActorTx(ctx, tx, assignedBy)
	if err != nil {
		return productionImportOutcome{}, err
	}
	removedActor, err := optionalImportActorTx(ctx, tx, removedBy)
	if err != nil {
		return productionImportOutcome{}, err
	}
	if organisation {
		var organisationIDInternal, userIDInternal int64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM organisations WHERE public_id=?`, organisationID).Scan(&organisationIDInternal); err != nil {
			return productionImportOutcome{}, err
		}
		if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE public_id=?`, userID).Scan(&userIDInternal); err != nil {
			return productionImportOutcome{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO organisation_memberships(public_id,organisation_id,user_id,role,active,assigned_by,assigned_at,removed_by,removed_at,removal_reason) VALUES(?,?,?,?,?,?,?,?,?,?)`, publicID, organisationIDInternal, userIDInternal, role, boolInt(active), assignedActor, assignedAt, removedActor, nullableString(removedAt), removalReason); err != nil {
			return productionImportOutcome{}, err
		}
		eventName := eventMembershipAdded
		if !active {
			eventName = eventMembershipDeactivated
		}
		if err := recordOrganisationAuditTx(ctx, tx, assignedActor, organisationID, eventName, organisationMutationDetails{OrganisationID: organisationID, MembershipID: publicID, UserID: userID, Role: role, Reason: removalReason}, assignedTime); err != nil {
			return productionImportOutcome{}, err
		}
	} else {
		var workspaceInternalID, userIDInternal int64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM workspaces WHERE public_id=?`, workspaceID).Scan(&workspaceInternalID); err != nil {
			return productionImportOutcome{}, err
		}
		if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE public_id=?`, userID).Scan(&userIDInternal); err != nil {
			return productionImportOutcome{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO workspace_memberships(public_id,workspace_id,user_id,role,active,assigned_by,assigned_at,removed_by,removed_at,removal_reason) VALUES(?,?,?,?,?,?,?,?,?,?)`, publicID, workspaceInternalID, userIDInternal, role, boolInt(active), assignedActor, assignedAt, removedActor, nullableString(removedAt), removalReason); err != nil {
			return productionImportOutcome{}, err
		}
		eventName := eventWorkspaceMembershipAdded
		if !active {
			eventName = eventWorkspaceMemberRemoved
		}
		if err := recordWorkspaceAuditTx(ctx, tx, assignedActor, workspaceID, eventName, workspaceMutationDetails{WorkspaceID: workspaceID, MembershipID: publicID, UserID: userID, Role: role, Reason: removalReason}, assignedTime); err != nil {
			return productionImportOutcome{}, err
		}
	}
	return productionImportOutcome{Outcome: "created", Created: true, CanonicalID: publicID}, nil
}

func applyImportedProductTx(ctx context.Context, tx *sql.Tx, record reconciliation.ProductionManifestRecord) (productionImportOutcome, error) {
	p := record.Payload.Product
	if p == nil || p.Key != record.PlatformID {
		return productionImportOutcome{}, reconciliation.ErrProductionRecord
	}
	if _, err := parseImportTimeValue(p.CreatedAt); err != nil {
		return productionImportOutcome{}, err
	}
	var name, status, created string
	var retired sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT display_name,status,created_at,retired_at FROM products WHERE product_key=?`, p.Key).Scan(&name, &status, &created, &retired)
	if errors.Is(err, sql.ErrNoRows) {
		if record.CreatePolicy == "reconcile_only" {
			return productionImportOutcome{}, ErrProductionImportRecord
		}
		actor, err := importActorTx(ctx, tx, p.History.ActorUserID)
		if err != nil {
			return productionImportOutcome{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO products(product_key,display_name,status,created_at,retired_at) VALUES(?,?,?,?,?)`, p.Key, p.DisplayName, p.Status, p.CreatedAt, nullableString(p.RetiredAt)); err != nil {
			return productionImportOutcome{}, err
		}
		if err := recordProductAuditTx(ctx, tx, actor, p.Key, eventProductCreated, productAuditDetails{ProductKey: p.Key, Status: p.Status, Reason: p.History.Reason}, parseImportTime(p.CreatedAt)); err != nil {
			return productionImportOutcome{}, err
		}
		return productionImportOutcome{Outcome: "created", Created: true, CanonicalID: p.Key}, nil
	}
	if err != nil {
		return productionImportOutcome{}, err
	}
	if name != p.DisplayName || status != p.Status || created != p.CreatedAt || nullableValue(retired) != optionalStringValue(p.RetiredAt) {
		return productionImportOutcome{}, ErrProductionImportRecord
	}
	return productionImportOutcome{Outcome: "reconciled", CanonicalID: p.Key}, nil
}

func applyImportedEntitlementTx(ctx context.Context, tx *sql.Tx, record reconciliation.ProductionManifestRecord) (productionImportOutcome, error) {
	p := record.Payload.OrganisationEntitlement
	if p == nil || p.ID != record.PlatformID {
		return productionImportOutcome{}, reconciliation.ErrProductionRecord
	}
	grantedAt, err := parseImportTimeValue(p.GrantedAt)
	if err != nil {
		return productionImportOutcome{}, err
	}
	var found int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM organisation_product_entitlements WHERE public_id=?`, p.ID).Scan(&found); err != nil {
		return productionImportOutcome{}, err
	}
	if found == 1 {
		var currentOrg, currentProduct, currentStatus, currentGrantedBy, currentGrantedAt, currentRevokedBy, currentRevokedAt, currentReason string
		if err := tx.QueryRowContext(ctx, `SELECT o.public_id,e.product_key,e.status,granted.public_id,e.granted_at,COALESCE(revoked.public_id,''),COALESCE(e.revoked_at,''),e.revocation_reason FROM organisation_product_entitlements e JOIN organisations o ON o.id=e.organisation_id JOIN users granted ON granted.id=e.granted_by LEFT JOIN users revoked ON revoked.id=e.revoked_by WHERE e.public_id=?`, p.ID).Scan(&currentOrg, &currentProduct, &currentStatus, &currentGrantedBy, &currentGrantedAt, &currentRevokedBy, &currentRevokedAt, &currentReason); err != nil {
			return productionImportOutcome{}, err
		}
		if currentOrg != p.OrganisationID || currentProduct != p.ProductKey || currentStatus != p.Status || currentGrantedBy != p.GrantedBy || currentGrantedAt != p.GrantedAt || currentRevokedBy != p.RevokedBy || currentRevokedAt != optionalStringValue(p.RevokedAt) || currentReason != p.RevocationReason {
			return productionImportOutcome{}, ErrProductionImportRecord
		}
		return productionImportOutcome{Outcome: "reconciled", CanonicalID: p.ID}, nil
	}
	if record.CreatePolicy == "reconcile_only" {
		return productionImportOutcome{}, ErrProductionImportRecord
	}
	actor, err := importActorTx(ctx, tx, p.GrantedBy)
	if err != nil {
		return productionImportOutcome{}, err
	}
	removedActor, err := optionalImportActorTx(ctx, tx, p.RevokedBy)
	if err != nil {
		return productionImportOutcome{}, err
	}
	var organisationInternal int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM organisations WHERE public_id=?`, p.OrganisationID).Scan(&organisationInternal); err != nil {
		return productionImportOutcome{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO organisation_product_entitlements(public_id,organisation_id,product_key,status,granted_by,granted_at,revoked_by,revoked_at,revocation_reason) VALUES(?,?,?,?,?,?,?,?,?)`, p.ID, organisationInternal, p.ProductKey, p.Status, actor, p.GrantedAt, removedActor, nullableString(p.RevokedAt), p.RevocationReason); err != nil {
		return productionImportOutcome{}, err
	}
	eventName := eventEntitlementGranted
	if p.Status == "revoked" {
		eventName = eventEntitlementRevoked
	}
	if err := recordOrganisationProductAuditTx(ctx, tx, actor, p.OrganisationID, eventName, productAuditDetails{ProductKey: p.ProductKey, OrganisationID: p.OrganisationID, EntitlementID: p.ID, Status: p.Status, Reason: p.RevocationReason}, grantedAt); err != nil {
		return productionImportOutcome{}, err
	}
	return productionImportOutcome{Outcome: "created", Created: true, CanonicalID: p.ID}, nil
}

func applyImportedAssignmentTx(ctx context.Context, tx *sql.Tx, record reconciliation.ProductionManifestRecord) (productionImportOutcome, error) {
	p := record.Payload.UserAssignment
	if p == nil || p.ID != record.PlatformID {
		return productionImportOutcome{}, reconciliation.ErrProductionRecord
	}
	assignedAt, err := parseImportTimeValue(p.AssignedAt)
	if err != nil {
		return productionImportOutcome{}, err
	}
	var found int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_product_assignments WHERE public_id=?`, p.ID).Scan(&found); err != nil {
		return productionImportOutcome{}, err
	}
	if found == 1 {
		var currentUser, currentOrg, currentProduct, currentStatus, currentAssignedBy, currentAssignedAt, currentRevokedBy, currentRevokedAt, currentReason string
		if err := tx.QueryRowContext(ctx, `SELECT u.public_id,o.public_id,a.product_key,a.status,assigned.public_id,a.assigned_at,COALESCE(revoked.public_id,''),COALESCE(a.revoked_at,''),a.revocation_reason FROM user_product_assignments a JOIN users u ON u.id=a.user_id JOIN organisations o ON o.id=a.organisation_id JOIN users assigned ON assigned.id=a.assigned_by LEFT JOIN users revoked ON revoked.id=a.revoked_by WHERE a.public_id=?`, p.ID).Scan(&currentUser, &currentOrg, &currentProduct, &currentStatus, &currentAssignedBy, &currentAssignedAt, &currentRevokedBy, &currentRevokedAt, &currentReason); err != nil {
			return productionImportOutcome{}, err
		}
		if currentUser != p.UserID || currentOrg != p.OrganisationID || currentProduct != p.ProductKey || currentStatus != p.Status || currentAssignedBy != p.AssignedBy || currentAssignedAt != p.AssignedAt || currentRevokedBy != p.RevokedBy || currentRevokedAt != optionalStringValue(p.RevokedAt) || currentReason != p.RevocationReason {
			return productionImportOutcome{}, ErrProductionImportRecord
		}
		return productionImportOutcome{Outcome: "reconciled", CanonicalID: p.ID}, nil
	}
	if record.CreatePolicy == "reconcile_only" {
		return productionImportOutcome{}, ErrProductionImportRecord
	}
	actor, err := importActorTx(ctx, tx, p.AssignedBy)
	if err != nil {
		return productionImportOutcome{}, err
	}
	removedActor, err := optionalImportActorTx(ctx, tx, p.RevokedBy)
	if err != nil {
		return productionImportOutcome{}, err
	}
	var userInternal, organisationInternal int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE public_id=?`, p.UserID).Scan(&userInternal); err != nil {
		return productionImportOutcome{}, err
	}
	if err := tx.QueryRowContext(ctx, `SELECT id FROM organisations WHERE public_id=?`, p.OrganisationID).Scan(&organisationInternal); err != nil {
		return productionImportOutcome{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO user_product_assignments(public_id,user_id,organisation_id,product_key,status,assigned_by,assigned_at,revoked_by,revoked_at,revocation_reason) VALUES(?,?,?,?,?,?,?,?,?,?)`, p.ID, userInternal, organisationInternal, p.ProductKey, p.Status, actor, p.AssignedAt, removedActor, nullableString(p.RevokedAt), p.RevocationReason); err != nil {
		return productionImportOutcome{}, err
	}
	eventName := eventAssignmentGranted
	if p.Status == "revoked" {
		eventName = eventAssignmentRevoked
	}
	if err := recordOrganisationProductAuditTx(ctx, tx, actor, p.OrganisationID, eventName, productAuditDetails{ProductKey: p.ProductKey, OrganisationID: p.OrganisationID, AssignmentID: p.ID, UserID: p.UserID, Status: p.Status, Reason: p.RevocationReason}, assignedAt); err != nil {
		return productionImportOutcome{}, err
	}
	return productionImportOutcome{Outcome: "created", Created: true, CanonicalID: p.ID}, nil
}

func importActorTx(ctx context.Context, tx *sql.Tx, publicID string) (int64, error) {
	if strings.TrimSpace(publicID) == "" {
		return 0, reconciliation.ErrProductionRecord
	}
	var id int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE public_id=? AND active=1`, strings.TrimSpace(publicID)).Scan(&id); err != nil {
		return 0, fmt.Errorf("resolve import actor: %w", err)
	}
	return id, nil
}

func optionalImportActorTx(ctx context.Context, tx *sql.Tx, publicID string) (any, error) {
	if strings.TrimSpace(publicID) == "" {
		return nil, nil
	}
	return importActorTx(ctx, tx, publicID)
}

func parseImportTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	return parsed
}
func parseImportTimeValue(value string) (time.Time, error) {
	parsed := parseImportTime(value)
	if parsed.IsZero() {
		return time.Time{}, reconciliation.ErrProductionRecord
	}
	return parsed, nil
}
func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}
func nullableValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func deleteImportedRecordTx(ctx context.Context, tx *sql.Tx, entity, publicID string) error {
	var query string
	switch reconciliation.ProductionEntityKind(entity) {
	case reconciliation.ProductionEntityUser:
		query = `DELETE FROM users WHERE public_id=?`
	case reconciliation.ProductionEntityOrganisation:
		query = `DELETE FROM organisations WHERE public_id=?`
	case reconciliation.ProductionEntityWorkspace:
		query = `DELETE FROM workspaces WHERE public_id=?`
	case reconciliation.ProductionEntityOrganisationMembership:
		query = `DELETE FROM organisation_memberships WHERE public_id=?`
	case reconciliation.ProductionEntityWorkspaceMembership:
		query = `DELETE FROM workspace_memberships WHERE public_id=?`
	case reconciliation.ProductionEntityProduct:
		query = `DELETE FROM products WHERE product_key=?`
	case reconciliation.ProductionEntityOrganisationEntitlement:
		query = `DELETE FROM organisation_product_entitlements WHERE public_id=?`
	case reconciliation.ProductionEntityUserAssignment:
		query = `DELETE FROM user_product_assignments WHERE public_id=?`
	default:
		return reconciliation.ErrProductionRecord
	}
	result, err := tx.ExecContext(ctx, query, publicID)
	if err != nil {
		return fmt.Errorf("%w: delete %s %q: %v", ErrProductionImportRollback, entity, publicID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("%w: imported record %s %q no longer exists", ErrProductionImportRollback, entity, publicID)
	}
	return nil
}

func digestBytes(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }

var _ store.PlatformStore = (*Store)(nil)
