// Package readauthority contains the stable wire contract shared by Platform
// and its local projection consumers. Runtime storage and HTTP handlers belong
// to later PF-B11 slices; this package deliberately contains no persistence or
// product authorization behavior.
package readauthority

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

const (
	VersionV1                 = "platform.read-authority.v1"
	VersionV2                 = "platform.read-authority.v2"
	Version                   = VersionV2
	VersionHeader             = "X-Tockr-Platform-Read-Authority-Version"
	ConsumerHeader            = "X-Tockr-Platform-Consumer"
	KeyIDHeader               = "X-Tockr-Platform-Key-ID"
	TimestampHeader           = "X-Tockr-Platform-Timestamp"
	NonceHeader               = "X-Tockr-Platform-Nonce"
	SignatureHeader           = "X-Tockr-Platform-Signature"
	ConsumerCTRL              = "tockrctrl"
	ConsumerIMS               = "tockrims"
	ProductCTRL               = "product.tockrctrl"
	ProductIMS                = "product.tockrims"
	SignatureMaxClockSkewSecs = int64(300)
	NonceMaxLength            = 128
	SnapshotMaxRecords        = 10000
	SnapshotMaxPageSize       = 500
	ChangeMaxPageSize         = 500
	RequestMaxBodyBytes       = 64 * 1024
	ResponseMaxBytes          = 4 * 1024 * 1024
	SnapshotMaxTTLSeconds     = 24 * 60 * 60
	CursorRetentionMinSeconds = 7 * 24 * 60 * 60
)

var (
	ErrUnsupportedVersion = errors.New("unsupported read-authority version")
	ErrInvalidConsumer    = errors.New("invalid read-authority consumer")
	ErrInvalidRequest     = errors.New("invalid read-authority request")
	ErrInvalidRecord      = errors.New("invalid read-authority record")
	ErrInvalidState       = errors.New("invalid read-authority state")
	ErrInvalidCursor      = errors.New("invalid read-authority cursor")
	ErrInvalidSignature   = errors.New("invalid read-authority signature input")

	requestTokenPattern = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)
)

// EntityKind is the complete shared-record allow-list. Product role and
// billing records intentionally have no value in this type.
type EntityKind string

const (
	EntityUser                    EntityKind = "user"
	EntityOrganisation            EntityKind = "organisation"
	EntityOrganisationMembership  EntityKind = "organisation_membership"
	EntityWorkspace               EntityKind = "workspace"
	EntityWorkspaceMembership     EntityKind = "workspace_membership"
	EntityProduct                 EntityKind = "product"
	EntityOrganisationEntitlement EntityKind = "organisation_product_entitlement"
	EntityUserProductAssignment   EntityKind = "user_product_assignment"
)

// allowedEntityKinds is the canonical deterministic record order. It is
// intentionally private so callers cannot mutate the contract in place.
var allowedEntityKinds = [...]EntityKind{
	EntityUser,
	EntityOrganisation,
	EntityOrganisationMembership,
	EntityWorkspace,
	EntityWorkspaceMembership,
	EntityProduct,
	EntityOrganisationEntitlement,
	EntityUserProductAssignment,
}

const SourceSchemaVersion = 1

// ProvenanceKind identifies how a Platform record entered authoritative
// history. Event provenance is the only kind supported by read-authority v1.
// Migration seeds are explicit bootstrap facts for rows created by an ordered
// schema migration before the outbox existed; they must never be represented
// as synthetic events.
type ProvenanceKind string

const (
	ProvenanceEvent         ProvenanceKind = "event"
	ProvenanceMigrationSeed ProvenanceKind = "migration_seed"
)

// AllowedEntityKinds returns a defensive copy of the complete allow-list in
// canonical snapshot order.
func AllowedEntityKinds() []EntityKind {
	return append([]EntityKind(nil), allowedEntityKinds[:]...)
}

type State string

const (
	StateCurrent        State = "current"
	StateStale          State = "stale"
	StateGap            State = "gap"
	StateBlocked        State = "blocked"
	StateUnavailable    State = "unavailable"
	StateResyncRequired State = "resync_required"
)

type RecordStatus string

const (
	StatusActive   RecordStatus = "active"
	StatusArchived RecordStatus = "archived"
	StatusRevoked  RecordStatus = "revoked"
	StatusRetired  RecordStatus = "retired"
)

type ErrorCode string

const (
	ErrorVersionMismatch  ErrorCode = "version_mismatch"
	ErrorUnauthenticated  ErrorCode = "unauthenticated"
	ErrorForbidden        ErrorCode = "forbidden"
	ErrorInvalidRequest   ErrorCode = "invalid_request"
	ErrorLimitExceeded    ErrorCode = "limit_exceeded"
	ErrorStale            ErrorCode = "stale"
	ErrorGap              ErrorCode = "gap"
	ErrorBlocked          ErrorCode = "blocked"
	ErrorUnavailable      ErrorCode = "unavailable"
	ErrorResyncRequired   ErrorCode = "resync_required"
	ErrorCursorExpired    ErrorCode = "cursor_expired"
	ErrorSnapshotExpired  ErrorCode = "snapshot_expired"
	ErrorSnapshotPartial  ErrorCode = "snapshot_incomplete"
	ErrorChecksumMismatch ErrorCode = "checksum_mismatch"
	ErrorConflict         ErrorCode = "conflict"
)

// SnapshotRequest is the only request body in the S01 contract. The consumer
// identity is taken from machine authentication headers, not trusted from the
// request body.
type SnapshotRequest struct {
	EntityKinds      []EntityKind `json:"entity_kinds"`
	ExpiresInSeconds int          `json:"expires_in_seconds"`
}

// SnapshotMetadata identifies a finalized immutable snapshot. S02 owns its
// durable implementation; this type freezes the fields and their meanings.
type SnapshotMetadata struct {
	Version           string `json:"version"`
	SnapshotID        string `json:"snapshot_id"`
	Consumer          string `json:"consumer"`
	ContractVersion   string `json:"contract_version"`
	SourceCursor      string `json:"source_cursor"`
	ChecksumAlgorithm string `json:"checksum_algorithm"`
	Checksum          string `json:"checksum"`
	CreatedAt         string `json:"created_at"`
	ExpiresAt         string `json:"expires_at"`
	RecordCount       int    `json:"record_count"`
	PageSize          int    `json:"page_size"`
	Complete          bool   `json:"complete"`
}

// Record is the bounded shared projection representation. Optional
// relationship fields are populated only for the entity kinds whose table in
// the wire contract permits them. V2 requires explicit provenance kind and
// validates exactly one of event or migration-seed provenance.
type Record struct {
	EntityKind          EntityKind     `json:"entity_kind"`
	ID                  string         `json:"id"`
	Status              RecordStatus   `json:"status"`
	UserID              string         `json:"user_id,omitempty"`
	OrganisationID      string         `json:"organisation_id,omitempty"`
	WorkspaceID         string         `json:"workspace_id,omitempty"`
	ProductKey          string         `json:"product_key,omitempty"`
	Role                string         `json:"role,omitempty"`
	Name                string         `json:"name,omitempty"`
	ProvenanceKind      ProvenanceKind `json:"provenance_kind,omitempty"`
	SourceEventID       string         `json:"source_event_id,omitempty"`
	SourceSequence      int64          `json:"source_sequence,omitempty"`
	SourceSchemaVersion int            `json:"source_schema_version,omitempty"`
	MigrationVersion    int            `json:"migration_version,omitempty"`
	MigrationName       string         `json:"migration_name,omitempty"`
	MigrationChecksum   string         `json:"migration_checksum,omitempty"`
}

// Change is a committed outbox event with a separate opaque global cursor.
// Its payload remains governed by platform-events-v1; no new event payload is
// introduced by this contract package.
type Change struct {
	Cursor string          `json:"cursor"`
	Event  json.RawMessage `json:"event"`
}

// ErrorResponse is intentionally non-sensitive. Implementations must not add
// SQL errors, key material, record values or signature material to Message.
type ErrorResponse struct {
	Version        string    `json:"version"`
	Code           ErrorCode `json:"code"`
	State          State     `json:"state,omitempty"`
	Retryable      bool      `json:"retryable"`
	ResyncRequired bool      `json:"resync_required"`
	Message        string    `json:"message"`
}

func ProductForConsumer(consumer string) (string, bool) {
	switch strings.TrimSpace(consumer) {
	case ConsumerCTRL:
		return ProductCTRL, true
	case ConsumerIMS:
		return ProductIMS, true
	default:
		return "", false
	}
}

func ValidateVersion(value string) error {
	switch strings.TrimSpace(value) {
	case VersionV1, VersionV2:
		return nil
	default:
		return ErrUnsupportedVersion
	}
}

func ValidateConsumer(value string) error {
	if _, ok := ProductForConsumer(value); !ok {
		return ErrInvalidConsumer
	}
	return nil
}

func IsAuthoritative(state State) bool { return state == StateCurrent }

func ValidateState(state State) error {
	switch state {
	case StateCurrent, StateStale, StateGap, StateBlocked, StateUnavailable, StateResyncRequired:
		return nil
	default:
		return ErrInvalidState
	}
}

func ValidateEntityKind(kind EntityKind) error {
	for _, allowed := range allowedEntityKinds {
		if kind == allowed {
			return nil
		}
	}
	return fmt.Errorf("%w: %q", ErrInvalidRequest, kind)
}

func ValidateSnapshotRequest(request SnapshotRequest) error {
	if len(request.EntityKinds) == 0 || len(request.EntityKinds) > len(allowedEntityKinds) {
		return fmt.Errorf("%w: entity_kinds must contain one to %d values", ErrInvalidRequest, len(allowedEntityKinds))
	}
	seen := make(map[EntityKind]struct{}, len(request.EntityKinds))
	for _, kind := range request.EntityKinds {
		if err := ValidateEntityKind(kind); err != nil {
			return err
		}
		if _, exists := seen[kind]; exists {
			return fmt.Errorf("%w: duplicate entity kind", ErrInvalidRequest)
		}
		seen[kind] = struct{}{}
	}
	if request.ExpiresInSeconds <= 0 || request.ExpiresInSeconds > SnapshotMaxTTLSeconds {
		return fmt.Errorf("%w: snapshot expiry is outside the bounded range", ErrInvalidRequest)
	}
	return nil
}

// ValidateRecord enforces the per-entity field and relationship allow-list at
// the contract seam. Later snapshot materialization must call this before a
// record can be persisted or returned.
func ValidateRecord(record Record) error {
	return ValidateRecordVersion(Version, record)
}

// ValidateRecordVersion preserves the terminal v1 event-only semantics while
// making the v2 migration-seed extension explicit and fail closed. Callers
// handling a versioned wire response must pass the response contract version.
func ValidateRecordVersion(version string, record Record) error {
	if err := ValidateVersion(version); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRecord, err)
	}
	if err := ValidateEntityKind(record.EntityKind); err != nil {
		return fmt.Errorf("%w: entity kind: %v", ErrInvalidRecord, err)
	}
	if err := validateProvenance(version, record); err != nil {
		return err
	}
	if record.Status != StatusActive && record.Status != StatusArchived && record.Status != StatusRevoked && record.Status != StatusRetired {
		return fmt.Errorf("%w: invalid status", ErrInvalidRecord)
	}
	if record.ID == "" {
		return fmt.Errorf("%w: missing id", ErrInvalidRecord)
	}
	switch record.EntityKind {
	case EntityUser:
		if !hasPrefixID(record.ID, "usr_") || !statusAllowed(record.Status, StatusActive, StatusArchived) || record.hasAny("organisation_id", "workspace_id", "product_key", "role", "name", "user_id") {
			return fmt.Errorf("%w: invalid user fields", ErrInvalidRecord)
		}
	case EntityOrganisation:
		if !hasPrefixID(record.ID, "org_") || !statusAllowed(record.Status, StatusActive, StatusArchived) || !validName(record.Name) || record.hasAny("user_id", "organisation_id", "workspace_id", "product_key", "role") {
			return fmt.Errorf("%w: invalid organisation fields", ErrInvalidRecord)
		}
	case EntityOrganisationMembership:
		if !hasPrefixID(record.ID, "omem_") || !hasPrefixID(record.UserID, "usr_") || !hasPrefixID(record.OrganisationID, "org_") || !validRole(record.Role) || !statusAllowed(record.Status, StatusActive, StatusRevoked) || record.hasAny("workspace_id", "product_key", "name") {
			return fmt.Errorf("%w: invalid organisation membership fields", ErrInvalidRecord)
		}
	case EntityWorkspace:
		if !hasPrefixID(record.ID, "wsp_") || !hasPrefixID(record.OrganisationID, "org_") || !statusAllowed(record.Status, StatusActive, StatusArchived) || !validName(record.Name) || record.hasAny("user_id", "workspace_id", "product_key", "role") {
			return fmt.Errorf("%w: invalid workspace fields", ErrInvalidRecord)
		}
	case EntityWorkspaceMembership:
		if !hasPrefixID(record.ID, "wmem_") || !hasPrefixID(record.UserID, "usr_") || !hasPrefixID(record.WorkspaceID, "wsp_") || !validRole(record.Role) || !statusAllowed(record.Status, StatusActive, StatusRevoked) || record.hasAny("organisation_id", "product_key", "name") {
			return fmt.Errorf("%w: invalid workspace membership fields", ErrInvalidRecord)
		}
	case EntityProduct:
		if !hasPrefixID(record.ID, "product.") || !statusAllowed(record.Status, StatusActive, StatusRetired) || record.hasAny("user_id", "organisation_id", "workspace_id", "product_key", "role", "name") {
			return fmt.Errorf("%w: invalid product fields", ErrInvalidRecord)
		}
	case EntityOrganisationEntitlement:
		if !hasPrefixID(record.ID, "ent_") || !hasPrefixID(record.OrganisationID, "org_") || !validProductKey(record.ProductKey) || !statusAllowed(record.Status, StatusActive, StatusRevoked) || record.hasAny("user_id", "workspace_id", "role", "name") {
			return fmt.Errorf("%w: invalid organisation entitlement fields", ErrInvalidRecord)
		}
	case EntityUserProductAssignment:
		if !hasPrefixID(record.ID, "upa_") || !hasPrefixID(record.UserID, "usr_") || !hasPrefixID(record.OrganisationID, "org_") || !validProductKey(record.ProductKey) || !statusAllowed(record.Status, StatusActive, StatusRevoked) || record.hasAny("workspace_id", "role", "name") {
			return fmt.Errorf("%w: invalid user assignment fields", ErrInvalidRecord)
		}
	}
	return nil
}

func validateProvenance(version string, record Record) error {
	if version == VersionV1 {
		if record.ProvenanceKind != "" && record.ProvenanceKind != ProvenanceEvent {
			return fmt.Errorf("%w: v1 only permits event provenance", ErrInvalidRecord)
		}
		if err := validateEventProvenance(record); err != nil {
			return err
		}
		if hasMigrationProvenance(record) {
			return fmt.Errorf("%w: v1 cannot carry migration provenance", ErrInvalidRecord)
		}
		return nil
	}

	switch record.ProvenanceKind {
	case ProvenanceEvent:
		if hasMigrationProvenance(record) {
			return fmt.Errorf("%w: event provenance cannot carry migration identity", ErrInvalidRecord)
		}
		return validateEventProvenance(record)
	case ProvenanceMigrationSeed:
		if record.SourceEventID != "" || record.SourceSequence != 0 || record.SourceSchemaVersion != 0 {
			return fmt.Errorf("%w: migration seed cannot carry event identity", ErrInvalidRecord)
		}
		if record.MigrationVersion <= 0 || strings.TrimSpace(record.MigrationName) == "" || len(record.MigrationName) > 200 || !isSHA256(record.MigrationChecksum) {
			return fmt.Errorf("%w: missing or invalid migration provenance", ErrInvalidRecord)
		}
		return nil
	default:
		return fmt.Errorf("%w: missing or unknown provenance kind", ErrInvalidRecord)
	}
}

func validateEventProvenance(record Record) error {
	if record.SourceEventID == "" || !strings.HasPrefix(record.SourceEventID, "evt_") || record.SourceSequence <= 0 || record.SourceSchemaVersion != SourceSchemaVersion {
		return fmt.Errorf("%w: missing or invalid source event provenance", ErrInvalidRecord)
	}
	return nil
}

func hasMigrationProvenance(record Record) bool {
	return record.MigrationVersion != 0 || strings.TrimSpace(record.MigrationName) != "" || record.MigrationChecksum != ""
}

func isSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func (r Record) hasAny(fields ...string) bool {
	for _, field := range fields {
		switch field {
		case "user_id":
			if r.UserID != "" {
				return true
			}
		case "organisation_id":
			if r.OrganisationID != "" {
				return true
			}
		case "workspace_id":
			if r.WorkspaceID != "" {
				return true
			}
		case "product_key":
			if r.ProductKey != "" {
				return true
			}
		case "role":
			if r.Role != "" {
				return true
			}
		case "name":
			if r.Name != "" {
				return true
			}
		}
	}
	return false
}

func CanonicalEntityKinds() []EntityKind { return AllowedEntityKinds() }

func hasPrefixID(value, prefix string) bool {
	return strings.HasPrefix(value, prefix) && len(value) > len(prefix)
}

func statusAllowed(value RecordStatus, allowed ...RecordStatus) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func validRole(value string) bool {
	return value == "owner" || value == "admin" || value == "member" || value == "viewer"
}

func validProductKey(value string) bool { return value == ProductCTRL || value == ProductIMS }

func validName(value string) bool { return value != "" && len(value) <= 200 }

// CanonicalRequest returns the exact bytes signed by a machine consumer. The
// body digest is lowercase SHA-256 hex of the raw request body, not parsed JSON.
func CanonicalRequest(method, path, bodyDigest, consumer, keyID, timestamp, nonce string) (string, error) {
	method = strings.TrimSpace(method)
	path = strings.TrimSpace(path)
	bodyDigest = strings.TrimSpace(bodyDigest)
	consumer = strings.TrimSpace(consumer)
	keyID = strings.TrimSpace(keyID)
	timestamp = strings.TrimSpace(timestamp)
	nonce = strings.TrimSpace(nonce)
	if method == "" || strings.ContainsAny(method, "\r\n") || path == "" || !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "\r\n") || !validRequestToken(consumer) || !validRequestToken(keyID) || !validRequestToken(timestamp) || !validRequestToken(nonce) || len(nonce) > NonceMaxLength || !validSHA256Hex(bodyDigest) {
		return "", ErrInvalidSignature
	}
	return strings.Join([]string{strings.ToUpper(method), path, bodyDigest, consumer, keyID, timestamp, nonce}, "\n"), nil
}

func BodyDigest(body []byte) string {
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func ResponseVersionHeader() (string, string) {
	return VersionHeader, Version
}

func IsReadAuthorityRoute(path string) bool {
	return strings.HasPrefix(path, "/api/v1/read-authority/")
}

func RequestHeaderNames() []string {
	return []string{VersionHeader, ConsumerHeader, KeyIDHeader, TimestampHeader, NonceHeader, SignatureHeader}
}

func HTTPStatus(code ErrorCode) int {
	switch code {
	case ErrorVersionMismatch:
		return http.StatusNotAcceptable
	case ErrorUnauthenticated:
		return http.StatusUnauthorized
	case ErrorForbidden:
		return http.StatusForbidden
	case ErrorInvalidRequest, ErrorLimitExceeded:
		return http.StatusBadRequest
	case ErrorStale, ErrorGap, ErrorBlocked, ErrorUnavailable, ErrorResyncRequired, ErrorCursorExpired, ErrorSnapshotExpired, ErrorSnapshotPartial, ErrorChecksumMismatch, ErrorConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func validRequestToken(value string) bool {
	return requestTokenPattern.MatchString(value)
}

func validSHA256Hex(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	return strings.ToLower(value) == value && requestTokenPattern.MatchString(value)
}
