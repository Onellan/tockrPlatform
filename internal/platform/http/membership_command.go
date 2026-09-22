package httpserver

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
	"github.com/Onellan/tockrplatform/internal/platform/assertion"
	"github.com/Onellan/tockrplatform/internal/platform/membershipcommand"
	"github.com/Onellan/tockrplatform/internal/platform/readauthority"
	"github.com/Onellan/tockrplatform/internal/store"
)

func (s *Server) membershipCommand(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	w.Header().Set(membershipcommand.VersionHeader, membershipcommand.Version)
	fail := func(code string, retryable bool) {
		writeJSON(w, membershipcommand.StatusForCode(code), membershipcommand.ErrorResponse{Version: membershipcommand.Version, Code: code, Retryable: retryable, Message: membershipCommandMessage(code)})
	}
	if strings.TrimSpace(r.Header.Get(membershipcommand.VersionHeader)) != membershipcommand.Version {
		fail("version_mismatch", false)
		return
	}
	consumer := strings.TrimSpace(r.Header.Get(membershipcommand.ConsumerHeader))
	keyID := strings.TrimSpace(r.Header.Get(membershipcommand.KeyIDHeader))
	timestamp := strings.TrimSpace(r.Header.Get(membershipcommand.TimestampHeader))
	nonce := strings.TrimSpace(r.Header.Get(membershipcommand.NonceHeader))
	signatureText := strings.TrimSpace(r.Header.Get(membershipcommand.SignatureHeader))
	actorProof := strings.TrimSpace(r.Header.Get(membershipcommand.ActorHeader))
	if !membershipcommand.ValidateConsumer(consumer) || !membershipcommand.ValidateKeyID(keyID) || !membershipcommand.ValidateTimestamp(timestamp) || !membershipcommand.ValidateNonce(nonce) || signatureText == "" {
		fail("unauthenticated", false)
		return
	}
	publicKey, ok := s.membershipCommandKeys[consumer][keyID]
	if !ok || len(publicKey) != ed25519.PublicKeySize {
		fail("unauthenticated", false)
		return
	}
	parsedTimestamp, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || parsedTimestamp < now.Unix()-membershipcommand.MaxClockSkewSecs || parsedTimestamp > now.Unix()+membershipcommand.MaxClockSkewSecs {
		fail("unauthenticated", false)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, membershipcommand.MaxBodyBytes))
	if err != nil {
		fail("invalid_request", false)
		return
	}
	path, err := canonicalReadAuthorityPath(r)
	if err != nil {
		fail("unauthenticated", false)
		return
	}
	if actorProof == "" {
		fail("unauthenticated", false)
		return
	}
	canonical, err := membershipcommand.CanonicalRequestWithActor(r.Method, path, membershipcommand.BodyDigest(body), membershipcommand.BodyDigest([]byte(actorProof)), consumer, keyID, timestamp, nonce)
	if err != nil {
		fail("unauthenticated", false)
		return
	}
	signature, err := base64.RawURLEncoding.DecodeString(signatureText)
	if err != nil || len(signature) != ed25519.SignatureSize || !ed25519.Verify(publicKey, []byte(canonical), signature) {
		fail("unauthenticated", false)
		return
	}
	if s.store == nil || s.cfg.AssertionIssuer == nil {
		fail("unavailable", true)
		return
	}
	if s.readAuthorityRate == nil || !s.readAuthorityRate.Allow("membership-command:"+consumer, now) {
		fail("limit_exceeded", true)
		return
	}
	if err := s.store.ReserveReadAuthorityNonce(r.Context(), consumer, nonce, now, now.Add(24*time.Hour)); err != nil {
		if errors.Is(err, store.ErrReadAuthorityNonceReplay) || errors.Is(err, store.ErrInvalidReadAuthorityNonce) {
			fail("unauthenticated", false)
			return
		}
		fail("unavailable", true)
		return
	}
	var request membershipcommand.Request
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		fail("invalid_request", false)
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		fail("invalid_request", false)
		return
	}
	if err := membershipcommand.ValidateRequest(request); err != nil {
		fail("invalid_request", false)
		return
	}
	audience, productKey := membershipCommandAudience(consumer)
	claims, err := s.cfg.AssertionIssuer.VerifyForProduct(actorProof, audience, productKey, now)
	if err != nil {
		switch assertion.FailureClassOf(err) {
		case assertion.FailureForbidden:
			fail("forbidden", false)
		case assertion.FailureStale:
			fail("stale", false)
		case assertion.FailureUnavailable:
			fail("unavailable", true)
		case assertion.FailureVersionMismatch:
			fail("version_mismatch", false)
		default:
			fail("unauthenticated", false)
		}
		return
	}
	if request.Scope == membershipcommand.ScopeOrganisation && claims.PlatformOrganisationID != request.OrganisationID || request.Scope == membershipcommand.ScopeWorkspace && claims.PlatformWorkspaceID != request.WorkspaceID {
		fail("forbidden", false)
		return
	}
	requestJSON, err := json.Marshal(request)
	if err != nil {
		fail("invalid_request", false)
		return
	}
	requestHash := membershipcommand.BodyDigest(requestJSON)
	result, err := s.store.ExecuteMembershipCommand(r.Context(), store.MembershipCommandRequest{Consumer: consumer, Scope: request.Scope, Operation: request.Operation, OrganisationID: request.OrganisationID, WorkspaceID: request.WorkspaceID, UserID: request.UserID, Role: request.Role, Reason: request.Reason, IdempotencyKey: request.IdempotencyKey, ExpectedVersion: request.ExpectedVersion, RequestHash: requestHash, ActorUserID: claims.PlatformUserID, OccurredAt: now})
	if err != nil {
		writeMembershipCommandStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, membershipcommand.Response{Version: membershipcommand.Version, Consumer: consumer, Scope: request.Scope, Operation: request.Operation, MembershipID: result.MembershipID, OrganisationID: result.OrganisationID, WorkspaceID: result.WorkspaceID, UserID: result.UserID, Role: result.Role, Active: result.Active, MembershipVersion: result.MembershipVersion, Replay: result.Replay})
}

func membershipCommandAudience(consumer string) (string, string) {
	if consumer == readauthority.ConsumerIMS {
		return "tockrims", readauthority.ProductIMS
	}
	return "tockrctrl", readauthority.ProductCTRL
}

func membershipCommandMessage(code string) string {
	switch code {
	case "version_mismatch":
		return "command contract version is not supported"
	case "unauthenticated":
		return "command authentication failed"
	case "forbidden":
		return "command is not authorised"
	case "invalid_request":
		return "command request is invalid"
	case "conflict":
		return "command conflicts with current membership state"
	case "stale":
		return "actor assertion is stale"
	case "limit_exceeded":
		return "command rate or size limit exceeded"
	case "unavailable":
		return "command service is unavailable"
	default:
		return "command failed"
	}
}

func writeMembershipCommandStoreError(w http.ResponseWriter, err error) {
	code, retryable := "unavailable", true
	switch {
	case errors.Is(err, store.ErrMembershipCommandConflict), errors.Is(err, store.ErrMembershipCommandReplayMismatch), errors.Is(err, store.ErrDuplicateOrganisationMember), errors.Is(err, store.ErrDuplicateWorkspaceMember):
		code, retryable = "conflict", false
	case errors.Is(err, store.ErrUnauthorisedOrganisationAction), errors.Is(err, store.ErrUnauthorisedWorkspaceAction), errors.Is(err, store.ErrOwnerMutationNotAuthorised), errors.Is(err, store.ErrOrganisationNotFound), errors.Is(err, store.ErrWorkspaceNotFound), errors.Is(err, store.ErrMembershipNotFound), errors.Is(err, store.ErrWorkspaceMembershipNotFound):
		code, retryable = "forbidden", false
	case errors.Is(err, domain.ErrInvalidOrganisationRole), errors.Is(err, domain.ErrInvalidWorkspaceRole), errors.Is(err, domain.ErrInvalidReason), errors.Is(err, store.ErrMembershipCommandUnsupported):
		code, retryable = "invalid_request", false
	}
	writeJSON(w, membershipcommand.StatusForCode(code), membershipcommand.ErrorResponse{Version: membershipcommand.Version, Code: code, Retryable: retryable, Message: membershipCommandMessage(code)})
}
