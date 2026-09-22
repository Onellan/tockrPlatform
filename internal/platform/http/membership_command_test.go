package httpserver

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Onellan/tockrplatform/internal/platform/assertion"
	"github.com/Onellan/tockrplatform/internal/platform/membershipcommand"
	"github.com/Onellan/tockrplatform/internal/platform/readauthority"
)

func TestMembershipCommandHTTPAuthenticatesActorAndReplaysIdempotently(t *testing.T) {
	f := newOrganisationHTTPFixture(t)
	servicePublic, servicePrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	issuer := testHTTPAssertionIssuer(t)
	server := NewServer(f.store, Config{AllowInsecureCookies: true, AssertionIssuer: issuer, MembershipCommandKeys: readauthority.PublicKeySet{readauthority.ConsumerCTRL: {"current": servicePublic}}})
	h := server.Handler()
	now := time.Now().UTC()
	token, _, err := issuer.Issue(assertion.IssueRequest{Audience: "tockrctrl", PlatformUserID: f.users[0].ID, OrganisationID: f.organisation.ID, WorkspaceID: "wsp_command_scope"}, now)
	if err != nil {
		t.Fatal(err)
	}
	requestBody := membershipCommandTestBody(t, map[string]any{"scope": membershipcommand.ScopeOrganisation, "operation": membershipcommand.OperationAdd, "organisation_id": f.organisation.ID, "user_id": f.users[3].ID, "role": "member", "reason": "approved command", "idempotency_key": "command-add-1", "expected_version": 0})
	first := signedMembershipCommandTestRequest(t, requestBody, token, servicePrivate, "nonce-command-1")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, first)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"active":true`) {
		t.Fatalf("first command = %d/%s", response.Code, response.Body.String())
	}
	var result struct {
		MembershipID string `json:"membership_id"`
		Replay       bool   `json:"replay"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || result.MembershipID == "" || result.Replay {
		t.Fatalf("first result = %#v, err=%v", result, err)
	}
	replayToken, _, err := issuer.Issue(assertion.IssueRequest{Audience: "tockrctrl", PlatformUserID: f.users[0].ID, OrganisationID: f.organisation.ID, WorkspaceID: "wsp_command_scope"}, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	replay := signedMembershipCommandTestRequest(t, requestBody, replayToken, servicePrivate, "nonce-command-2")
	replayResponse := httptest.NewRecorder()
	h.ServeHTTP(replayResponse, replay)
	if replayResponse.Code != http.StatusOK || !strings.Contains(replayResponse.Body.String(), `"replay":true`) {
		t.Fatalf("replay command = %d/%s", replayResponse.Code, replayResponse.Body.String())
	}
	badActorToken, _, err := issuer.Issue(assertion.IssueRequest{Audience: "tockrctrl", PlatformUserID: f.users[2].ID, OrganisationID: f.organisation.ID, WorkspaceID: "wsp_command_scope"}, now)
	if err != nil {
		t.Fatal(err)
	}
	denied := signedMembershipCommandTestRequest(t, requestBody, badActorToken, servicePrivate, "nonce-command-3")
	deniedResponse := httptest.NewRecorder()
	h.ServeHTTP(deniedResponse, denied)
	if deniedResponse.Code != http.StatusForbidden {
		t.Fatalf("member actor command = %d/%s", deniedResponse.Code, deniedResponse.Body.String())
	}
	tamperToken, _, err := issuer.Issue(assertion.IssueRequest{Audience: "tockrctrl", PlatformUserID: f.users[2].ID, OrganisationID: f.organisation.ID, WorkspaceID: "wsp_command_scope"}, now.Add(1500*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	tampered := signedMembershipCommandTestRequest(t, requestBody, token, servicePrivate, "nonce-command-tampered")
	tampered.Header.Set(membershipcommand.ActorHeader, tamperToken)
	tamperedResponse := httptest.NewRecorder()
	h.ServeHTTP(tamperedResponse, tampered)
	if tamperedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("tampered actor proof = %d/%s", tamperedResponse.Code, tamperedResponse.Body.String())
	}
	roleBody := membershipCommandTestBody(t, map[string]any{"scope": membershipcommand.ScopeOrganisation, "operation": membershipcommand.OperationRole, "organisation_id": f.organisation.ID, "user_id": f.users[3].ID, "role": "admin", "reason": "approved promotion", "idempotency_key": "command-role-1", "expected_version": 1})
	roleToken, _, err := issuer.Issue(assertion.IssueRequest{Audience: "tockrctrl", PlatformUserID: f.users[0].ID, OrganisationID: f.organisation.ID, WorkspaceID: "wsp_command_scope"}, now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	roleResponse := httptest.NewRecorder()
	h.ServeHTTP(roleResponse, signedMembershipCommandTestRequest(t, roleBody, roleToken, servicePrivate, "nonce-command-4"))
	if roleResponse.Code != http.StatusOK || !strings.Contains(roleResponse.Body.String(), `"membership_version":2`) {
		t.Fatalf("role command = %d/%s", roleResponse.Code, roleResponse.Body.String())
	}
	staleBody := membershipCommandTestBody(t, map[string]any{"scope": membershipcommand.ScopeOrganisation, "operation": membershipcommand.OperationRemove, "organisation_id": f.organisation.ID, "user_id": f.users[3].ID, "reason": "stale removal", "idempotency_key": "command-remove-stale", "expected_version": 1})
	staleToken, _, err := issuer.Issue(assertion.IssueRequest{Audience: "tockrctrl", PlatformUserID: f.users[0].ID, OrganisationID: f.organisation.ID, WorkspaceID: "wsp_command_scope"}, now.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	staleResponse := httptest.NewRecorder()
	h.ServeHTTP(staleResponse, signedMembershipCommandTestRequest(t, staleBody, staleToken, servicePrivate, "nonce-command-5"))
	if staleResponse.Code != http.StatusConflict {
		t.Fatalf("stale command = %d/%s", staleResponse.Code, staleResponse.Body.String())
	}
}

func membershipCommandTestBody(t *testing.T, value map[string]any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func signedMembershipCommandTestRequest(t *testing.T, body []byte, actor string, privateKey ed25519.PrivateKey, nonce string) *http.Request {
	t.Helper()
	timestamp := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	canonical, err := membershipcommand.CanonicalRequestWithActor(http.MethodPost, "/api/v1/membership-commands", membershipcommand.BodyDigest(body), membershipcommand.BodyDigest([]byte(actor)), membershipcommand.ConsumerCTRL, "current", timestamp, nonce)
	if err != nil {
		t.Fatal(err)
	}
	signature := ed25519.Sign(privateKey, []byte(canonical))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/membership-commands", strings.NewReader(string(body)))
	request.Header.Set(membershipcommand.VersionHeader, membershipcommand.Version)
	request.Header.Set(membershipcommand.ConsumerHeader, membershipcommand.ConsumerCTRL)
	request.Header.Set(membershipcommand.KeyIDHeader, "current")
	request.Header.Set(membershipcommand.TimestampHeader, timestamp)
	request.Header.Set(membershipcommand.NonceHeader, nonce)
	request.Header.Set(membershipcommand.SignatureHeader, base64.RawURLEncoding.EncodeToString(signature))
	request.Header.Set(membershipcommand.ActorHeader, actor)
	return request
}
