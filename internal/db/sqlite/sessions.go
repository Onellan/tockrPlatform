package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Onellan/tockrplatform/internal/auth"
	"github.com/Onellan/tockrplatform/internal/store"
)

func (s *Store) CreateSession(ctx context.Context, userID, token, csrfToken string, expiresAt time.Time) error {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(token) == "" || strings.TrimSpace(csrfToken) == "" {
		return errors.New("session user and tokens are required")
	}
	var internalID int64
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE public_id=? AND active=1`, userID).Scan(&internalID); err != nil {
		return fmt.Errorf("resolve session user: %w", err)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO sessions(user_id,token_hash,csrf_token_hash,created_at,expires_at) VALUES(?,?,?,?,?)`,
		internalID, auth.HashToken(token), auth.HashToken(csrfToken), formatTime(time.Now().UTC()), formatTime(expiresAt.UTC()))
	return err
}

func (s *Store) EstablishSession(ctx context.Context, userID, token, csrfToken string, expiresAt, loginAt time.Time) error {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(token) == "" || strings.TrimSpace(csrfToken) == "" {
		return errors.New("session user and tokens are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin session establishment: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var internalID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE public_id=? AND active=1`, userID).Scan(&internalID); err != nil {
		return fmt.Errorf("resolve session user: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO sessions(user_id,token_hash,csrf_token_hash,created_at,expires_at) VALUES(?,?,?,?,?)`,
		internalID, auth.HashToken(token), auth.HashToken(csrfToken), formatTime(loginAt.UTC()), formatTime(expiresAt.UTC())); err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET last_login_at=? WHERE id=?`, formatTime(loginAt.UTC()), internalID); err != nil {
		return fmt.Errorf("touch login: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO security_events(user_id,event,details,occurred_at) VALUES(?,?,?,?)`, internalID, "login", "password", formatTime(loginAt.UTC())); err != nil {
		return fmt.Errorf("record login audit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit session establishment: %w", err)
	}
	return nil
}

func (s *Store) AuthenticatedSession(ctx context.Context, token string, now time.Time) (store.AuthenticatedSession, error) {
	var result store.AuthenticatedSession
	var active int
	var expires, created string
	var lastLogin sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT u.public_id,u.email,u.display_name,u.active,u.created_at,u.last_login_at,s.expires_at
		FROM sessions s JOIN users u ON u.id=s.user_id
		WHERE s.token_hash=? AND s.revoked_at IS NULL AND s.expires_at>? AND u.active=1`, auth.HashToken(token), formatTime(now.UTC())).
		Scan(&result.User.ID, &result.User.Email, &result.User.DisplayName, &active, &created, &lastLogin, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return store.AuthenticatedSession{}, sql.ErrNoRows
	}
	if err != nil {
		return store.AuthenticatedSession{}, fmt.Errorf("authenticate session: %w", err)
	}
	result.User.Active = active == 1
	result.User.CreatedAt = parseTime(created)
	if lastLogin.Valid {
		value := parseTime(lastLogin.String)
		result.User.LastLoginAt = &value
	}
	result.User.PasswordHash = ""
	result.Session.UserID = result.User.ID
	result.Session.ExpiresAt = parseTime(expires)
	return result, nil
}

func (s *Store) SessionCSRFHash(ctx context.Context, token string) (string, error) {
	var hash string
	err := s.db.QueryRowContext(ctx, `SELECT csrf_token_hash FROM sessions WHERE token_hash=? AND revoked_at IS NULL`, auth.HashToken(token)).Scan(&hash)
	return hash, err
}

func (s *Store) RevokeSession(ctx context.Context, token string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET revoked_at=COALESCE(revoked_at,?) WHERE token_hash=?`, formatTime(at.UTC()), auth.HashToken(token))
	return err
}

func (s *Store) RevokeSessionAndRecordLogout(ctx context.Context, token, userID string, at time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin logout: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `UPDATE sessions SET revoked_at=COALESCE(revoked_at,?) WHERE token_hash=? AND user_id=(SELECT id FROM users WHERE public_id=?)`, formatTime(at.UTC()), auth.HashToken(token), userID)
	if err != nil {
		return fmt.Errorf("revoke logout session: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("check logout session: %w", err)
	} else if rows != 1 {
		return sql.ErrNoRows
	}
	var internalID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE public_id=?`, userID).Scan(&internalID); err != nil {
		return fmt.Errorf("resolve logout user: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO security_events(user_id,event,details,occurred_at) VALUES(?,?,?,?)`, internalID, "logout", "session revoked", formatTime(at.UTC())); err != nil {
		return fmt.Errorf("record logout audit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit logout: %w", err)
	}
	return nil
}

func (s *Store) VerifySessionCSRF(ctx context.Context, token, csrfToken string) (bool, error) {
	hash, err := s.SessionCSRFHash(ctx, token)
	if err != nil {
		return false, err
	}
	return auth.VerifyTokenHash(hash, csrfToken), nil
}
