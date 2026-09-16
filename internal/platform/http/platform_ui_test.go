package httpserver

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
)

func TestPlatformShellRendersAuthorizedContextAndProductsWithoutJavaScript(t *testing.T) {
	f := newOrganisationHTTPFixture(t)
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	workspace, _, err := f.store.CreateWorkspace(ctx, f.users[0].ID, f.organisation.ID, domain.Workspace{Name: "Operate Workspace"}, "create shell workspace", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.EntitleOrganisation(ctx, f.users[0].ID, f.organisation.ID, "product.tockrctrl", "enable shell product", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.AssignUserProduct(ctx, f.users[0].ID, f.organisation.ID, f.users[0].ID, "product.tockrctrl", "assign shell product", now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	session, csrf := loginHTTPUser(t, f, f.users[0])

	response := organisationHTTPRequest(t, f, http.MethodGet, "/", session, csrf, "", "")
	body := response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(body, "Available products") || !strings.Contains(body, "TockrCTRL") || !strings.Contains(body, "organisation-select") || !strings.Contains(body, "workspace-select") {
		t.Fatalf("launcher status/body = %d/%s", response.Code, body)
	}
	if strings.Contains(strings.ToLower(body), "<script") {
		t.Fatalf("launcher requires JavaScript: %s", body)
	}
	if !strings.Contains(body, "/launch/product.tockrctrl") {
		t.Fatalf("launcher omitted authorized product access link: %s", body)
	}

	workspacePage := organisationHTTPRequest(t, f, http.MethodGet, "/organisations/"+f.organisation.ID+"/workspaces", session, csrf, "", "")
	if workspacePage.Code != http.StatusOK || !strings.Contains(workspacePage.Body.String(), workspace.Name) {
		t.Fatalf("workspace selector status/body = %d/%s", workspacePage.Code, workspacePage.Body.String())
	}

	accessPage := organisationHTTPRequest(t, f, http.MethodGet, "/launch/product.tockrctrl?organisation_id="+f.organisation.ID+"&workspace_id="+workspace.ID, session, csrf, "", "")
	if accessPage.Code != http.StatusOK || !strings.Contains(accessPage.Body.String(), "Product access verified") {
		t.Fatalf("product access page status/body = %d/%s", accessPage.Code, accessPage.Body.String())
	}
}

func TestPlatformShellDeniesUnauthorizedContextAndProductURL(t *testing.T) {
	f := newOrganisationHTTPFixture(t)
	ownerSession, ownerCSRF := loginHTTPUser(t, f, f.users[0])
	outsiderSession, outsiderCSRF := loginHTTPUser(t, f, f.users[3])

	response := organisationHTTPRequest(t, f, http.MethodGet, "/?organisation_id="+f.organisation.ID, outsiderSession, outsiderCSRF, "", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("outsider launcher status = %d/%s", response.Code, response.Body.String())
	}

	workspace, _, err := f.store.CreateWorkspace(t.Context(), f.users[0].ID, f.organisation.ID, domain.Workspace{Name: "Denied Workspace"}, "create denied workspace", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	response = organisationHTTPRequest(t, f, http.MethodGet, "/launch/product.tockrctrl?organisation_id="+f.organisation.ID+"&workspace_id="+workspace.ID, ownerSession, ownerCSRF, "", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("unassigned product access status = %d/%s", response.Code, response.Body.String())
	}
}
