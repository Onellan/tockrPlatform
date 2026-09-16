package httpserver

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
	"github.com/Onellan/tockrplatform/internal/platform/assertion"
)

func TestProductHTTPAuthorisationCSRFAndRedaction(t *testing.T) {
	f := newOrganisationHTTPFixture(t)
	ownerSession, ownerCSRF := loginHTTPUser(t, f, f.users[0])
	adminSession, adminCSRF := loginHTTPUser(t, f, f.users[1])
	memberSession, memberCSRF := loginHTTPUser(t, f, f.users[2])
	outsiderSession, outsiderCSRF := loginHTTPUser(t, f, f.users[3])
	systemSession, systemCSRF := loginHTTPUser(t, f, f.users[4])
	organisationPath := "/api/organisations/" + f.organisation.ID

	response := organisationHTTPRequest(t, f, http.MethodGet, "/api/platform/products", ownerSession, ownerCSRF, "", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("owner catalogue response = %d/%s, want safe not found", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodGet, "/api/platform/products", systemSession, systemCSRF, "", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "product.tockrctrl") || !strings.Contains(response.Body.String(), "product.tockrims") {
		t.Fatalf("system catalogue response = %d/%s", response.Code, response.Body.String())
	}
	var products []productResponse
	if err := json.Unmarshal(response.Body.Bytes(), &products); err != nil || len(products) != 2 {
		t.Fatalf("catalogue read model = %#v, err=%v", response.Body.String(), err)
	}

	body := `{"product_key":"product.tockrctrl","reason":"enable CTRL for organisation"}`
	response = organisationHTTPRequest(t, f, http.MethodPost, organisationPath+"/product-entitlements", ownerSession, nil, body, "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF entitlement response = %d/%s, want forbidden", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodPost, organisationPath+"/product-entitlements", ownerSession, ownerCSRF, body, "csrf-header")
	if response.Code != http.StatusCreated || strings.Contains(response.Body.String(), "password") || strings.Contains(response.Body.String(), "billing") || strings.Contains(response.Body.String(), "payment") || strings.Contains(response.Body.String(), "product_role") {
		t.Fatalf("owner entitlement response = %d/%s", response.Code, response.Body.String())
	}
	var entitlement organisationProductEntitlementResponse
	if err := json.Unmarshal(response.Body.Bytes(), &entitlement); err != nil || entitlement.ProductKey != "product.tockrctrl" || entitlement.Status != domain.OrganisationProductEntitlementActive || !entitlement.Active {
		t.Fatalf("entitlement response = %#v, err=%v", response.Body.String(), err)
	}

	response = organisationHTTPRequest(t, f, http.MethodGet, organisationPath+"/product-entitlements", memberSession, memberCSRF, "", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("member entitlement read = %d/%s, want safe not found", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodPost, organisationPath+"/product-entitlements", memberSession, memberCSRF, `{"product_key":"product.tockrims","reason":"member attempt"}`, "csrf-header")
	if response.Code != http.StatusNotFound {
		t.Fatalf("member entitlement write = %d/%s, want safe not found", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodGet, organisationPath+"/product-entitlements", adminSession, adminCSRF, "", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), entitlement.ID) {
		t.Fatalf("admin entitlement read = %d/%s", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodDelete, organisationPath+"/product-entitlements/"+entitlement.ID, adminSession, adminCSRF, `{"reason":"disable CTRL for organisation"}`, "csrf-header")
	if response.Code != http.StatusNoContent {
		t.Fatalf("admin entitlement revoke = %d/%s", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodGet, organisationPath+"/product-entitlements", outsiderSession, outsiderCSRF, "", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("outsider entitlement read = %d/%s, want safe not found", response.Code, response.Body.String())
	}

	response = organisationHTTPRequest(t, f, http.MethodPost, "/api/platform/products/product.tockrims/retire", systemSession, systemCSRF, `{"reason":"retire product"}`, "csrf-header")
	if response.Code != http.StatusNoContent {
		t.Fatalf("system product retirement = %d/%s", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodPost, "/api/platform/products/product.tockrctrl/retire", ownerSession, ownerCSRF, `{"reason":"owner attempt"}`, "csrf-header")
	if response.Code != http.StatusNotFound {
		t.Fatalf("owner product retirement = %d/%s, want safe not found", response.Code, response.Body.String())
	}
}

func TestProductAccessHTTPUsesCentralEvaluatorAndFailsClosed(t *testing.T) {
	f := newOrganisationHTTPFixture(t)
	adminSession, adminCSRF := loginHTTPUser(t, f, f.users[1])
	memberSession, memberCSRF := loginHTTPUser(t, f, f.users[2])
	now := time.Now().UTC().Truncate(time.Second)
	workspace, _, err := f.store.CreateWorkspace(context.Background(), f.users[0].ID, f.organisation.ID, domain.Workspace{Name: "HTTP Access Workspace"}, "create HTTP access workspace", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.AddWorkspaceMember(context.Background(), f.users[0].ID, workspace.ID, f.users[2].ID, domain.WorkspaceMember, "grant HTTP access workspace", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.EntitleOrganisation(context.Background(), f.users[0].ID, f.organisation.ID, "product.tockrctrl", "enable HTTP CTRL", now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	organisationPath := "/api/organisations/" + f.organisation.ID
	accessPath := organisationPath + "/product-access/product.tockrctrl/workspaces/" + workspace.ID
	response := organisationHTTPRequest(t, f, http.MethodGet, accessPath, memberSession, memberCSRF, "", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("unassigned product access = %d/%s, want safe not found", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodPost, organisationPath+"/product-assignments", adminSession, adminCSRF, `{"user_id":"`+f.users[2].ID+`","product_key":"product.tockrctrl","reason":"assign HTTP CTRL"}`, "csrf-header")
	if response.Code != http.StatusCreated || strings.Contains(response.Body.String(), "billing") || strings.Contains(response.Body.String(), "payment") || strings.Contains(response.Body.String(), "product_role") {
		t.Fatalf("HTTP assignment = %d/%s", response.Code, response.Body.String())
	}
	var assignment userProductAssignmentResponse
	if err := json.Unmarshal(response.Body.Bytes(), &assignment); err != nil || !assignment.Active || assignment.UserID != f.users[2].ID {
		t.Fatalf("assignment response = %#v, err=%v", response.Body.String(), err)
	}
	response = organisationHTTPRequest(t, f, http.MethodGet, accessPath, memberSession, memberCSRF, "", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"allowed":true`) || !strings.Contains(response.Body.String(), workspace.ID) {
		t.Fatalf("effective HTTP access = %d/%s", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodGet, organisationPath+"/product-assignments", memberSession, memberCSRF, "", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("member assignment enumeration = %d/%s, want safe not found", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodDelete, organisationPath+"/product-assignments/"+assignment.ID, adminSession, adminCSRF, `{"reason":"revoke HTTP CTRL"}`, "csrf-header")
	if response.Code != http.StatusNoContent {
		t.Fatalf("HTTP assignment revoke = %d/%s", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodGet, accessPath, memberSession, memberCSRF, "", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("revoked HTTP access = %d/%s, want safe not found", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodPost, organisationPath+"/product-assignments", adminSession, nil, `{"user_id":"`+f.users[2].ID+`","product_key":"product.tockrctrl","reason":"missing CSRF"}`, "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("missing assignment CSRF = %d/%s, want forbidden", response.Code, response.Body.String())
	}
}

func TestProductAssertionHTTPRequiresEffectiveAccessAndProtectsContract(t *testing.T) {
	f := newOrganisationHTTPFixture(t)
	issuer := testHTTPAssertionIssuer(t)
	f.h = NewServer(f.store, Config{SessionTTL: time.Hour, AssertionIssuer: issuer}).Handler()
	adminSession, adminCSRF := loginHTTPUser(t, f, f.users[1])
	memberSession, memberCSRF := loginHTTPUser(t, f, f.users[2])
	outsiderSession, outsiderCSRF := loginHTTPUser(t, f, f.users[3])
	now := time.Now().UTC().Truncate(time.Second)
	workspace, _, err := f.store.CreateWorkspace(context.Background(), f.users[0].ID, f.organisation.ID, domain.Workspace{Name: "Assertion Workspace"}, "create assertion workspace", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.AddWorkspaceMember(context.Background(), f.users[0].ID, workspace.ID, f.users[2].ID, domain.WorkspaceMember, "grant assertion workspace", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.EntitleOrganisation(context.Background(), f.users[0].ID, f.organisation.ID, "product.tockrctrl", "enable assertion CTRL", now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	organisationPath := "/api/organisations/" + f.organisation.ID
	assertionPath := organisationPath + "/product-assertions/product.tockrctrl/workspaces/" + workspace.ID

	response := organisationHTTPRequest(t, f, http.MethodGet, "/.well-known/tockr-platform-assertion-keys", nil, nil, "", "")
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "private") {
		t.Fatalf("assertion keys response = %d/%s", response.Code, response.Body.String())
	}
	var keys []assertionKeyResponse
	if err := json.Unmarshal(response.Body.Bytes(), &keys); err != nil || len(keys) != 1 || keys[0].KeyID != "key-http" || keys[0].Algorithm != assertion.Algorithm || keys[0].PublicKey == "" {
		t.Fatalf("assertion public keys = %#v, err=%v", response.Body.String(), err)
	}

	response = organisationHTTPRequest(t, f, http.MethodPost, assertionPath, memberSession, nil, `{"audience":"tockrctrl"}`, "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("missing assertion CSRF = %d/%s, want forbidden", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodPost, assertionPath, memberSession, memberCSRF, `{"audience":"tockrctrl"}`, "csrf-header")
	if response.Code != http.StatusNotFound {
		t.Fatalf("unassigned assertion = %d/%s, want safe not found", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodPost, organisationPath+"/product-assignments", adminSession, adminCSRF, `{"user_id":"`+f.users[2].ID+`","product_key":"product.tockrctrl","reason":"assign assertion CTRL"}`, "csrf-header")
	if response.Code != http.StatusCreated {
		t.Fatalf("assertion assignment = %d/%s", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodPost, assertionPath, outsiderSession, outsiderCSRF, `{"audience":"tockrctrl"}`, "csrf-header")
	if response.Code != http.StatusNotFound {
		t.Fatalf("outsider assertion = %d/%s, want safe not found", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodPost, assertionPath, memberSession, memberCSRF, `{"audience":"unknown"}`, "csrf-header")
	if response.Code != http.StatusBadRequest || strings.Contains(response.Body.String(), "product_role") || strings.Contains(response.Body.String(), "billing") {
		t.Fatalf("unknown audience assertion = %d/%s", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodPost, assertionPath, memberSession, memberCSRF, `{"audience":"tockrctrl"}`, "csrf-header")
	if response.Code != http.StatusCreated {
		t.Fatalf("effective assertion = %d/%s", response.Code, response.Body.String())
	}
	var issued productAssertionResponse
	if err := json.Unmarshal(response.Body.Bytes(), &issued); err != nil || issued.Assertion == "" || issued.AssertionVersion != assertion.CurrentVersion {
		t.Fatalf("issued assertion response = %#v, err=%v", response.Body.String(), err)
	}
	claims, err := issuer.Verify(issued.Assertion, issued.IssuedAt.Add(time.Second))
	if err != nil || claims.PlatformUserID != f.users[2].ID || claims.PlatformOrganisationID != f.organisation.ID || claims.PlatformWorkspaceID != workspace.ID || claims.Audience != "tockrctrl" {
		t.Fatalf("issued assertion claims = %#v, err=%v", claims, err)
	}
	if strings.Contains(response.Body.String(), "product_role") || strings.Contains(response.Body.String(), "billing") || strings.Contains(response.Body.String(), "password") {
		t.Fatalf("issued assertion response leaked prohibited claim: %s", response.Body.String())
	}
}

func testHTTPAssertionIssuer(t *testing.T) *assertion.Issuer {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	issuer, err := assertion.New(assertion.Config{
		Issuer:           "https://platform.example",
		ActiveKeyID:      "key-http",
		ActivePrivateKey: privateKey,
		VerificationKeys: map[string]ed25519.PublicKey{"key-http": publicKey},
		Audiences:        []string{"tockrctrl", "tockrims"},
		MaxLifetime:      5 * time.Minute,
		ClockSkew:        5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	return issuer
}
