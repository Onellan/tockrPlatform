package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
	"github.com/Onellan/tockrplatform/internal/events"
	"github.com/Onellan/tockrplatform/internal/store"
)

func TestPlatformOutboxIsTransactionalSequencedAndRedacted(t *testing.T) {
	ctx := context.Background()
	persistence, users := newOrganisationStore(t, 3)
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	organisation, _, err := persistence.CreateOrganisation(ctx, users[0].ID, domain.Organisation{Name: "Events"}, "create event organisation", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[1].ID, domain.OrganisationMember, "add event member", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	eventsBeforeFailure, err := persistence.ListPendingPlatformEvents(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	organisationEvents := filterEvents(eventsBeforeFailure, events.AggregateOrganisation, organisation.ID)
	if len(organisationEvents) != 3 {
		t.Fatalf("organisation outbox count = %d, want 3", len(organisationEvents))
	}
	for index, event := range organisationEvents {
		if event.Sequence != int64(index+1) {
			t.Fatalf("organisation event sequence at %d = %d, want %d", index, event.Sequence, index+1)
		}
		if event.SchemaVersion != events.SchemaVersion || event.EventID == "" || !json.Valid(event.Payload) {
			t.Fatalf("invalid outbox event = %#v", event)
		}
		if string(event.Payload) == "" || containsJSONKey(event.Payload, "reason") || containsJSONKey(event.Payload, "password_hash") {
			t.Fatalf("redaction failure payload = %s", event.Payload)
		}
	}
	if err := persistence.MarkPlatformEventsPublished(ctx, []string{organisationEvents[0].EventID}, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	remaining, err := persistence.ListPendingPlatformEvents(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(filterEvents(remaining, events.AggregateOrganisation, organisation.ID)) != 2 {
		t.Fatalf("remaining organisation events = %d, want 2", len(filterEvents(remaining, events.AggregateOrganisation, organisation.ID)))
	}
	if err := persistence.MarkPlatformEventsPublished(ctx, []string{organisationEvents[0].EventID}, now.Add(3*time.Minute)); err == nil {
		t.Fatal("republishing an outbox event succeeded")
	}
	if err := persistence.MarkPlatformEventsPublished(ctx, nil, now); !errors.Is(err, store.ErrInvalidOutboxLimit) {
		t.Fatalf("empty publication batch = %v, want invalid limit", err)
	}
}

func TestFailedAuthorityMutationLeavesNoOutboxFact(t *testing.T) {
	ctx := context.Background()
	persistence, users := newOrganisationStore(t, 2)
	now := time.Now().UTC().Truncate(time.Second)
	organisation, _, err := persistence.CreateOrganisation(ctx, users[0].ID, domain.Organisation{Name: "Rollback"}, "create rollback organisation", now)
	if err != nil {
		t.Fatal(err)
	}
	before, err := persistence.ListPendingPlatformEvents(ctx, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[1].ID, domain.OrganisationMember, "add once", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	beforeDuplicate, err := persistence.ListPendingPlatformEvents(ctx, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[1].ID, domain.OrganisationMember, "duplicate", now.Add(2*time.Minute)); !errors.Is(err, ErrDuplicateOrganisationMember) {
		t.Fatalf("duplicate mutation = %v, want duplicate", err)
	}
	after, err := persistence.ListPendingPlatformEvents(ctx, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(beforeDuplicate) || len(after) <= len(before) {
		t.Fatalf("failed mutation changed outbox count from %d to %d", len(beforeDuplicate), len(after))
	}
}

func TestOutboxAppendFailureRollsBackAuthorityAndAudit(t *testing.T) {
	ctx := context.Background()
	persistence, users := newOrganisationStore(t, 1)
	now := time.Now().UTC().Truncate(time.Second)
	if _, err := persistence.DB().ExecContext(ctx, `CREATE TRIGGER reject_platform_events BEFORE INSERT ON platform_outbox BEGIN SELECT RAISE(ABORT,'reject platform event'); END`); err != nil {
		t.Fatal(err)
	}
	_, _, err := persistence.CreateOrganisation(ctx, users[0].ID, domain.Organisation{Name: "Rejected"}, "trigger rollback", now)
	if err == nil {
		t.Fatal("authority mutation succeeded while outbox rejected")
	}
	var organisations int
	if err := persistence.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM organisations WHERE name='Rejected'`).Scan(&organisations); err != nil {
		t.Fatal(err)
	}
	if organisations != 0 {
		t.Fatalf("rolled-back organisation count = %d, want 0", organisations)
	}
	var audit int
	if err := persistence.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE aggregate_type='Organisation' AND event='organisation_created'`).Scan(&audit); err != nil {
		t.Fatal(err)
	}
	if audit != 0 {
		t.Fatalf("rolled-back audit count = %d, want 0", audit)
	}
	if _, err := persistence.DB().ExecContext(ctx, `DROP TRIGGER reject_platform_events`); err != nil {
		t.Fatal(err)
	}
}

func filterEvents(values []store.OutboxEvent, aggregateType events.AggregateType, aggregateID string) []store.OutboxEvent {
	filtered := make([]store.OutboxEvent, 0)
	for _, value := range values {
		if value.AggregateType == string(aggregateType) && value.AggregateID == aggregateID {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func containsJSONKey(payload []byte, key string) bool {
	var object map[string]json.RawMessage
	if json.Unmarshal(payload, &object) != nil {
		return true
	}
	_, exists := object[key]
	return exists
}
