package httpserver

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Onellan/tockrplatform/internal/auth"
	"github.com/Onellan/tockrplatform/internal/db/sqlite"
	"github.com/Onellan/tockrplatform/internal/domain"
)

type authFixture struct {
	store *sqlite.Store
	user  domain.User
	h     http.Handler
}

func newAuthFixture(t *testing.T) authFixture {
	t.Helper()
	ctx := context.Background()
	store, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "platform.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	userID, err := domain.NewUserID()
	if err != nil {
		t.Fatal(err)
	}
	user, err := store.CreateUser(ctx, domain.User{ID: userID, Email: "owner@example.test", DisplayName: "Platform Owner", Active: true}, "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, Config{SessionTTL: time.Hour})
	return authFixture{store: store, user: user, h: server.Handler()}
}

func TestLoginProtectsAccountAndLogoutRequiresSessionCSRF(t *testing.T) {
	f := newAuthFixture(t)
	loginResponse := httptest.NewRecorder()
	f.h.ServeHTTP(loginResponse, httptest.NewRequest(http.MethodGet, "/login", nil))
	if loginResponse.Code != http.StatusOK || !strings.Contains(loginResponse.Body.String(), `name="csrf"`) {
		t.Fatalf("login page status/body = %d/%s", loginResponse.Code, loginResponse.Body.String())
	}
	loginCSRF := findCookie(loginResponse.Result().Cookies(), loginCSRFCookie)
	if loginCSRF == nil {
		t.Fatal("login CSRF cookie missing")
	}

	form := url.Values{"csrf": {loginCSRF.Value}, "email": {f.user.Email}, "password": {"correct horse battery staple"}}
	loginRequest := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	loginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRequest.AddCookie(loginCSRF)
	loginResponse = httptest.NewRecorder()
	f.h.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusSeeOther || loginResponse.Header().Get("Location") != "/account" {
		t.Fatalf("login status/location = %d/%s", loginResponse.Code, loginResponse.Header().Get("Location"))
	}
	sessionCookie := findCookie(loginResponse.Result().Cookies(), sessionCookieName)
	csrfCookie := findCookie(loginResponse.Result().Cookies(), "platform_csrf")
	if sessionCookie == nil || csrfCookie == nil || !sessionCookie.HttpOnly || !sessionCookie.Secure || csrfCookie.Secure != sessionCookie.Secure || sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookies = %#v %#v", sessionCookie, csrfCookie)
	}
	loginAudits, err := f.store.SecurityEventCount(context.Background(), "login")
	if err != nil || loginAudits != 1 {
		t.Fatalf("login audit count = %d, err=%v", loginAudits, err)
	}

	accountRequest := httptest.NewRequest(http.MethodGet, "/account", nil)
	accountRequest.AddCookie(sessionCookie)
	accountRequest.AddCookie(csrfCookie)
	accountResponse := httptest.NewRecorder()
	f.h.ServeHTTP(accountResponse, accountRequest)
	stored, err := f.store.FindUserByID(context.Background(), f.user.ID)
	if err != nil || stored == nil {
		t.Fatalf("read stored user: %v", err)
	}
	if accountResponse.Code != http.StatusOK || !strings.Contains(accountResponse.Body.String(), f.user.ID) || strings.Contains(accountResponse.Body.String(), stored.PasswordHash) {
		t.Fatalf("account status/body = %d/%s", accountResponse.Code, accountResponse.Body.String())
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/logout", strings.NewReader(url.Values{"csrf": {csrfCookie.Value}}.Encode()))
	logoutRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	logoutRequest.AddCookie(sessionCookie)
	logoutRequest.AddCookie(csrfCookie)
	logoutResponse := httptest.NewRecorder()
	f.h.ServeHTTP(logoutResponse, logoutRequest)
	if logoutResponse.Code != http.StatusSeeOther || logoutResponse.Header().Get("Location") != "/login" {
		t.Fatalf("logout status/location = %d/%s", logoutResponse.Code, logoutResponse.Header().Get("Location"))
	}
	if _, err := f.store.AuthenticatedSession(context.Background(), sessionCookie.Value, time.Now().UTC()); err != sql.ErrNoRows {
		t.Fatalf("revoked session error = %v", err)
	}
	count, err := f.store.SecurityEventCount(context.Background(), "logout")
	if err != nil || count != 1 {
		t.Fatalf("logout audit count = %d, err=%v", count, err)
	}
}

func TestLoginDeniesUnknownInactiveAndMissingCSRFWithoutDisclosure(t *testing.T) {
	f := newAuthFixture(t)
	response := httptest.NewRecorder()
	f.h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/login", nil))
	csrf := findCookie(response.Result().Cookies(), loginCSRFCookie)
	unknown := url.Values{"csrf": {csrf.Value}, "email": {"nobody@example.test"}, "password": {"wrong"}}
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(unknown.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(csrf)
	response = httptest.NewRecorder()
	f.h.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/login?error=1" {
		t.Fatalf("unknown login status/location = %d/%s", response.Code, response.Header().Get("Location"))
	}
	count, err := f.store.SecurityEventCount(context.Background(), "failed_login")
	if err != nil || count != 1 {
		t.Fatalf("unknown login audit count = %d, err=%v", count, err)
	}

	missingCSRF := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(url.Values{"email": {f.user.Email}, "password": {"correct horse battery staple"}}.Encode()))
	missingCSRF.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response = httptest.NewRecorder()
	f.h.ServeHTTP(response, missingCSRF)
	if response.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF status = %d", response.Code)
	}

	if err := f.store.SetUserActive(context.Background(), f.user.ID, false); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	f.h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/login", nil))
	csrf = findCookie(response.Result().Cookies(), loginCSRFCookie)
	inactive := url.Values{"csrf": {csrf.Value}, "email": {f.user.Email}, "password": {"correct horse battery staple"}}
	request = httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(inactive.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(csrf)
	response = httptest.NewRecorder()
	f.h.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/login?error=1" {
		t.Fatalf("inactive login status/location = %d/%s", response.Code, response.Header().Get("Location"))
	}
}

func TestProtectedRouteRedirectsWithoutSessionAndGetLogoutIsNotAllowed(t *testing.T) {
	f := newAuthFixture(t)
	response := httptest.NewRecorder()
	f.h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/account", nil))
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/login" {
		t.Fatalf("unauthenticated account = %d/%s", response.Code, response.Header().Get("Location"))
	}
	response = httptest.NewRecorder()
	f.h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/logout", nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET logout status = %d", response.Code)
	}
}

func TestLoginRateLimitBlocksRepeatedFailures(t *testing.T) {
	f := newAuthFixture(t)
	server := NewServer(f.store, Config{
		RateLimitEnabled: true,
		LoginLimiter: auth.LoginLimiterConfig{
			FailureThreshold: 2,
			BaseBackoff:      time.Minute,
			MaxBackoff:       time.Minute,
			EntryTTL:         time.Hour,
			MaxEntries:       10,
		},
	})
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/login", nil))
	csrf := findCookie(response.Result().Cookies(), loginCSRFCookie)
	for attempt := 0; attempt < 2; attempt++ {
		form := url.Values{"csrf": {csrf.Value}, "email": {f.user.Email}, "password": {"wrong password here"}}
		request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(csrf)
		response = httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
	}
	if response.Code != http.StatusTooManyRequests || response.Header().Get("Retry-After") == "" {
		t.Fatalf("rate-limit response = %d, retry-after=%q", response.Code, response.Header().Get("Retry-After"))
	}
}

func TestInactiveUserCannotContinueAnExistingSession(t *testing.T) {
	f := newAuthFixture(t)
	loginResponse := httptest.NewRecorder()
	f.h.ServeHTTP(loginResponse, httptest.NewRequest(http.MethodGet, "/login", nil))
	loginCSRF := findCookie(loginResponse.Result().Cookies(), loginCSRFCookie)
	form := url.Values{"csrf": {loginCSRF.Value}, "email": {f.user.Email}, "password": {"correct horse battery staple"}}
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(loginCSRF)
	loginResponse = httptest.NewRecorder()
	f.h.ServeHTTP(loginResponse, request)
	session := findCookie(loginResponse.Result().Cookies(), sessionCookieName)
	csrf := findCookie(loginResponse.Result().Cookies(), "platform_csrf")
	if err := f.store.SetUserActive(context.Background(), f.user.ID, false); err != nil {
		t.Fatal(err)
	}
	account := httptest.NewRequest(http.MethodGet, "/account", nil)
	account.AddCookie(session)
	account.AddCookie(csrf)
	response := httptest.NewRecorder()
	f.h.ServeHTTP(response, account)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/login" {
		t.Fatalf("inactive session status/location = %d/%s", response.Code, response.Header().Get("Location"))
	}
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
