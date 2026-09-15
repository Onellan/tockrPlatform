package sqlite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/Onellan/tockrplatform/internal/auth"
)

func TestSessionExpiryRevocationProjectionAndBoundedCleanup(t *testing.T) {
	ctx := context.Background()
	store, user := newMFAStore(t)
	now := time.Now().UTC()
	token, err := auth.NewOpaqueToken(32)
	if err != nil {
		t.Fatal(err)
	}
	csrf, err := auth.NewOpaqueToken(32)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSession(ctx, user.ID, token, csrf, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	projection, err := store.AuthenticatedSession(ctx, token, now)
	if err != nil || projection.User.PasswordHash != "" || projection.Session.UserID != user.ID {
		t.Fatalf("authenticated projection = %#v, err=%v", projection, err)
	}
	if _, err := store.AuthenticatedSession(ctx, token, now.Add(2*time.Hour)); err != sql.ErrNoRows {
		t.Fatalf("expired session error = %v", err)
	}
	if err := store.RevokeSession(ctx, token, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AuthenticatedSession(ctx, token, now); err != sql.ErrNoRows {
		t.Fatalf("revoked session error = %v", err)
	}

	var internalUserID int64
	if err := store.DB().QueryRowContext(ctx, `SELECT id FROM users WHERE public_id=?`, user.ID).Scan(&internalUserID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		raw, tokenErr := auth.NewOpaqueToken(32)
		if tokenErr != nil {
			t.Fatal(tokenErr)
		}
		if _, err := store.DB().ExecContext(ctx, `INSERT INTO sessions(user_id,token_hash,csrf_token_hash,created_at,expires_at) VALUES(?,?,?,?,?)`, internalUserID, auth.HashToken(raw), auth.HashToken("cleanup-csrf"+raw), formatTime(now.Add(-time.Hour)), formatTime(now.Add(-time.Minute))); err != nil {
			t.Fatal(err)
		}
	}
	cleaned, err := store.CleanupExpiredSessions(ctx, now, 1)
	if err != nil || cleaned != 1 {
		t.Fatalf("bounded cleanup = %d, err=%v", cleaned, err)
	}
	var remaining int
	if err := store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions WHERE expires_at<=?`, formatTime(now)).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 1 {
		t.Fatalf("remaining expired sessions = %d, want 1", remaining)
	}
	if _, err := store.CleanupExpiredSessions(ctx, now, 0); err == nil {
		t.Fatal("zero cleanup limit was accepted")
	}
}
