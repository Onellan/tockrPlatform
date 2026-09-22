// Package membershipcommand defines the machine-authenticated membership
// mutation contract consumed by CTRL and IMS. It contains no persistence or
// product-role authority.
package membershipcommand

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

const (
	Version           = "platform.membership-command.v1"
	VersionHeader     = "X-Tockr-Platform-Membership-Command-Version"
	ConsumerHeader    = "X-Tockr-Platform-Consumer"
	KeyIDHeader       = "X-Tockr-Platform-Key-ID"
	TimestampHeader   = "X-Tockr-Platform-Timestamp"
	NonceHeader       = "X-Tockr-Platform-Nonce"
	SignatureHeader   = "X-Tockr-Platform-Signature"
	ActorHeader       = "X-Tockr-Platform-Actor-Assertion"
	ConsumerCTRL      = "tockrctrl"
	ConsumerIMS       = "tockrims"
	OperationAdd      = "add"
	OperationRole     = "change_role"
	OperationRemove   = "deactivate"
	ScopeOrganisation = "organisation_membership"
	ScopeWorkspace    = "workspace_membership"
	MaxReason         = 500
	MaxIdempotencyKey = 128
	MaxClockSkewSecs  = int64(300)
	MaxBodyBytes      = 64 * 1024
)

var tokenPattern = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)

var (
	ErrInvalidRequest   = errors.New("invalid membership command request")
	ErrInvalidSignature = errors.New("invalid membership command signature input")
	ErrConflict         = errors.New("membership command conflict")
)

// Request is the authenticated mutation body. Consumer and actor identity
// are bound by headers and the Platform assertion, never trusted from JSON.
type Request struct {
	Scope           string `json:"scope"`
	Operation       string `json:"operation"`
	OrganisationID  string `json:"organisation_id,omitempty"`
	WorkspaceID     string `json:"workspace_id,omitempty"`
	UserID          string `json:"user_id"`
	Role            string `json:"role,omitempty"`
	Reason          string `json:"reason"`
	IdempotencyKey  string `json:"idempotency_key"`
	ExpectedVersion int64  `json:"expected_version"`
}

type Response struct {
	Version           string `json:"version"`
	Consumer          string `json:"consumer"`
	Scope             string `json:"scope"`
	Operation         string `json:"operation"`
	MembershipID      string `json:"membership_id"`
	OrganisationID    string `json:"organisation_id,omitempty"`
	WorkspaceID       string `json:"workspace_id,omitempty"`
	UserID            string `json:"user_id"`
	Role              string `json:"role,omitempty"`
	Active            bool   `json:"active"`
	MembershipVersion int64  `json:"membership_version"`
	Replay            bool   `json:"replay,omitempty"`
}

type ErrorResponse struct {
	Version   string `json:"version"`
	Code      string `json:"code"`
	Retryable bool   `json:"retryable"`
	Message   string `json:"message"`
}

func ValidateRequest(r Request) error {
	if r.Scope != ScopeOrganisation && r.Scope != ScopeWorkspace {
		return fmt.Errorf("%w: scope", ErrInvalidRequest)
	}
	if r.Operation != OperationAdd && r.Operation != OperationRole && r.Operation != OperationRemove {
		return fmt.Errorf("%w: operation", ErrInvalidRequest)
	}
	if !validID(r.UserID, "usr_") || (!validID(r.OrganisationID, "org_") && r.Scope == ScopeOrganisation) || (!validID(r.WorkspaceID, "wsp_") && r.Scope == ScopeWorkspace) {
		return fmt.Errorf("%w: target identity", ErrInvalidRequest)
	}
	if r.Scope == ScopeOrganisation && r.WorkspaceID != "" || r.Scope == ScopeWorkspace && r.OrganisationID != "" {
		return fmt.Errorf("%w: mixed scope", ErrInvalidRequest)
	}
	if len(strings.TrimSpace(r.Reason)) == 0 || len(r.Reason) > MaxReason || strings.ContainsAny(r.Reason, "\r\n") {
		return fmt.Errorf("%w: reason", ErrInvalidRequest)
	}
	if !validToken(r.IdempotencyKey) || len(r.IdempotencyKey) > MaxIdempotencyKey {
		return fmt.Errorf("%w: idempotency key", ErrInvalidRequest)
	}
	if r.Operation == OperationAdd {
		if r.ExpectedVersion != 0 || strings.TrimSpace(r.Role) == "" {
			return fmt.Errorf("%w: add version or role", ErrInvalidRequest)
		}
	} else if r.ExpectedVersion <= 0 || (r.Operation == OperationRole && strings.TrimSpace(r.Role) == "") || (r.Operation == OperationRemove && strings.TrimSpace(r.Role) != "") {
		return fmt.Errorf("%w: mutation version or role", ErrInvalidRequest)
	}
	return nil
}

func CanonicalRequest(method, path, bodyDigest, consumer, keyID, timestamp, nonce string) (string, error) {
	if strings.TrimSpace(method) == "" || !strings.HasPrefix(path, "/") || !validToken(consumer) || !validToken(keyID) || !validToken(timestamp) || !validToken(nonce) || !validSHA256(bodyDigest) {
		return "", ErrInvalidSignature
	}
	return strings.Join([]string{strings.ToUpper(strings.TrimSpace(method)), strings.TrimSpace(path), strings.TrimSpace(bodyDigest), strings.TrimSpace(consumer), strings.TrimSpace(keyID), strings.TrimSpace(timestamp), strings.TrimSpace(nonce)}, "\n"), nil
}

// CanonicalRequestWithActor binds the separately transported actor proof to
// the product signature. The proof is represented by its SHA-256 digest so a
// signed request never places the assertion itself in the canonical string.
func CanonicalRequestWithActor(method, path, bodyDigest, actorDigest, consumer, keyID, timestamp, nonce string) (string, error) {
	if !validSHA256(actorDigest) {
		return "", ErrInvalidSignature
	}
	base, err := CanonicalRequest(method, path, bodyDigest, consumer, keyID, timestamp, nonce)
	if err != nil {
		return "", err
	}
	return strings.Join([]string{base, strings.TrimSpace(actorDigest)}, "\n"), nil
}

func BodyDigest(body []byte) string { sum := sha256.Sum256(body); return hex.EncodeToString(sum[:]) }

func ValidateConsumer(value string) bool  { return value == ConsumerCTRL || value == ConsumerIMS }
func ValidateKeyID(value string) bool     { return validToken(value) && len(value) <= 100 }
func ValidateNonce(value string) bool     { return validToken(value) && len(value) <= 128 }
func ValidateTimestamp(value string) bool { return validToken(value) && len(value) <= 20 }
func StatusForCode(code string) int {
	switch code {
	case "version_mismatch":
		return http.StatusNotAcceptable
	case "unauthenticated":
		return http.StatusUnauthorized
	case "forbidden":
		return http.StatusForbidden
	case "invalid_request":
		return http.StatusBadRequest
	case "conflict":
		return http.StatusConflict
	case "stale":
		return http.StatusConflict
	case "limit_exceeded":
		return http.StatusTooManyRequests
	case "unavailable":
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
func validToken(value string) bool { return value != "" && tokenPattern.MatchString(value) }
func validID(value, prefix string) bool {
	return strings.HasPrefix(value, prefix) && len(value) > len(prefix) && len(value) <= 200 && !strings.ContainsAny(value, "\r\n \t")
}
func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
