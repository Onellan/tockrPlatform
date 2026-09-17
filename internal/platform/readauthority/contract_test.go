package readauthority

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestContractVersionAndConsumerMatrixAreIndependentOfAssertions(t *testing.T) {
	if err := ValidateVersion(Version); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(ValidateVersion("platform.v1"), ErrUnsupportedVersion) {
		t.Fatal("assertion version was accepted as read-authority version")
	}
	for consumer, wantProduct := range map[string]string{
		ConsumerCTRL: ProductCTRL,
		ConsumerIMS:  ProductIMS,
	} {
		product, ok := ProductForConsumer(consumer)
		if !ok || product != wantProduct {
			t.Fatalf("consumer %q maps to %q/%v, want %q/true", consumer, product, ok, wantProduct)
		}
	}
	if _, ok := ProductForConsumer(ProductCTRL); ok {
		t.Fatal("product key was accepted as a consumer identity")
	}
}

func TestContractRejectsUnknownFieldsAndUnboundedSnapshotRequests(t *testing.T) {
	var request SnapshotRequest
	decoder := json.NewDecoder(strings.NewReader(`{"entity_kinds":["user"],"expires_in_seconds":60,"password":"not allowed"}`))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err == nil {
		t.Fatal("unknown secret field was accepted")
	}
	if err := ValidateSnapshotRequest(SnapshotRequest{EntityKinds: []EntityKind{EntityUser}, ExpiresInSeconds: SnapshotMaxTTLSeconds + 1}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("unbounded expiry error = %v", err)
	}
	if err := ValidateSnapshotRequest(SnapshotRequest{EntityKinds: []EntityKind{EntityUser, EntityUser}, ExpiresInSeconds: 60}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("duplicate entity error = %v", err)
	}
}

func TestContractStatesFailClosed(t *testing.T) {
	for _, state := range []State{StateStale, StateGap, StateBlocked, StateUnavailable, StateResyncRequired} {
		if err := ValidateState(state); err != nil {
			t.Fatal(err)
		}
		if IsAuthoritative(state) {
			t.Fatalf("non-current state %q was authoritative", state)
		}
	}
	if !IsAuthoritative(StateCurrent) {
		t.Fatal("current state was not authoritative")
	}
	if !errors.Is(ValidateState(State("unknown")), ErrInvalidState) {
		t.Fatal("unknown state did not fail closed")
	}
}

func TestCanonicalRequestBindsAllMachineAuthenticationInputs(t *testing.T) {
	bodyDigest := BodyDigest([]byte(`{"entity_kinds":["user"]}`))
	canonical, err := CanonicalRequest("post", "/api/v1/read-authority/snapshots", bodyDigest, ConsumerCTRL, "key-current", "1789646400", "nonce-1")
	if err != nil {
		t.Fatal(err)
	}
	want := "POST\n/api/v1/read-authority/snapshots\n" + bodyDigest + "\ntockrctrl\nkey-current\n1789646400\nnonce-1"
	if canonical != want {
		t.Fatalf("canonical request = %q, want %q", canonical, want)
	}
	if _, err := CanonicalRequest("GET", "/api/v1/read-authority/status", "not-a-digest", ConsumerCTRL, "key-current", "1789646400", "nonce-2"); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("invalid digest error = %v", err)
	}
	if _, err := CanonicalRequest("GET", "api/v1/read-authority/status", BodyDigest(nil), ConsumerCTRL, "key-current", "1789646400", "nonce-3"); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("relative path error = %v", err)
	}
}

func TestContractHeaderAndRouteSurfaceIsStable(t *testing.T) {
	name, value := ResponseVersionHeader()
	if name != VersionHeader || value != Version {
		t.Fatalf("response version header = %q:%q", name, value)
	}
	for _, route := range []string{
		"/api/v1/read-authority/snapshots",
		"/api/v1/read-authority/snapshots/snap_1/records",
		"/api/v1/read-authority/changes",
		"/api/v1/read-authority/status",
	} {
		if !IsReadAuthorityRoute(route) {
			t.Fatalf("route %q was not recognised", route)
		}
	}
	if IsReadAuthorityRoute("/api/v1/products") {
		t.Fatal("product route was classified as read authority")
	}
}
