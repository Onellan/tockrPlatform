package assertion

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
	"time"
)

func TestConsumerAudienceProductCompatibilityMatrix(t *testing.T) {
	for _, test := range []struct {
		audience string
		product  string
	}{
		{audience: "tockrctrl", product: "product.tockrctrl"},
		{audience: "tockrims", product: "product.tockrims"},
	} {
		product, ok := ProductForAudience(test.audience)
		if !ok || product != test.product {
			t.Fatalf("audience %q maps to %q/%v, want %q/true", test.audience, product, ok, test.product)
		}
	}
	if _, ok := ProductForAudience("product.tockrctrl"); ok {
		t.Fatal("product key was accepted as an audience")
	}
}

func TestConsumerValidationFailureTaxonomy(t *testing.T) {
	issuer, now := testIssuer(t, "key-current")
	validToken, claims, err := issuer.Issue(IssueRequest{Audience: "tockrctrl", PlatformUserID: "usr_1", OrganisationID: "org_1", WorkspaceID: "wsp_1"}, now)
	if err != nil {
		t.Fatal(err)
	}
	validVerifier := verifierForIssuer(t, issuer, "key-current")
	verified, err := validVerifier.VerifyForProduct(validToken, "tockrctrl", "product.tockrctrl", now.Add(time.Second))
	if err != nil || verified != claims {
		t.Fatalf("valid consumer assertion = %#v, err=%v", verified, err)
	}

	wrongAudienceVerifier := verifierForIssuer(t, issuer, "key-current")
	if _, err := wrongAudienceVerifier.VerifyForProduct(validToken, "tockrims", "product.tockrims", now.Add(time.Second)); FailureClassOf(err) != FailureForbidden {
		t.Fatalf("wrong audience class = %v, err=%v", FailureClassOf(err), err)
	}
	wrongProductVerifier := verifierForIssuer(t, issuer, "key-current")
	if _, err := wrongProductVerifier.VerifyForProduct(validToken, "tockrctrl", "product.tockrims", now.Add(time.Second)); FailureClassOf(err) != FailureForbidden {
		t.Fatalf("wrong product class = %v, err=%v", FailureClassOf(err), err)
	}

	expiredToken, _, err := issuer.Issue(IssueRequest{Audience: "tockrctrl", PlatformUserID: "usr_2", OrganisationID: "org_2", WorkspaceID: "wsp_2"}, now.Add(-DefaultLifetime-time.Second))
	if err != nil {
		t.Fatal(err)
	}
	expiredVerifier := verifierForIssuer(t, issuer, "key-current")
	if _, err := expiredVerifier.VerifyForProduct(expiredToken, "tockrctrl", "product.tockrctrl", now); FailureClassOf(err) != FailureStale {
		t.Fatalf("expired class = %v, err=%v", FailureClassOf(err), err)
	}

	staleScope := claims
	staleScope.AssertionID = "ast_stale_scope"
	staleScope.PlatformWorkspaceID = "not-a-workspace"
	staleScopeToken, _, err := issuer.sign(staleScope)
	if err != nil {
		t.Fatal(err)
	}
	staleScopeVerifier := verifierForIssuer(t, issuer, "key-current")
	if _, err := staleScopeVerifier.VerifyForProduct(staleScopeToken, "tockrctrl", "product.tockrctrl", now); FailureClassOf(err) != FailureForbidden {
		t.Fatalf("stale scope class = %v, err=%v", FailureClassOf(err), err)
	}

	versionMismatch := claims
	versionMismatch.AssertionID = "ast_version_mismatch"
	versionMismatch.AssertionVersion = CurrentVersion + 1
	versionToken, _, err := issuer.sign(versionMismatch)
	if err != nil {
		t.Fatal(err)
	}
	versionVerifier := verifierForIssuer(t, issuer, "key-current")
	if _, err := versionVerifier.VerifyForProduct(versionToken, "tockrctrl", "product.tockrctrl", now); FailureClassOf(err) != FailureVersionMismatch {
		t.Fatalf("version mismatch class = %v, err=%v", FailureClassOf(err), err)
	}

	_, retiredPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	retiredIssuer, err := New(Config{Issuer: "https://platform.example", ActiveKeyID: "key-retired", ActivePrivateKey: retiredPrivate, Audiences: []string{"tockrctrl"}})
	if err != nil {
		t.Fatal(err)
	}
	retiredToken, _, err := retiredIssuer.Issue(IssueRequest{Audience: "tockrctrl", PlatformUserID: "usr_3", OrganisationID: "org_3", WorkspaceID: "wsp_3"}, now)
	if err != nil {
		t.Fatal(err)
	}
	retiredKeyVerifier := verifierForIssuer(t, issuer, "key-current")
	if _, err := retiredKeyVerifier.VerifyForProduct(retiredToken, "tockrctrl", "product.tockrctrl", now); FailureClassOf(err) != FailureUnauthenticated {
		t.Fatalf("retired key class = %v, err=%v", FailureClassOf(err), err)
	}

	var unavailable *Issuer
	if _, err := unavailable.VerifyForProduct(validToken, "tockrctrl", "product.tockrctrl", now); FailureClassOf(err) != FailureUnavailable {
		t.Fatalf("unavailable class = %v, err=%v", FailureClassOf(err), err)
	}
}

func verifierForIssuer(t *testing.T, issuer *Issuer, keyID string) *Issuer {
	t.Helper()
	keys := issuer.PublicKeys()
	for _, key := range keys {
		if key.KeyID == keyID {
			verifier, err := NewVerifier(VerifierConfig{Issuer: "https://platform.example", VerificationKeys: map[string]ed25519.PublicKey{keyID: key.PublicKey}, Audiences: []string{"tockrctrl", "tockrims"}, MaxLifetime: 5 * time.Minute, ClockSkew: 5 * time.Second})
			if err != nil {
				t.Fatal(err)
			}
			return verifier
		}
	}
	t.Fatalf("verification key %q not found", keyID)
	return nil
}

func TestConsumerValidationErrorDoesNotExposeClaims(t *testing.T) {
	err := ConsumerValidationError{Class: FailureForbidden}
	if errors.Is(err, ErrInvalidScope) || err.Error() != "consumer assertion validation failed: forbidden" {
		t.Fatalf("consumer error = %v", err)
	}
}
