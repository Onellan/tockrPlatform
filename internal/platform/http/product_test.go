package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

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
