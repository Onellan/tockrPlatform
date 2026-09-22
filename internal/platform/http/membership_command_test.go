package httpserver

import (
	"context"
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

	"github.com/Onellan/tockrplatform/internal/domain"
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
	var payload string
	if err := f.store.DB().QueryRowContext(context.Background(), `SELECT payload FROM platform_outbox WHERE aggregate_id=? ORDER BY sequence DESC LIMIT 1`, f.organisation.ID).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(payload, "approved command") || strings.Contains(payload, "approved promotion") {
		t.Fatalf("membership command reason leaked into outbox payload: %s", payload)
	}
}

func TestMembershipCommandHTTPSupportsIMSWorkspaceAndFailsClosedAcrossScopes(t *testing.T) {
	f := newOrganisationHTTPFixture(t)
	workspace, _, err := f.store.CreateWorkspace(context.Background(), f.users[0].ID, f.organisation.ID, domain.Workspace{Name: "Command Workspace", Status: domain.WorkspaceActive}, "create command workspace", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	servicePublic, servicePrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	issuer := testHTTPAssertionIssuer(t)
	server := NewServer(f.store, Config{AllowInsecureCookies: true, AssertionIssuer: issuer, MembershipCommandKeys: readauthority.PublicKeySet{readauthority.ConsumerIMS: {"current": servicePublic}}})
	h := server.Handler()
	now := time.Now().UTC()
	actor := func(at time.Time) string {
		token, _, issueErr := issuer.Issue(assertion.IssueRequest{Audience: "tockrims", PlatformUserID: f.users[0].ID, OrganisationID: f.organisation.ID, WorkspaceID: workspace.ID}, at)
		if issueErr != nil {
			t.Fatal(issueErr)
		}
		return token
	}
	addBody := membershipCommandTestBody(t, map[string]any{"scope": membershipcommand.ScopeWorkspace, "operation": membershipcommand.OperationAdd, "workspace_id": workspace.ID, "user_id": f.users[2].ID, "role": "member", "reason": "IMS workspace add", "idempotency_key": "ims-workspace-add-1", "expected_version": 0})
	addResponse := httptest.NewRecorder()
	h.ServeHTTP(addResponse, signedMembershipCommandTestRequestForConsumer(t, addBody, actor(now), servicePrivate, "ims-workspace-add-nonce", membershipcommand.ConsumerIMS))
	if addResponse.Code != http.StatusOK || !strings.Contains(addResponse.Body.String(), `"membership_version":1`) {
		t.Fatalf("IMS workspace add = %d/%s", addResponse.Code, addResponse.Body.String())
	}
	alteredBody := membershipCommandTestBody(t, map[string]any{"scope": membershipcommand.ScopeWorkspace, "operation": membershipcommand.OperationAdd, "workspace_id": workspace.ID, "user_id": f.users[2].ID, "role": "viewer", "reason": "altered replay", "idempotency_key": "ims-workspace-add-1", "expected_version": 0})
	alteredResponse := httptest.NewRecorder()
	h.ServeHTTP(alteredResponse, signedMembershipCommandTestRequestForConsumer(t, alteredBody, actor(now.Add(time.Second)), servicePrivate, "ims-workspace-altered-nonce", membershipcommand.ConsumerIMS))
	if alteredResponse.Code != http.StatusConflict {
		t.Fatalf("altered idempotency replay = %d/%s", alteredResponse.Code, alteredResponse.Body.String())
	}
	roleBody := membershipCommandTestBody(t, map[string]any{"scope": membershipcommand.ScopeWorkspace, "operation": membershipcommand.OperationRole, "workspace_id": workspace.ID, "user_id": f.users[2].ID, "role": "admin", "reason": "IMS workspace role", "idempotency_key": "ims-workspace-role-1", "expected_version": 1})
	roleResponse := httptest.NewRecorder()
	h.ServeHTTP(roleResponse, signedMembershipCommandTestRequestForConsumer(t, roleBody, actor(now.Add(2*time.Second)), servicePrivate, "ims-workspace-role-nonce", membershipcommand.ConsumerIMS))
	if roleResponse.Code != http.StatusOK || !strings.Contains(roleResponse.Body.String(), `"membership_version":2`) {
		t.Fatalf("IMS workspace role change = %d/%s", roleResponse.Code, roleResponse.Body.String())
	}
	deactivateBody := membershipCommandTestBody(t, map[string]any{"scope": membershipcommand.ScopeWorkspace, "operation": membershipcommand.OperationRemove, "workspace_id": workspace.ID, "user_id": f.users[2].ID, "reason": "IMS workspace deactivate", "idempotency_key": "ims-workspace-remove-1", "expected_version": 2})
	deactivateResponse := httptest.NewRecorder()
	h.ServeHTTP(deactivateResponse, signedMembershipCommandTestRequestForConsumer(t, deactivateBody, actor(now.Add(3*time.Second)), servicePrivate, "ims-workspace-remove-nonce", membershipcommand.ConsumerIMS))
	if deactivateResponse.Code != http.StatusOK || !strings.Contains(deactivateResponse.Body.String(), `"active":false`) {
		t.Fatalf("IMS workspace deactivate = %d/%s", deactivateResponse.Code, deactivateResponse.Body.String())
	}
	if _, err := f.store.AddWorkspaceMember(context.Background(), f.users[0].ID, workspace.ID, f.users[1].ID, domain.WorkspaceMember, "guarded workspace member", now); err != nil {
		t.Fatal(err)
	}
	if err := f.store.DeactivateOrganisationMember(context.Background(), f.users[0].ID, f.organisation.ID, f.users[1].ID, "remove parent membership", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	guardedBody := membershipCommandTestBody(t, map[string]any{"scope": membershipcommand.ScopeWorkspace, "operation": membershipcommand.OperationRole, "workspace_id": workspace.ID, "user_id": f.users[1].ID, "role": "admin", "reason": "cross scope role", "idempotency_key": "ims-workspace-guarded-1", "expected_version": 1})
	guardedResponse := httptest.NewRecorder()
	h.ServeHTTP(guardedResponse, signedMembershipCommandTestRequestForConsumer(t, guardedBody, actor(now.Add(4*time.Second)), servicePrivate, "ims-workspace-guarded-nonce", membershipcommand.ConsumerIMS))
	if guardedResponse.Code != http.StatusForbidden {
		t.Fatalf("cross-scope workspace role change = %d/%s", guardedResponse.Code, guardedResponse.Body.String())
	}
	var active int
	if err := f.store.DB().QueryRowContext(context.Background(), `SELECT active FROM workspace_memberships WHERE workspace_id=(SELECT id FROM workspaces WHERE public_id=?) AND user_id=(SELECT id FROM users WHERE public_id=?) ORDER BY id DESC LIMIT 1`, workspace.ID, f.users[1].ID).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if active != 1 {
		t.Fatalf("cross-scope rejection changed workspace membership active state: %d", active)
	}
	outOfScopeBody := membershipCommandTestBody(t, map[string]any{"scope": membershipcommand.ScopeOrganisation, "operation": "assign_product", "organisation_id": f.organisation.ID, "user_id": f.users[3].ID, "reason": "out of scope", "idempotency_key": "ims-out-of-scope-1", "expected_version": 0})
	outOfScopeResponse := httptest.NewRecorder()
	h.ServeHTTP(outOfScopeResponse, signedMembershipCommandTestRequestForConsumer(t, outOfScopeBody, actor(now.Add(5*time.Second)), servicePrivate, "ims-out-of-scope-nonce", membershipcommand.ConsumerIMS))
	if outOfScopeResponse.Code != http.StatusBadRequest {
		t.Fatalf("out-of-scope command = %d/%s", outOfScopeResponse.Code, outOfScopeResponse.Body.String())
	}
}

func TestMembershipCommandHTTPRollsBackWhenOutboxAppendFails(t *testing.T) {
	f := newOrganisationHTTPFixture(t)
	servicePublic, servicePrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	issuer := testHTTPAssertionIssuer(t)
	server := NewServer(f.store, Config{AllowInsecureCookies: true, AssertionIssuer: issuer, MembershipCommandKeys: readauthority.PublicKeySet{readauthority.ConsumerCTRL: {"current": servicePublic}}})
	now := time.Now().UTC()
	actor, _, err := issuer.Issue(assertion.IssueRequest{Audience: "tockrctrl", PlatformUserID: f.users[0].ID, OrganisationID: f.organisation.ID, WorkspaceID: "wsp_command_scope"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.DB().ExecContext(context.Background(), `CREATE TRIGGER reject_command_platform_events BEFORE INSERT ON platform_outbox BEGIN SELECT RAISE(ABORT,'reject command event'); END`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.store.DB().ExecContext(context.Background(), `DROP TRIGGER IF EXISTS reject_command_platform_events`)
	})
	body := membershipCommandTestBody(t, map[string]any{"scope": membershipcommand.ScopeOrganisation, "operation": membershipcommand.OperationAdd, "organisation_id": f.organisation.ID, "user_id": f.users[3].ID, "role": "member", "reason": "rollback command", "idempotency_key": "rollback-command-1", "expected_version": 0})
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, signedMembershipCommandTestRequest(t, body, actor, servicePrivate, "rollback-command-nonce"))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("outbox failure response = %d/%s", response.Code, response.Body.String())
	}
	var active, audit, results int
	if err := f.store.DB().QueryRowContext(context.Background(), `SELECT COUNT(*) FROM organisation_memberships m JOIN users u ON u.id=m.user_id WHERE m.organisation_id=(SELECT id FROM organisations WHERE public_id=?) AND u.public_id=? AND m.active=1`, f.organisation.ID, f.users[3].ID).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if err := f.store.DB().QueryRowContext(context.Background(), `SELECT COUNT(*) FROM audit_events WHERE aggregate_id=? AND event='organisation_membership_added' AND details LIKE ?`, f.organisation.ID, "%"+f.users[3].ID+"%").Scan(&audit); err != nil {
		t.Fatal(err)
	}
	if err := f.store.DB().QueryRowContext(context.Background(), `SELECT COUNT(*) FROM platform_membership_command_results WHERE idempotency_key=?`, "rollback-command-1").Scan(&results); err != nil {
		t.Fatal(err)
	}
	if active != 0 || audit != 0 || results != 0 {
		t.Fatalf("outbox rollback left active=%d audit=%d result=%d", active, audit, results)
	}
}

func TestMembershipCommandHTTPRejectsInactiveStaleAndCrossOrganisationActors(t *testing.T) {
	f := newOrganisationHTTPFixture(t)
	servicePublic, servicePrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	issuer := testHTTPAssertionIssuer(t)
	server := NewServer(f.store, Config{AllowInsecureCookies: true, AssertionIssuer: issuer, MembershipCommandKeys: readauthority.PublicKeySet{readauthority.ConsumerCTRL: {"current": servicePublic}}})
	h := server.Handler()
	now := time.Now().UTC()
	issue := func(userID, organisationID string, at time.Time) string {
		token, _, issueErr := issuer.Issue(assertion.IssueRequest{Audience: "tockrctrl", PlatformUserID: userID, OrganisationID: organisationID, WorkspaceID: "wsp_command_scope"}, at)
		if issueErr != nil {
			t.Fatal(issueErr)
		}
		return token
	}
	request := func(body []byte, actor, nonce string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		h.ServeHTTP(response, signedMembershipCommandTestRequest(t, body, actor, servicePrivate, nonce))
		return response
	}
	if err := f.store.SetUserActive(context.Background(), f.users[1].ID, false); err != nil {
		t.Fatal(err)
	}
	inactiveBody := membershipCommandTestBody(t, map[string]any{"scope": membershipcommand.ScopeOrganisation, "operation": membershipcommand.OperationAdd, "organisation_id": f.organisation.ID, "user_id": f.users[3].ID, "role": "member", "reason": "inactive actor", "idempotency_key": "inactive-actor-1", "expected_version": 0})
	if response := request(inactiveBody, issue(f.users[1].ID, f.organisation.ID, now), "inactive-actor-nonce"); response.Code != http.StatusForbidden {
		t.Fatalf("inactive actor response = %d/%s", response.Code, response.Body.String())
	}
	staleBody := membershipCommandTestBody(t, map[string]any{"scope": membershipcommand.ScopeOrganisation, "operation": membershipcommand.OperationAdd, "organisation_id": f.organisation.ID, "user_id": f.users[3].ID, "role": "member", "reason": "stale actor", "idempotency_key": "stale-actor-1", "expected_version": 0})
	if response := request(staleBody, issue(f.users[0].ID, f.organisation.ID, now.Add(-10*time.Minute)), "stale-actor-nonce"); response.Code != http.StatusConflict {
		t.Fatalf("stale actor response = %d/%s", response.Code, response.Body.String())
	}
	other, _, err := f.store.CreateOrganisation(context.Background(), f.users[3].ID, domain.Organisation{Name: "Other command organisation", Status: domain.OrganisationActive}, "create other command organisation", now)
	if err != nil {
		t.Fatal(err)
	}
	crossBody := membershipCommandTestBody(t, map[string]any{"scope": membershipcommand.ScopeOrganisation, "operation": membershipcommand.OperationAdd, "organisation_id": other.ID, "user_id": f.users[2].ID, "role": "member", "reason": "cross organisation", "idempotency_key": "cross-organisation-1", "expected_version": 0})
	if response := request(crossBody, issue(f.users[0].ID, f.organisation.ID, now), "cross-organisation-nonce"); response.Code != http.StatusForbidden {
		t.Fatalf("cross-organisation actor response = %d/%s", response.Code, response.Body.String())
	}
	var membershipCount int
	if err := f.store.DB().QueryRowContext(context.Background(), `SELECT COUNT(*) FROM organisation_memberships WHERE organisation_id=(SELECT id FROM organisations WHERE public_id=?) AND user_id=(SELECT id FROM users WHERE public_id=?) AND active=1`, other.ID, f.users[2].ID).Scan(&membershipCount); err != nil {
		t.Fatal(err)
	}
	if membershipCount != 0 {
		t.Fatalf("cross-organisation denial changed membership count=%d", membershipCount)
	}
}

func TestMembershipCommandHTTPMapsArchivedScopeToNonRetryableForbidden(t *testing.T) {
	f := newOrganisationHTTPFixture(t)
	servicePublic, servicePrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	issuer := testHTTPAssertionIssuer(t)
	server := NewServer(f.store, Config{AllowInsecureCookies: true, AssertionIssuer: issuer, MembershipCommandKeys: readauthority.PublicKeySet{readauthority.ConsumerCTRL: {"current": servicePublic}}})
	now := time.Now().UTC()
	actor, _, err := issuer.Issue(assertion.IssueRequest{Audience: "tockrctrl", PlatformUserID: f.users[0].ID, OrganisationID: f.organisation.ID, WorkspaceID: "wsp_command_scope"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.store.ArchiveOrganisation(context.Background(), f.users[0].ID, f.organisation.ID, "archive scope", now); err != nil {
		t.Fatal(err)
	}
	body := membershipCommandTestBody(t, map[string]any{"scope": membershipcommand.ScopeOrganisation, "operation": membershipcommand.OperationAdd, "organisation_id": f.organisation.ID, "user_id": f.users[3].ID, "role": "member", "reason": "archived scope", "idempotency_key": "archived-scope-1", "expected_version": 0})
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, signedMembershipCommandTestRequest(t, body, actor, servicePrivate, "archived-scope-nonce"))
	if response.Code != http.StatusForbidden || strings.Contains(response.Body.String(), `"retryable":true`) {
		t.Fatalf("archived scope response = %d/%s", response.Code, response.Body.String())
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
	return signedMembershipCommandTestRequestForConsumer(t, body, actor, privateKey, nonce, membershipcommand.ConsumerCTRL)
}

func signedMembershipCommandTestRequestForConsumer(t *testing.T, body []byte, actor string, privateKey ed25519.PrivateKey, nonce, consumer string) *http.Request {
	t.Helper()
	timestamp := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	canonical, err := membershipcommand.CanonicalRequestWithActor(http.MethodPost, "/api/v1/membership-commands", membershipcommand.BodyDigest(body), membershipcommand.BodyDigest([]byte(actor)), consumer, "current", timestamp, nonce)
	if err != nil {
		t.Fatal(err)
	}
	signature := ed25519.Sign(privateKey, []byte(canonical))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/membership-commands", strings.NewReader(string(body)))
	request.Header.Set(membershipcommand.VersionHeader, membershipcommand.Version)
	request.Header.Set(membershipcommand.ConsumerHeader, consumer)
	request.Header.Set(membershipcommand.KeyIDHeader, "current")
	request.Header.Set(membershipcommand.TimestampHeader, timestamp)
	request.Header.Set(membershipcommand.NonceHeader, nonce)
	request.Header.Set(membershipcommand.SignatureHeader, base64.RawURLEncoding.EncodeToString(signature))
	request.Header.Set(membershipcommand.ActorHeader, actor)
	return request
}
