package httpserver

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
	platformconfig "github.com/Onellan/tockrplatform/internal/platform/config"
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
	if !strings.Contains(body, `method="get" action="/"`) || !strings.Contains(body, `name="organisation_id"`) || !strings.Contains(body, `name="workspace_id"`) {
		t.Fatalf("launcher context selection requires JavaScript: %s", body)
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

func TestPlatformBrowserHandoffUsesConfiguredTargetAudienceAndCSRF(t *testing.T) {
	f := newOrganisationHTTPFixture(t)
	issuer := testHTTPAssertionIssuer(t)
	target := platformconfig.ProductLaunchTarget{URL: "http://localhost:8029/auth/platform/session", Audience: "tockrctrl"}
	f.h = NewServer(f.store, Config{
		SessionTTL:      time.Hour,
		AssertionIssuer: issuer,
		ProductLaunchTargets: map[string]platformconfig.ProductLaunchTarget{
			"product.tockrctrl": target,
		},
	}).Handler()
	workspace, _, err := f.store.CreateWorkspace(t.Context(), f.users[0].ID, f.organisation.ID, domain.Workspace{Name: "Browser Handoff Workspace"}, "create browser handoff workspace", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.EntitleOrganisation(t.Context(), f.users[0].ID, f.organisation.ID, "product.tockrctrl", "enable browser handoff", time.Now().UTC().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.AssignUserProduct(t.Context(), f.users[0].ID, f.organisation.ID, f.users[0].ID, "product.tockrctrl", "assign browser handoff", time.Now().UTC().Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	ownerSession, ownerCSRF := loginHTTPUser(t, f, f.users[0])
	outsiderSession, outsiderCSRF := loginHTTPUser(t, f, f.users[3])

	launcher := organisationHTTPRequest(t, f, http.MethodGet, "/", ownerSession, ownerCSRF, "", "")
	if launcher.Code != http.StatusOK || !strings.Contains(launcher.Body.String(), `method="post" action="/launch/product.tockrctrl/handoff"`) || !strings.Contains(launcher.Header().Get("Content-Security-Policy"), "form-action 'self' http://localhost:8029") {
		t.Fatalf("configured launcher = %d, CSP %q, body %s", launcher.Code, launcher.Header().Get("Content-Security-Policy"), launcher.Body.String())
	}
	form := url.Values{"organisation_id": {f.organisation.ID}, "workspace_id": {workspace.ID}, "audience": {"tockrims"}}
	response := organisationFormRequest(t, f, http.MethodPost, "/launch/product.tockrctrl/handoff", ownerSession, ownerCSRF, form)
	if response.Code != http.StatusForbidden {
		t.Fatalf("missing browser handoff CSRF = %d/%s", response.Code, response.Body.String())
	}
	response = organisationFormRequest(t, f, http.MethodPost, "/launch/product.tockrctrl/handoff", outsiderSession, outsiderCSRF, url.Values{
		"csrf": {outsiderCSRF.Value}, "organisation_id": {f.organisation.ID}, "workspace_id": {workspace.ID},
	})
	if response.Code != http.StatusNotFound {
		t.Fatalf("unauthorized browser handoff = %d/%s", response.Code, response.Body.String())
	}
	form.Set("csrf", ownerCSRF.Value)
	response = organisationFormRequest(t, f, http.MethodPost, "/launch/product.tockrctrl/handoff", ownerSession, ownerCSRF, form)
	body := response.Body.String()
	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" || !strings.Contains(body, `action="http://localhost:8029/auth/platform/session"`) {
		t.Fatalf("browser handoff page = %d, headers %#v, body %s", response.Code, response.Header(), body)
	}
	match := regexp.MustCompile(`name="assertion" value="([^"]+)"`).FindStringSubmatch(body)
	if len(match) != 2 {
		t.Fatalf("browser handoff assertion missing: %s", body)
	}
	claims, err := issuer.Verify(match[1], time.Now().UTC())
	if err != nil || claims.Audience != "tockrctrl" || claims.PlatformUserID != f.users[0].ID || claims.PlatformOrganisationID != f.organisation.ID || claims.PlatformWorkspaceID != workspace.ID {
		t.Fatalf("browser handoff claims = %#v, err=%v", claims, err)
	}
	unauthorizedTarget := organisationFormRequest(t, f, http.MethodPost, "/launch/product.tockrims/handoff", ownerSession, ownerCSRF, form)
	if unauthorizedTarget.Code != http.StatusNotFound {
		t.Fatalf("unconfigured product handoff = %d/%s", unauthorizedTarget.Code, unauthorizedTarget.Body.String())
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
