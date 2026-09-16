package events

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestVersionedEnvelopeIsStrictAndRedacted(t *testing.T) {
	event, err := New(EventOrganisationMembershipAdded, "org_example", time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC), map[string]any{
		"organisation_id": "org_example",
		"membership_id":   "omem_example",
		"user_id":         "usr_example",
		"role":            "member",
		"active":          true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if event.Sequence != 0 {
		t.Fatalf("new event sequence = %d, want unset", event.Sequence)
	}
	sequenced, err := event.WithSequence(1)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := sequenced.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &envelope); err != nil {
		t.Fatal(err)
	}
	if _, exists := envelope["payload"]; !exists {
		t.Fatal("envelope omitted payload")
	}
	if _, exists := envelope["reason"]; exists {
		t.Fatal("envelope exposed mutation reason")
	}
	if err := sequenced.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestEnvelopeRejectsUnknownSecretsAndInvalidIdentity(t *testing.T) {
	base := map[string]any{"user_id": "usr_example", "active": true}
	withSecret := map[string]any{"user_id": "usr_example", "active": true, "password": "not allowed"}
	if _, err := New(EventUserCreated, "usr_example", time.Now().UTC(), withSecret); !errors.Is(err, ErrInvalidEventPayload) {
		t.Fatalf("secret payload error = %v, want invalid payload", err)
	}
	if _, err := New(EventUserCreated, "org_example", time.Now().UTC(), base); !errors.Is(err, ErrInvalidAggregateID) {
		t.Fatalf("aggregate identity error = %v, want invalid aggregate", err)
	}
	if _, err := New("platform.unknown.event", "usr_example", time.Now().UTC(), base); !errors.Is(err, ErrUnknownEventType) {
		t.Fatalf("unknown event error = %v, want unknown event", err)
	}
	if _, err := New(EventUserCreated, "usr_example", time.Now().UTC(), map[string]any{"user_id": "usr_example"}); !errors.Is(err, ErrInvalidEventPayload) {
		t.Fatalf("missing field error = %v, want invalid payload", err)
	}
}

func TestEnvelopeSequenceAndPayloadBoundsFailClosed(t *testing.T) {
	event, err := New(EventProductRetired, "product.tockrctrl", time.Now().UTC(), map[string]any{"product_key": "product.tockrctrl", "status": "retired"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := event.WithSequence(0); !errors.Is(err, ErrInvalidEventSequence) {
		t.Fatalf("zero sequence error = %v, want invalid sequence", err)
	}
	large := make(map[string]any)
	large["product_key"] = "product.tockrctrl"
	large["status"] = string(make([]byte, MaxPayloadBytes))
	if _, err := New(EventProductRetired, "product.tockrctrl", time.Now().UTC(), large); !errors.Is(err, ErrInvalidEventPayload) {
		t.Fatalf("large payload error = %v, want invalid payload", err)
	}
}
