package assertion

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestIssueAndVerifyAllowlistedSharedClaims(t *testing.T) {
	issuer, now := testIssuer(t, "key-2026-09")
	token, claims, err := issuer.Issue(IssueRequest{
		Audience:       "tockrctrl",
		PlatformUserID: "usr_123",
		OrganisationID: "org_456",
		WorkspaceID:    "wsp_789",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(token, ".") != 2 {
		t.Fatalf("assertion segments = %d, want 3", strings.Count(token, ".")+1)
	}
	if claims.Issuer != "https://platform.example" || claims.Audience != "tockrctrl" || claims.AssertionVersion != CurrentVersion {
		t.Fatalf("issued claims = %#v", claims)
	}
	if claims.PlatformUserID != "usr_123" || claims.PlatformOrganisationID != "org_456" || claims.PlatformWorkspaceID != "wsp_789" {
		t.Fatalf("issued scope = %#v", claims)
	}
	if !claims.ExpiresAt.After(claims.IssuedAt) || claims.ExpiresAt.Sub(claims.IssuedAt) != DefaultLifetime {
		t.Fatalf("issued lifetime = %s", claims.ExpiresAt.Sub(claims.IssuedAt))
	}
	verified, err := issuer.Verify(token, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if verified != claims {
		t.Fatalf("verified claims = %#v, want %#v", verified, claims)
	}
	if strings.Contains(token, "product_role") || strings.Contains(token, "billing") {
		t.Fatal("assertion contains prohibited product or billing claim text")
	}
}

func TestAssertionRejectsTamperingExpiryAudienceAndScope(t *testing.T) {
	issuer, now := testIssuer(t, "key-current")
	token, _, err := issuer.Issue(IssueRequest{Audience: "tockrims", PlatformUserID: "usr_1", OrganisationID: "org_1", WorkspaceID: "wsp_1"}, now)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")
	parts[1] = base64.RawURLEncoding.EncodeToString([]byte(`{"issuer":"https://platform.example","audience":"tockrims","platform_user_id":"usr_1","platform_organisation_id":"org_1","platform_workspace_id":"wsp_1","issued_at":"2026-09-16T12:00:00Z","expires_at":"2026-09-16T12:02:00Z","assertion_id":"ast_tampered","assertion_version":1,"product_role":"owner"}`))
	parts[2] = base64.RawURLEncoding.EncodeToString(ed25519.Sign(issuer.privateKey, []byte(parts[0]+"."+parts[1])))
	if _, err := issuer.Verify(strings.Join(parts, "."), now); !errors.Is(err, ErrInvalidAssertion) {
		t.Fatalf("unknown claim verification error = %v, want invalid assertion", err)
	}
	if _, err := issuer.Verify(token, now.Add(DefaultLifetime+time.Second)); !errors.Is(err, ErrAssertionExpired) {
		t.Fatalf("expired assertion error = %v, want expired", err)
	}
	if _, _, err := issuer.Issue(IssueRequest{Audience: "unknown", PlatformUserID: "usr_1", OrganisationID: "org_1", WorkspaceID: "wsp_1"}, now); !errors.Is(err, ErrInvalidAudience) {
		t.Fatalf("unknown audience issue error = %v, want invalid audience", err)
	}
	if _, _, err := issuer.Issue(IssueRequest{Audience: "tockrctrl", PlatformUserID: "user_1", OrganisationID: "org_1", WorkspaceID: "wsp_1"}, now); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("invalid user scope issue error = %v, want invalid scope", err)
	}
}

func TestAssertionReplayAndKeyRotation(t *testing.T) {
	oldIssuer, now := testIssuer(t, "key-old")
	oldToken, _, err := oldIssuer.Issue(IssueRequest{Audience: "tockrctrl", PlatformUserID: "usr_1", OrganisationID: "org_1", WorkspaceID: "wsp_1"}, now)
	if err != nil {
		t.Fatal(err)
	}
	newPublic, newPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	oldKeys := oldIssuer.PublicKeys()
	rotated, err := New(Config{
		Issuer:           "https://platform.example",
		ActiveKeyID:      "key-new",
		ActivePrivateKey: newPrivate,
		VerificationKeys: map[string]ed25519.PublicKey{"key-old": oldKeys[0].PublicKey, "key-new": newPublic},
		Audiences:        []string{"tockrctrl", "tockrims"},
		MaxLifetime:      5 * time.Minute,
		ClockSkew:        5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rotated.Verify(oldToken, now.Add(time.Second)); err != nil {
		t.Fatalf("old key verification after rotation = %v", err)
	}
	if _, err := rotated.Verify(oldToken, now.Add(time.Second)); !errors.Is(err, ErrAssertionReplay) {
		t.Fatalf("replayed assertion error = %v, want replay", err)
	}
	newToken, _, err := rotated.Issue(IssueRequest{Audience: "tockrims", PlatformUserID: "usr_2", OrganisationID: "org_2", WorkspaceID: "wsp_2"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rotated.Verify(newToken, now.Add(time.Second)); err != nil {
		t.Fatalf("new key verification = %v", err)
	}
	keys := rotated.PublicKeys()
	if len(keys) != 2 || keys[0].KeyID != "key-new" || keys[1].KeyID != "key-old" {
		t.Fatalf("rotated public keys = %#v", keys)
	}
}

func TestConsumerVerifierNeedsOnlyPublicKeys(t *testing.T) {
	issuer, now := testIssuer(t, "key-consumer")
	token, _, err := issuer.Issue(IssueRequest{Audience: "tockrctrl", PlatformUserID: "usr_1", OrganisationID: "org_1", WorkspaceID: "wsp_1"}, now)
	if err != nil {
		t.Fatal(err)
	}
	publicKeys := issuer.PublicKeys()
	verifier, err := NewVerifier(VerifierConfig{
		Issuer:           "https://platform.example",
		VerificationKeys: map[string]ed25519.PublicKey{"key-consumer": publicKeys[0].PublicKey},
		Audiences:        []string{"tockrctrl"},
		MaxLifetime:      5 * time.Minute,
		ClockSkew:        5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.Verify(token, now.Add(time.Second)); err != nil {
		t.Fatalf("public-key-only consumer verification = %v", err)
	}
}

func TestConfigFromEnvironmentRequiresExplicitKeyMaterial(t *testing.T) {
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{
		"PLATFORM_ASSERTION_ISSUER":      "https://platform.example",
		"PLATFORM_ASSERTION_KEY_ID":      "key-current",
		"PLATFORM_ASSERTION_PRIVATE_KEY": hex.EncodeToString(private),
		"PLATFORM_ASSERTION_AUDIENCES":   "tockrctrl,tockrims",
		"PLATFORM_ASSERTION_VERIFY_KEYS": "",
	}
	config, err := ConfigFromEnvironment(func(key string) string { return values[key] })
	if err != nil {
		t.Fatal(err)
	}
	if config.Issuer != values["PLATFORM_ASSERTION_ISSUER"] || config.ActiveKeyID != values["PLATFORM_ASSERTION_KEY_ID"] || len(config.Audiences) != 2 {
		t.Fatalf("environment config = %#v", config)
	}
	if _, err := ConfigFromEnvironment(func(key string) string {
		if key == "PLATFORM_ASSERTION_ISSUER" {
			return ""
		}
		return values[key]
	}); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("missing issuer error = %v, want configuration error", err)
	}
}

func TestAssertionPayloadHasStrictContractShape(t *testing.T) {
	issuer, now := testIssuer(t, "key-current")
	token, _, err := issuer.Issue(IssueRequest{Audience: "tockrctrl", PlatformUserID: "usr_1", OrganisationID: "org_1", WorkspaceID: "wsp_1"}, now)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"issuer", "audience", "platform_user_id", "platform_organisation_id", "platform_workspace_id", "issued_at", "expires_at", "assertion_id", "assertion_version"} {
		if _, ok := fields[field]; !ok {
			t.Fatalf("payload missing approved field %q", field)
		}
	}
	for _, field := range []string{"product_key", "product_role", "billing", "password", "session_token", "entitlement"} {
		if _, ok := fields[field]; ok {
			t.Fatalf("payload contains prohibited field %q", field)
		}
	}
}

func testIssuer(t *testing.T, keyID string) (*Issuer, time.Time) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	issuer, err := New(Config{
		Issuer:           "https://platform.example",
		ActiveKeyID:      keyID,
		ActivePrivateKey: privateKey,
		VerificationKeys: map[string]ed25519.PublicKey{keyID: publicKey},
		Audiences:        []string{"tockrctrl", "tockrims"},
		MaxLifetime:      5 * time.Minute,
		ClockSkew:        5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	return issuer, time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
}
