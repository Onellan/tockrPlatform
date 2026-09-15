package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
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
	if err := json.Unmarshal(response.Body.Bytes(), &assignment); err != nil || !assignment.Active || !assignment.EffectiveActive || assignment.UserID != f.users[2].ID {
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
