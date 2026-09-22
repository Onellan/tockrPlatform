package sqlite

import (
	"context"
	"testing"
	"time"

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
