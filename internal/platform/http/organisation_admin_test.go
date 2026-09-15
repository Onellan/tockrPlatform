package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Onellan/tockrplatform/internal/db/sqlite"
	"github.com/Onellan/tockrplatform/internal/domain"
)

type organisationHTTPFixture struct {
	store        *sqlite.Store
	users        []domain.User
	organisation domain.Organisation
	h            http.Handler
}

func newOrganisationHTTPFixture(t *testing.T) organisationHTTPFixture {
	t.Helper()
	ctx := context.Background()
	store, err := sqlite.OpenWithKey(ctx, filepath.Join(t.TempDir(), "platform.db"), httpMFAKey)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	users := make([]domain.User, 0, 5)
	for i, email := range []string{"organisation-owner@example.test", "organisation-admin@example.test", "organisation-member@example.test", "organisation-outsider@example.test", "organisation-system@example.test"} {
		id, err := domain.NewUserID()
		if err != nil {
			t.Fatal(err)
		}
		user, err := store.CreateUser(ctx, domain.User{ID: id, Email: email, DisplayName: "Organisation Test User " + string(rune('A'+i)), Active: true}, "correct horse battery staple")
		if err != nil {
			t.Fatal(err)
		}
		users = append(users, user)
	}
	var systemInternalID int64
	if err := store.DB().QueryRowContext(ctx, `SELECT id FROM users WHERE public_id=?`, users[4].ID).Scan(&systemInternalID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB().ExecContext(ctx, `INSERT INTO system_role_assignments(user_id,role,active,assigned_by,assigned_at) VALUES(?,?,1,?,?)`, systemInternalID, "system_admin", systemInternalID, formatHTTPTestTime(time.Now().UTC())); err != nil {
		t.Fatal(err)
	}
	organisationID, err := domain.NewOrganisationID()
	if err != nil {
		t.Fatal(err)
	}
	organisation, _, err := store.CreateOrganisation(ctx, users[0].ID, domain.Organisation{ID: organisationID, Name: "HTTP Organisation", Status: domain.OrganisationActive}, "create HTTP organisation", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[1].ID, domain.OrganisationAdmin, "appoint HTTP admin", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[2].ID, domain.OrganisationMember, "add HTTP member", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, Config{SessionTTL: time.Hour})
	return organisationHTTPFixture{store: store, users: users, organisation: organisation, h: server.Handler()}
}

func TestOrganisationHTTPAuthorizationCSRFRedactionAndSafeScopeErrors(t *testing.T) {
	f := newOrganisationHTTPFixture(t)
	ownerSession, ownerCSRF := loginHTTPUser(t, f, f.users[0])
	adminSession, adminCSRF := loginHTTPUser(t, f, f.users[1])
	memberSession, memberCSRF := loginHTTPUser(t, f, f.users[2])
	outsiderSession, outsiderCSRF := loginHTTPUser(t, f, f.users[3])
	systemSession, systemCSRF := loginHTTPUser(t, f, f.users[4])
	path := "/api/organisations/" + f.organisation.ID

	response := organisationHTTPRequest(t, f, http.MethodGet, path, memberSession, memberCSRF, "", "")
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "password_hash") || strings.Contains(response.Body.String(), "product_role") {
		t.Fatalf("member organisation response = %d/%s", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodGet, path+"/workspace-entry", memberSession, memberCSRF, "", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"available":false`) || strings.Contains(response.Body.String(), "product_role") {
		t.Fatalf("workspace entry response = %d/%s", response.Code, response.Body.String())
	}

	response = organisationHTTPRequest(t, f, http.MethodGet, path+"/members", memberSession, memberCSRF, "", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("member enumeration response = %d/%s, want safe not found", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodGet, path+"/members", adminSession, adminCSRF, "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("admin member response = %d/%s", response.Code, response.Body.String())
	}
	var members []organisationMemberResponse
	if err := json.Unmarshal(response.Body.Bytes(), &members); err != nil || len(members) != 3 {
		t.Fatalf("admin members = %#v, err=%v", response.Body.String(), err)
	}
	response = organisationHTTPRequest(t, f, http.MethodGet, path+"/members", outsiderSession, outsiderCSRF, "", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("outsider member response = %d/%s, want safe not found", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodGet, path, systemSession, systemCSRF, "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("system organisation response = %d/%s", response.Code, response.Body.String())
	}

	addBody := `{"user_id":"` + f.users[3].ID + `","role":"member","reason":"invite through API"}`
	response = organisationHTTPRequest(t, f, http.MethodPost, path+"/members", adminSession, nil, addBody, "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF response = %d/%s", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodPost, path+"/members", memberSession, memberCSRF, addBody, "csrf-header")
	if response.Code != http.StatusNotFound {
		t.Fatalf("member mutation response = %d/%s, want safe not found", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodPost, path+"/members", adminSession, adminCSRF, addBody, "csrf-header")
	if response.Code != http.StatusCreated || strings.Contains(response.Body.String(), "product_role") {
		t.Fatalf("admin mutation response = %d/%s", response.Code, response.Body.String())
	}

	renameBody := `{"name":"Renamed HTTP Organisation","reason":"correct general settings"}`
	response = organisationHTTPRequest(t, f, http.MethodPatch, path, memberSession, memberCSRF, renameBody, "csrf-header")
	if response.Code != http.StatusNotFound {
		t.Fatalf("member rename response = %d/%s, want safe not found", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodPatch, path, ownerSession, ownerCSRF, renameBody, "csrf-header")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Renamed HTTP Organisation") {
		t.Fatalf("owner rename response = %d/%s", response.Code, response.Body.String())
	}

	response = organisationHTTPRequest(t, f, http.MethodGet, path+"/audit", memberSession, memberCSRF, "", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("member audit response = %d/%s, want safe not found", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodGet, path+"/audit?limit=10", ownerSession, ownerCSRF, "", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "organisation_created") || strings.Contains(response.Body.String(), "password_hash") || strings.Contains(response.Body.String(), "product_role") {
		t.Fatalf("owner audit response = %d/%s", response.Code, response.Body.String())
	}

	otherID, err := domain.NewOrganisationID()
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.store.CreateOrganisation(context.Background(), f.users[3].ID, domain.Organisation{ID: otherID, Name: "Other HTTP Organisation", Status: domain.OrganisationActive}, "create other HTTP organisation", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	response = organisationHTTPRequest(t, f, http.MethodGet, "/api/organisations/"+otherID, ownerSession, ownerCSRF, "", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("cross-organisation response = %d/%s, want safe not found", response.Code, response.Body.String())
	}
}

func TestWorkspaceHTTPAuthorizationLifecycleAndDefaultEntry(t *testing.T) {
	f := newOrganisationHTTPFixture(t)
	ownerSession, ownerCSRF := loginHTTPUser(t, f, f.users[0])
	memberSession, memberCSRF := loginHTTPUser(t, f, f.users[2])
	outsiderSession, outsiderCSRF := loginHTTPUser(t, f, f.users[3])
	organisationPath := "/api/organisations/" + f.organisation.ID

	createBody := `{"name":"HTTP Workspace","reason":"create workspace through API"}`
	response := organisationHTTPRequest(t, f, http.MethodPost, organisationPath+"/workspaces", ownerSession, ownerCSRF, createBody, "csrf-header")
	if response.Code != http.StatusCreated || strings.Contains(response.Body.String(), "product_role") {
		t.Fatalf("workspace create response = %d/%s", response.Code, response.Body.String())
	}
	var workspace workspaceResponse
	if err := json.Unmarshal(response.Body.Bytes(), &workspace); err != nil || workspace.ID == "" || workspace.Status != domain.WorkspaceActive {
		t.Fatalf("created workspace = %s, err=%v", response.Body.String(), err)
	}
	if !strings.HasPrefix(workspace.ID, "wsp_") || workspace.OrganisationID != f.organisation.ID {
		t.Fatalf("created workspace identity = %#v", workspace)
	}

	response = organisationHTTPRequest(t, f, http.MethodGet, organisationPath+"/workspace-entry", ownerSession, ownerCSRF, "", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"available":true`) || !strings.Contains(response.Body.String(), workspace.ID) || !strings.Contains(response.Body.String(), `"default_workspace_id":"`+workspace.ID+`"`) {
		t.Fatalf("workspace entry response = %d/%s", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodGet, organisationPath+"/workspaces", memberSession, memberCSRF, "", "")
	if response.Code != http.StatusOK || response.Body.String() != "[]\n" {
		t.Fatalf("organisation member workspace list = %d/%s, want empty", response.Code, response.Body.String())
	}

	addBody := `{"user_id":"` + f.users[2].ID + `","role":"member","reason":"grant workspace access"}`
	response = organisationHTTPRequest(t, f, http.MethodPost, "/api/workspaces/"+workspace.ID+"/members", ownerSession, ownerCSRF, addBody, "csrf-header")
	if response.Code != http.StatusCreated {
		t.Fatalf("workspace member create response = %d/%s", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodGet, "/api/workspaces/"+workspace.ID, memberSession, memberCSRF, "", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), workspace.ID) || strings.Contains(response.Body.String(), "password_hash") {
		t.Fatalf("workspace member read response = %d/%s", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodGet, "/api/workspaces/"+workspace.ID+"/members", memberSession, memberCSRF, "", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("workspace member enumeration response = %d/%s, want safe not found", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodGet, "/api/workspaces/"+workspace.ID+"/members", ownerSession, ownerCSRF, "", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), f.users[2].ID) {
		t.Fatalf("workspace admin member enumeration response = %d/%s", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodGet, "/api/workspaces/"+workspace.ID, outsiderSession, outsiderCSRF, "", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("cross-organisation workspace response = %d/%s, want safe not found", response.Code, response.Body.String())
	}

	response = organisationHTTPRequest(t, f, http.MethodPost, "/api/workspaces/"+workspace.ID+"/archive", memberSession, memberCSRF, `{"reason":"member archive attempt"}`, "csrf-header")
	if response.Code != http.StatusNotFound {
		t.Fatalf("workspace member archive response = %d/%s, want safe not found", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodPost, "/api/workspaces/"+workspace.ID+"/archive", ownerSession, ownerCSRF, `{"reason":"archive workspace"}`, "csrf-header")
	if response.Code != http.StatusNoContent {
		t.Fatalf("workspace archive response = %d/%s", response.Code, response.Body.String())
	}
	response = organisationHTTPRequest(t, f, http.MethodGet, "/api/workspaces/"+workspace.ID, ownerSession, ownerCSRF, "", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("archived workspace HTTP read = %d/%s, want safe not found", response.Code, response.Body.String())
	}
}

func loginHTTPUser(t *testing.T, f organisationHTTPFixture, user domain.User) (*http.Cookie, *http.Cookie) {
	t.Helper()
	page := httptest.NewRecorder()
	f.h.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/login", nil))
	loginCSRF := findCookie(page.Result().Cookies(), loginCSRFCookie)
	if loginCSRF == nil {
		t.Fatal("login CSRF cookie missing")
	}
	form := url.Values{"csrf": {loginCSRF.Value}, "email": {user.Email}, "password": {"correct horse battery staple"}}
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(loginCSRF)
	response := httptest.NewRecorder()
	f.h.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("login status = %d/%s", response.Code, response.Body.String())
	}
	session := findCookie(response.Result().Cookies(), sessionCookieName)
	csrf := findCookie(response.Result().Cookies(), "platform_csrf")
	if session == nil || csrf == nil {
		t.Fatal("authenticated cookies missing")
	}
	return session, csrf
}

func organisationHTTPRequest(t *testing.T, f organisationHTTPFixture, method, path string, session, csrf *http.Cookie, body, csrfMode string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if csrfMode == "csrf-header" && csrf != nil {
		request.Header.Set("X-CSRF-Token", csrf.Value)
	}
	if session != nil {
		request.AddCookie(session)
	}
	if csrf != nil {
		request.AddCookie(csrf)
	}
	response := httptest.NewRecorder()
	f.h.ServeHTTP(response, request)
	return response
}

func formatHTTPTestTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
