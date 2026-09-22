package sqlite

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
	"github.com/Onellan/tockrplatform/internal/store"
)

func TestMembershipCommandResultCleanupIsBounded(t *testing.T) {
	ctx := context.Background()
	persistence, _ := newOrganisationStore(t, 1)
	old := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	for index := 0; index < 3; index++ {
		if _, err := persistence.DB().ExecContext(ctx, `INSERT INTO platform_membership_command_results(consumer_key,idempotency_key,request_hash,result_json,created_at) VALUES(?,?,?,?,?)`, "tockrctrl", "cleanup-"+string(rune('a'+index)), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "{}", formatTime(old.Add(time.Duration(index)*time.Minute))); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := persistence.CleanupMembershipCommandResults(ctx, old.Add(2*time.Minute), 2)
	if err != nil || removed != 2 {
		t.Fatalf("cleanup removed=%d err=%v, want two rows", removed, err)
	}
	var remaining int
	if err := persistence.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM platform_membership_command_results`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 1 {
		t.Fatalf("remaining command results=%d, want one", remaining)
	}
	if _, err := persistence.CleanupMembershipCommandResults(ctx, old, 0); err != store.ErrInvalidReadAuthorityRequest {
		t.Fatalf("invalid cleanup limit=%v, want invalid request", err)
	}
	if _, err := persistence.CleanupMembershipCommandResults(ctx, time.Now().UTC().Add(-time.Hour), 10); err != store.ErrInvalidReadAuthorityRequest {
		t.Fatalf("recent cleanup cutoff=%v, want invalid request", err)
	}
}

func TestMembershipCommandConcurrentExpectedVersionAllowsOneMutation(t *testing.T) {
	ctx := context.Background()
	persistence, users := newOrganisationStore(t, 3)
	now := time.Now().UTC().Truncate(time.Second)
	organisation, _, err := persistence.CreateOrganisation(ctx, users[0].ID, domain.Organisation{Name: "Concurrent commands"}, "create concurrent command organisation", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.AddOrganisationMember(ctx, users[0].ID, organisation.ID, users[1].ID, domain.OrganisationMember, "seed concurrent command", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	requests := []store.MembershipCommandRequest{
		{Consumer: "tockrctrl", Scope: "organisation_membership", Operation: "change_role", OrganisationID: organisation.ID, UserID: users[1].ID, Role: "admin", Reason: "concurrent command one", IdempotencyKey: "concurrent-command-one", ExpectedVersion: 1, RequestHash: strings.Repeat("1", 64), ActorUserID: users[0].ID, OccurredAt: now.Add(2 * time.Minute)},
		{Consumer: "tockrctrl", Scope: "organisation_membership", Operation: "change_role", OrganisationID: organisation.ID, UserID: users[1].ID, Role: "admin", Reason: "concurrent command two", IdempotencyKey: "concurrent-command-two", ExpectedVersion: 1, RequestHash: strings.Repeat("2", 64), ActorUserID: users[0].ID, OccurredAt: now.Add(2 * time.Minute)},
	}
	results := make([]error, len(requests))
	var group sync.WaitGroup
	for index := range requests {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			_, results[index] = persistence.ExecuteMembershipCommand(ctx, requests[index])
		}(index)
	}
	group.Wait()
	successes := 0
	conflicts := 0
	for _, err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, store.ErrMembershipCommandConflict) || errors.Is(err, ErrDuplicateOrganisationMember) {
			conflicts++
		} else {
			t.Fatalf("concurrent command error = %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent command results successes=%d conflicts=%d errors=%v", successes, conflicts, results)
	}
}
