package reconciliation

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
)

// Production imports are deliberately separate from the terminal fixture
// manifest.  A fixture manifest can never be upgraded into a production
// write by changing a scope string.
const (
	ProductionSchemaVersion = 1
	ProductionScope         = "production"
	ProductionManifestType  = "platform.reconciliation.import.v1"
)

type ProductionEntityKind string

const (
	ProductionEntityUser                    ProductionEntityKind = "user"
	ProductionEntityOrganisation            ProductionEntityKind = "organisation"
	ProductionEntityWorkspace               ProductionEntityKind = "workspace"
	ProductionEntityOrganisationMembership  ProductionEntityKind = "organisation_membership"
	ProductionEntityWorkspaceMembership     ProductionEntityKind = "workspace_membership"
	ProductionEntityProduct                 ProductionEntityKind = "product"
	ProductionEntityOrganisationEntitlement ProductionEntityKind = "organisation_product_entitlement"
	ProductionEntityUserAssignment          ProductionEntityKind = "user_product_assignment"
)

var (
	ErrProductionManifestIntegrity = errors.New("production manifest identity is invalid")
	ErrProductionApprovalRequired  = errors.New("production manifest approval is required")
	ErrProductionSignature         = errors.New("production manifest signature is invalid")
	ErrProductionExecution         = errors.New("production execution authorization is invalid")
	ErrProductionRecord            = errors.New("production manifest record is invalid")
)

// ProductionSourceSnapshot binds every record to the exact source export
// that was reviewed.  SourceSHA256 is the source repository commit or export
// digest, never a locally generated timestamp.
type ProductionSourceSnapshot struct {
	Source       SourceSystem `json:"source"`
	SourceSHA256 string       `json:"source_sha256"`
	ReportSHA256 string       `json:"report_sha256"`
}

// ProductionApproval records the human approval that authorizes production
// scope.  It is signed with the manifest and is never inferred by the
// importer.
type ProductionApproval struct {
	ApprovalID string `json:"approval_id"`
	OperatorID string `json:"operator_id"`
	ApprovedAt string `json:"approved_at"`
	Reason     string `json:"reason"`
	Scope      string `json:"scope"`
	KeyID      string `json:"key_id"`
}

// ProductionExecution is supplied by the protected operator boundary at
// apply time.  The values must match the signed approval exactly.
type ProductionExecution struct {
	OperatorID string
	ApprovalID string
	KeyID      string
}

type ImportHistory struct {
	ActorUserID string `json:"actor_user_id,omitempty"`
	OccurredAt  string `json:"occurred_at"`
	Reason      string `json:"reason"`
}

type UserImportPayload struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Active      bool   `json:"active"`
	CreatedAt   string `json:"created_at"`
}

type OrganisationImportPayload struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Status     string        `json:"status"`
	CreatedAt  string        `json:"created_at"`
	ArchivedAt *string       `json:"archived_at,omitempty"`
	History    ImportHistory `json:"history"`
}

type WorkspaceImportPayload struct {
	ID             string        `json:"id"`
	OrganisationID string        `json:"organisation_id"`
	Name           string        `json:"name"`
	Status         string        `json:"status"`
	CreatedAt      string        `json:"created_at"`
	ArchivedAt     *string       `json:"archived_at,omitempty"`
	History        ImportHistory `json:"history"`
}

type MembershipImportPayload struct {
	ID             string  `json:"id"`
	OrganisationID string  `json:"organisation_id,omitempty"`
	WorkspaceID    string  `json:"workspace_id,omitempty"`
	UserID         string  `json:"user_id"`
	Role           string  `json:"role"`
	Active         bool    `json:"active"`
	AssignedBy     string  `json:"assigned_by"`
	AssignedAt     string  `json:"assigned_at"`
	RemovedBy      string  `json:"removed_by,omitempty"`
	RemovedAt      *string `json:"removed_at,omitempty"`
	RemovalReason  string  `json:"removal_reason,omitempty"`
}

type ProductImportPayload struct {
	Key         string        `json:"key"`
	DisplayName string        `json:"display_name"`
	Status      string        `json:"status"`
	CreatedAt   string        `json:"created_at"`
	RetiredAt   *string       `json:"retired_at,omitempty"`
	History     ImportHistory `json:"history"`
}

type EntitlementImportPayload struct {
	ID               string  `json:"id"`
	OrganisationID   string  `json:"organisation_id"`
	ProductKey       string  `json:"product_key"`
	Status           string  `json:"status"`
	GrantedBy        string  `json:"granted_by"`
	GrantedAt        string  `json:"granted_at"`
	RevokedBy        string  `json:"revoked_by,omitempty"`
	RevokedAt        *string `json:"revoked_at,omitempty"`
	RevocationReason string  `json:"revocation_reason,omitempty"`
}

type AssignmentImportPayload struct {
	ID               string  `json:"id"`
	UserID           string  `json:"user_id"`
	OrganisationID   string  `json:"organisation_id"`
	ProductKey       string  `json:"product_key"`
	Status           string  `json:"status"`
	AssignedBy       string  `json:"assigned_by"`
	AssignedAt       string  `json:"assigned_at"`
	RevokedBy        string  `json:"revoked_by,omitempty"`
	RevokedAt        *string `json:"revoked_at,omitempty"`
	RevocationReason string  `json:"revocation_reason,omitempty"`
}

// ProductionPayload contains the minimum canonical data needed to create or
// reconcile one record.  The matching pointer must be the only non-nil field.
type ProductionPayload struct {
	User                    *UserImportPayload         `json:"user,omitempty"`
	Organisation            *OrganisationImportPayload `json:"organisation,omitempty"`
	Workspace               *WorkspaceImportPayload    `json:"workspace,omitempty"`
	OrganisationMembership  *MembershipImportPayload   `json:"organisation_membership,omitempty"`
	WorkspaceMembership     *MembershipImportPayload   `json:"workspace_membership,omitempty"`
	Product                 *ProductImportPayload      `json:"product,omitempty"`
	OrganisationEntitlement *EntitlementImportPayload  `json:"organisation_entitlement,omitempty"`
	UserAssignment          *AssignmentImportPayload   `json:"user_assignment,omitempty"`
}

type ProductionManifestRecord struct {
	Entity         ProductionEntityKind `json:"entity"`
	PlatformID     string               `json:"platform_id"`
	MatchKeySHA256 string               `json:"match_key_sha256"`
	SourceRefs     []SourceRef          `json:"source_refs"`
	CreatePolicy   string               `json:"create_policy"`
	Payload        ProductionPayload    `json:"payload"`
}

type ProductionManifest struct {
	SchemaVersion      int                        `json:"schema_version"`
	ManifestType       string                     `json:"manifest_type"`
	Scope              string                     `json:"scope"`
	ManifestID         string                     `json:"manifest_id"`
	SourceReportSHA256 string                     `json:"source_report_sha256"`
	SourceSnapshots    []ProductionSourceSnapshot `json:"source_snapshots"`
	Records            []ProductionManifestRecord `json:"records"`
	Approval           *ProductionApproval        `json:"approval,omitempty"`
}

type SignedProductionManifest struct {
	Manifest  ProductionManifest `json:"manifest"`
	Algorithm string             `json:"algorithm"`
	Signature string             `json:"signature"`
}

type ProductionImportReceipt struct {
	ManifestID         string `json:"manifest_id"`
	ManifestSHA256     string `json:"manifest_sha256"`
	SourceReportSHA256 string `json:"source_report_sha256"`
	OperatorID         string `json:"operator_id"`
	ApprovalID         string `json:"approval_id"`
	KeyID              string `json:"key_id"`
	Status             string `json:"status"`
	NextIndex          int    `json:"next_index"`
	Applied            int    `json:"applied"`
	Skipped            int    `json:"skipped"`
	Created            int    `json:"created"`
	Reconciled         int    `json:"reconciled"`
	Conflicts          int    `json:"conflicts"`
	OccurredAt         string `json:"occurred_at"`
}

const (
	ProductionImportPaused     = "paused"
	ProductionImportCompleted  = "completed"
	ProductionImportIdempotent = "idempotent"
	ProductionImportRolledBack = "rolled_back"
	ProductionImportDryRun     = "dry_run"
)

func SignProductionManifest(manifest ProductionManifest, privateKey ed25519.PrivateKey) (SignedProductionManifest, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return SignedProductionManifest{}, ErrProductionSignature
	}
	if err := ValidateProductionManifest(manifest); err != nil {
		return SignedProductionManifest{}, err
	}
	payload, err := json.Marshal(manifest)
	if err != nil {
		return SignedProductionManifest{}, err
	}
	return SignedProductionManifest{Manifest: manifest, Algorithm: "Ed25519", Signature: base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, payload))}, nil
}

func VerifySignedProductionManifest(signed SignedProductionManifest, publicKey ed25519.PublicKey) error {
	if len(publicKey) != ed25519.PublicKeySize || signed.Algorithm != "Ed25519" {
		return ErrProductionSignature
	}
	if err := ValidateProductionManifest(signed.Manifest); err != nil {
		return err
	}
	payload, err := json.Marshal(signed.Manifest)
	if err != nil {
		return ErrProductionSignature
	}
	signature, err := base64.RawURLEncoding.DecodeString(signed.Signature)
	if err != nil || !ed25519.Verify(publicKey, payload, signature) {
		return ErrProductionSignature
	}
	return nil
}

func ValidateProductionManifest(manifest ProductionManifest) error {
	if manifest.SchemaVersion != ProductionSchemaVersion || manifest.ManifestType != ProductionManifestType || manifest.Scope != ProductionScope || !isSHA256Hex(manifest.ManifestID) || !isSHA256Hex(manifest.SourceReportSHA256) || len(manifest.Records) == 0 || len(manifest.SourceSnapshots) == 0 {
		return ErrProductionManifestIntegrity
	}
	if manifest.ManifestID != productionManifestIdentity(manifest) {
		return ErrProductionManifestIntegrity
	}
	if manifest.Approval == nil || manifest.Approval.Scope != ProductionScope || strings.TrimSpace(manifest.Approval.ApprovalID) == "" || strings.TrimSpace(manifest.Approval.OperatorID) == "" || strings.TrimSpace(manifest.Approval.Reason) == "" || strings.TrimSpace(manifest.Approval.KeyID) == "" {
		return ErrProductionApprovalRequired
	}
	if _, err := time.Parse(time.RFC3339Nano, manifest.Approval.ApprovedAt); err != nil {
		return ErrProductionApprovalRequired
	}
	seenSources := make(map[SourceSystem]struct{}, len(manifest.SourceSnapshots))
	for _, snapshot := range manifest.SourceSnapshots {
		if snapshot.Source != SourceCTRL && snapshot.Source != SourceIMS || !isSourceDigest(snapshot.SourceSHA256) || !isSHA256Hex(snapshot.ReportSHA256) {
			return ErrProductionManifestIntegrity
		}
		if snapshot.ReportSHA256 != manifest.SourceReportSHA256 {
			return ErrProductionManifestIntegrity
		}
		if _, exists := seenSources[snapshot.Source]; exists {
			return ErrProductionManifestIntegrity
		}
		seenSources[snapshot.Source] = struct{}{}
	}
	seenIDs := make(map[string]struct{}, len(manifest.Records))
	for index, record := range manifest.Records {
		if !validProductionRecord(record) || record.PlatformID == "" || !isSHA256Hex(record.MatchKeySHA256) || len(record.SourceRefs) == 0 || (index > 0 && productionRecordLess(record, manifest.Records[index-1])) {
			return ErrProductionRecord
		}
		if _, exists := seenIDs[record.PlatformID]; exists {
			return ErrProductionRecord
		}
		seenIDs[record.PlatformID] = struct{}{}
		for _, ref := range record.SourceRefs {
			if (ref.Source != SourceCTRL && ref.Source != SourceIMS) || !isSourceSHA(ref.SourceVersion) || strings.TrimSpace(ref.SourceID) == "" {
				return ErrProductionRecord
			}
			if _, exists := seenSources[ref.Source]; !exists {
				return ErrProductionRecord
			}
		}
	}
	return nil
}

func isSourceDigest(value string) bool {
	return isSourceSHA(value) || isSHA256Hex(value)
}

func ValidateProductionExecution(manifest ProductionManifest, execution ProductionExecution) error {
	if manifest.Approval == nil || strings.TrimSpace(execution.OperatorID) == "" || strings.TrimSpace(execution.ApprovalID) == "" || strings.TrimSpace(execution.KeyID) == "" || execution.OperatorID != manifest.Approval.OperatorID || execution.ApprovalID != manifest.Approval.ApprovalID || execution.KeyID != manifest.Approval.KeyID {
		return ErrProductionExecution
	}
	return nil
}

func productionManifestIdentity(manifest ProductionManifest) string {
	unsigned := manifest
	unsigned.ManifestID = ""
	unsigned.Approval = nil
	encoded, _ := json.Marshal(unsigned)
	hash := sha256.Sum256(encoded)
	return hex.EncodeToString(hash[:])
}

func ProductionManifestID(manifest ProductionManifest) string {
	return productionManifestIdentity(manifest)
}

func ProductionManifestDigest(manifest ProductionManifest) string {
	encoded, _ := json.Marshal(manifest)
	hash := sha256.Sum256(encoded)
	return hex.EncodeToString(hash[:])
}

func validProductionRecord(record ProductionManifestRecord) bool {
	if strings.TrimSpace(record.CreatePolicy) != "create_or_reconcile" && strings.TrimSpace(record.CreatePolicy) != "reconcile_only" {
		return false
	}
	count := 0
	switch record.Entity {
	case ProductionEntityUser:
		count = boolInt(record.Payload.User != nil)
	case ProductionEntityOrganisation:
		count = boolInt(record.Payload.Organisation != nil)
	case ProductionEntityWorkspace:
		count = boolInt(record.Payload.Workspace != nil)
	case ProductionEntityOrganisationMembership:
		count = boolInt(record.Payload.OrganisationMembership != nil)
	case ProductionEntityWorkspaceMembership:
		count = boolInt(record.Payload.WorkspaceMembership != nil)
	case ProductionEntityProduct:
		count = boolInt(record.Payload.Product != nil)
	case ProductionEntityOrganisationEntitlement:
		count = boolInt(record.Payload.OrganisationEntitlement != nil)
	case ProductionEntityUserAssignment:
		count = boolInt(record.Payload.UserAssignment != nil)
	default:
		return false
	}
	return count == 1
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func productionRecordRank(kind ProductionEntityKind) int {
	for index, allowed := range []ProductionEntityKind{ProductionEntityUser, ProductionEntityOrganisation, ProductionEntityWorkspace, ProductionEntityOrganisationMembership, ProductionEntityWorkspaceMembership, ProductionEntityProduct, ProductionEntityOrganisationEntitlement, ProductionEntityUserAssignment} {
		if kind == allowed {
			return index
		}
	}
	return 999
}

func productionRecordLess(left, right ProductionManifestRecord) bool {
	leftRank, rightRank := productionRecordRank(left.Entity), productionRecordRank(right.Entity)
	if leftRank != rightRank {
		return leftRank < rightRank
	}
	return left.PlatformID < right.PlatformID
}

// SortProductionRecords returns a defensive, dependency-first copy.
func SortProductionRecords(records []ProductionManifestRecord) []ProductionManifestRecord {
	result := append([]ProductionManifestRecord(nil), records...)
	sort.SliceStable(result, func(i, j int) bool { return productionRecordLess(result[i], result[j]) })
	return result
}

func NewProductionManifest(reportSHA string, snapshots []ProductionSourceSnapshot, records []ProductionManifestRecord, approval ProductionApproval) (ProductionManifest, error) {
	manifest := ProductionManifest{SchemaVersion: ProductionSchemaVersion, ManifestType: ProductionManifestType, Scope: ProductionScope, SourceReportSHA256: strings.TrimSpace(reportSHA), SourceSnapshots: append([]ProductionSourceSnapshot(nil), snapshots...), Records: SortProductionRecords(records), Approval: &approval}
	manifest.ManifestID = productionManifestIdentity(manifest)
	if err := ValidateProductionManifest(manifest); err != nil {
		return ProductionManifest{}, err
	}
	return manifest, nil
}
