package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthAndReadinessAreDistinctSafeAndBounded(t *testing.T) {
	readinessErr := errors.New("database details must stay internal")
	server := NewServer(nil, Config{
		ReadinessCheck:      func(_ context.Context) error { return readinessErr },
		MaxRequestBodyBytes: 16,
	})
	handler := server.Handler()

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK || health.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatalf("health = %d/%q", health.Code, health.Body.String())
	}
	if health.Header().Get("Content-Security-Policy") == "" || health.Header().Get("Permissions-Policy") == "" {
		t.Fatalf("security headers = %#v", health.Header())
	}

	ready := httptest.NewRecorder()
	handler.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if ready.Code != http.StatusServiceUnavailable || ready.Body.String() != "{\"status\":\"not_ready\"}\n" {
		t.Fatalf("ready = %d/%q", ready.Code, ready.Body.String())
	}
	if strings.Contains(ready.Body.String(), "database details") {
		t.Fatal("readiness response disclosed internal error details")
	}

	large := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(strings.Repeat("x", 32)))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handler.ServeHTTP(large, request)
	if large.Code != http.StatusBadRequest {
		t.Fatalf("bounded request = %d/%q", large.Code, large.Body.String())
	}
}

func TestReadinessReturnsReadyOnlyAfterDependencyCheck(t *testing.T) {
	server := NewServer(nil, Config{ReadinessCheck: func(_ context.Context) error { return nil }})
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusOK || response.Body.String() != "{\"status\":\"ready\"}\n" {
		t.Fatalf("ready = %d/%q", response.Code, response.Body.String())
	}
}
