// Package readauthority contains the stable wire contract shared by Platform
// and its local projection consumers. Runtime storage and HTTP handlers belong
// to later PF-B11 slices; this package deliberately contains no persistence or
// product authorization behavior.
package readauthority

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

const (
	Version                   = "platform.read-authority.v1"
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

// AllowedEntityKinds is sorted and must be treated as immutable by callers.
var AllowedEntityKinds = []EntityKind{
	EntityUser,
	EntityOrganisation,
	EntityOrganisationMembership,
	EntityWorkspace,
	EntityWorkspaceMembership,
	EntityProduct,
	EntityOrganisationEntitlement,
	EntityUserProductAssignment,
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
// the wire contract permits them. Source provenance is mandatory for every
// record.
type Record struct {
	EntityKind          EntityKind   `json:"entity_kind"`
	ID                  string       `json:"id"`
	Status              RecordStatus `json:"status"`
	UserID              string       `json:"user_id,omitempty"`
	OrganisationID      string       `json:"organisation_id,omitempty"`
	WorkspaceID         string       `json:"workspace_id,omitempty"`
	ProductKey          string       `json:"product_key,omitempty"`
	Role                string       `json:"role,omitempty"`
	Name                string       `json:"name,omitempty"`
	SourceEventID       string       `json:"source_event_id"`
	SourceSequence      int64        `json:"source_sequence"`
	SourceSchemaVersion int          `json:"source_schema_version"`
}

// Change is a committed outbox event with a separate opaque global cursor.
// Its payload remains governed by platform-events-v1; no new event payload is
// introduced by this contract package.
type Change struct {
	Cursor string `json:"cursor"`
	Event  any    `json:"event"`
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
	if strings.TrimSpace(value) != Version {
		return ErrUnsupportedVersion
	}
	return nil
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
	for _, allowed := range AllowedEntityKinds {
		if kind == allowed {
			return nil
		}
	}
	return fmt.Errorf("%w: %q", ErrInvalidRequest, kind)
}

func ValidateSnapshotRequest(request SnapshotRequest) error {
	if len(request.EntityKinds) == 0 || len(request.EntityKinds) > len(AllowedEntityKinds) {
		return fmt.Errorf("%w: entity_kinds must contain one to %d values", ErrInvalidRequest, len(AllowedEntityKinds))
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

func SortEntityKinds(kinds []EntityKind) []EntityKind {
	copyOf := append([]EntityKind(nil), kinds...)
	sort.Slice(copyOf, func(i, j int) bool { return copyOf[i] < copyOf[j] })
	return copyOf
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
