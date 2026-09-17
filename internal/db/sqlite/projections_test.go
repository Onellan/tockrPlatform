package sqlite

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
	"github.com/Onellan/tockrplatform/internal/events"
	"github.com/Onellan/tockrplatform/internal/store"
)

func TestProjectionInboxIsIdempotentOrderedBoundedAndReconciled(t *testing.T) {
	ctx := context.Background()
	persistence, users := newOrganisationStore(t, 2)
	now := time.Date(2026, 9, 16, 13, 0, 0, 0, time.UTC)
	organisation, _, err := persistence.CreateOrganisation(ctx, users[0].ID, domain.Organisation{Name: "Projection"}, "projection test", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[1].ID, domain.OrganisationMember, "projection member", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	pending, err := persistence.ListPendingPlatformEvents(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	organisationEvents := filterEvents(pending, events.AggregateOrganisation, organisation.ID)
	if len(organisationEvents) != 3 {
		t.Fatalf("organisation source events = %d, want 3", len(organisationEvents))
	}

	consumer := "tockrctrl"
	gapResult, err := persistence.IngestPlatformProjectionEvent(ctx, consumer, organisationEvents[2], now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if gapResult.Outcome != store.ProjectionOutcomeGapFound || gapResult.State != store.ProjectionInboxGap {
		t.Fatalf("gap result = %#v", gapResult)
	}
	checkpoint, err := persistence.GetPlatformProjectionCheckpoint(ctx, consumer, string(events.AggregateOrganisation), organisation.ID)
	if err != nil || checkpoint.State != store.ProjectionGap {
		t.Fatalf("gap checkpoint = %#v, err=%v", checkpoint, err)
	}
	if err := persistence.MarkPlatformProjectionEventApplied(ctx, consumer, organisationEvents[2].EventID, now.Add(3*time.Minute)); !errors.Is(err, store.ErrProjectionGap) {
		t.Fatalf("apply gap event = %v, want gap", err)
	}

	firstResult, err := persistence.IngestPlatformProjectionEvent(ctx, consumer, organisationEvents[0], now.Add(4*time.Minute))
	if err != nil || firstResult.Outcome != store.ProjectionOutcomeAccepted {
		t.Fatalf("first ingest = %#v, err=%v", firstResult, err)
	}
	duplicateResult, err := persistence.IngestPlatformProjectionEvent(ctx, consumer, organisationEvents[0], now.Add(5*time.Minute))
	if err != nil || duplicateResult.Outcome != store.ProjectionOutcomeDuplicate {
		t.Fatalf("duplicate ingest = %#v, err=%v", duplicateResult, err)
	}
	if err := persistence.MarkPlatformProjectionEventApplied(ctx, consumer, organisationEvents[0].EventID, now.Add(6*time.Minute)); err != nil {
		t.Fatal(err)
	}
	secondResult, err := persistence.IngestPlatformProjectionEvent(ctx, consumer, organisationEvents[1], now.Add(7*time.Minute))
	if err != nil || secondResult.Outcome != store.ProjectionOutcomeAccepted {
		t.Fatalf("second ingest = %#v, err=%v", secondResult, err)
	}
	if err := persistence.MarkPlatformProjectionEventApplied(ctx, consumer, organisationEvents[1].EventID, now.Add(8*time.Minute)); err != nil {
		t.Fatal(err)
	}
	reconciled, err := persistence.ReconcilePlatformProjection(ctx, consumer, string(events.AggregateOrganisation), organisation.ID, 1, now.Add(9*time.Minute))
	if err != nil || reconciled.Promoted != 1 || reconciled.Checkpoint.State != store.ProjectionStale {
		t.Fatalf("reconciliation = %#v, err=%v", reconciled, err)
	}
	if err := persistence.MarkPlatformProjectionEventApplied(ctx, consumer, organisationEvents[2].EventID, now.Add(10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	checkpoint, err = persistence.GetPlatformProjectionCheckpoint(ctx, consumer, string(events.AggregateOrganisation), organisation.ID)
	if err != nil || checkpoint.State != store.ProjectionCurrent || checkpoint.LastSequence != 3 {
		t.Fatalf("current checkpoint = %#v, err=%v", checkpoint, err)
	}
	if err := persistence.MarkPlatformProjectionEventApplied(ctx, consumer, organisationEvents[2].EventID, now.Add(11*time.Minute)); err != nil {
		t.Fatal(err)
	}
	inbox, err := persistence.ListPlatformProjectionInbox(ctx, consumer, 100)
	if err != nil || len(inbox) != 3 {
		t.Fatalf("inbox = %d/%v, want 3", len(inbox), err)
	}
	if _, err := persistence.ListPlatformProjectionInbox(ctx, consumer, 0); !errors.Is(err, store.ErrInvalidProjectionLimit) {
		t.Fatalf("invalid inbox limit = %v", err)
	}
}

func TestProjectionInboxRetainsUnknownVersionBlockedAndUnavailableState(t *testing.T) {
	ctx := context.Background()
	persistence, users := newOrganisationStore(t, 1)
	now := time.Now().UTC().Truncate(time.Second)
	organisation, _, err := persistence.CreateOrganisation(ctx, users[0].ID, domain.Organisation{Name: "Blocked projection"}, "projection blocked", now)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := persistence.ListPendingPlatformEvents(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	source := filterEvents(pending, events.AggregateOrganisation, organisation.ID)[0]
	eventID, err := events.NewEventID()
	if err != nil {
		t.Fatal(err)
	}
	source.EventID = eventID
	source.SchemaVersion = 2
	result, err := persistence.IngestPlatformProjectionEvent(ctx, "tockrims", source, now.Add(time.Minute))
	if err != nil || result.Outcome != store.ProjectionOutcomeBlocked || result.State != store.ProjectionInboxBlocked || result.Reason != "unknown_schema_version" {
		t.Fatalf("unknown version result = %#v, err=%v", result, err)
	}
	checkpoint, err := persistence.GetPlatformProjectionCheckpoint(ctx, "tockrims", string(events.AggregateOrganisation), organisation.ID)
	if err != nil || checkpoint.State != store.ProjectionBlocked {
		t.Fatalf("blocked checkpoint = %#v, err=%v", checkpoint, err)
	}
	if err := persistence.MarkPlatformProjectionEventApplied(ctx, "tockrims", eventID, now.Add(2*time.Minute)); !errors.Is(err, store.ErrProjectionBlocked) {
		t.Fatalf("apply blocked event = %v", err)
	}
	if err := persistence.MarkPlatformProjectionUnavailable(ctx, "tockrims", string(events.AggregateOrganisation), "org_unavailable", "source unavailable", now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	checkpoint, err = persistence.GetPlatformProjectionCheckpoint(ctx, "tockrims", string(events.AggregateOrganisation), "org_unavailable")
	if err != nil || checkpoint.State != store.ProjectionUnavailable || checkpoint.Reason != "source unavailable" {
		t.Fatalf("unavailable checkpoint = %#v, err=%v", checkpoint, err)
	}
	resumed, err := persistence.ResumePlatformProjection(ctx, "tockrims", string(events.AggregateOrganisation), "org_unavailable", now.Add(4*time.Minute))
	if err != nil || resumed.State != store.ProjectionCurrent {
		t.Fatalf("resumed checkpoint = %#v, err=%v", resumed, err)
	}
	if err := persistence.MarkPlatformProjectionUnavailable(ctx, "tockrims", string(events.AggregateOrganisation), "org_unavailable", "", now); err == nil {
		t.Fatal("empty unavailable reason succeeded")
	}
}

func TestProjectionInboxMigrationFreshUpgradeAndReopen(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir() + "/projection.db"
	first, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	assertProjectionTables(t, first)
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	assertProjectionTables(t, second)
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestProjectionInboxConcurrentDuplicateAndIdentityConflictFailClosed(t *testing.T) {
	ctx := context.Background()
	persistence, users := newOrganisationStore(t, 1)
	now := time.Now().UTC().Truncate(time.Second)
	organisation, _, err := persistence.CreateOrganisation(ctx, users[0].ID, domain.Organisation{Name: "Projection concurrency"}, "projection concurrency", now)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := persistence.ListPendingPlatformEvents(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	source := filterEvents(pending, events.AggregateOrganisation, organisation.ID)[0]
	results := make(chan store.ProjectionIngestResult, 2)
	errorsFound := make(chan error, 2)
	var group sync.WaitGroup
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			result, callErr := persistence.IngestPlatformProjectionEvent(ctx, "concurrent", source, now.Add(time.Minute))
			results <- result
			errorsFound <- callErr
		}()
	}
	group.Wait()
	close(results)
	close(errorsFound)
	accepted := 0
	duplicates := 0
	for result := range results {
		switch result.Outcome {
		case store.ProjectionOutcomeAccepted:
			accepted++
		case store.ProjectionOutcomeDuplicate:
			duplicates++
		default:
			t.Fatalf("concurrent result = %#v", result)
		}
	}
	for callErr := range errorsFound {
		if callErr != nil {
			t.Fatal(callErr)
		}
	}
	if accepted != 1 || duplicates != 1 {
		t.Fatalf("concurrent accepted/duplicates = %d/%d, want 1/1", accepted, duplicates)
	}
	conflict := source
	conflict.AggregateID = "org_conflicting"
	if _, err := persistence.IngestPlatformProjectionEvent(ctx, "concurrent", conflict, now.Add(2*time.Minute)); !errors.Is(err, store.ErrProjectionEventConflict) {
		t.Fatalf("conflicting event = %v, want identity conflict", err)
	}
}

func assertProjectionTables(t *testing.T, persistence *Store) {
	t.Helper()
	var version int
	if err := persistence.DB().QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 10 {
		t.Fatalf("projection migration version = %d, want 10", version)
	}
	for _, table := range []string{"platform_projection_inbox", "platform_projection_checkpoints"} {
		var count int
		if err := persistence.DB().QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("table %s exists = %d, want 1", table, count)
		}
	}
}
