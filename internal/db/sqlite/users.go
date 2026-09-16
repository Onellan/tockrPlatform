package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Onellan/tockrplatform/internal/auth"
	"github.com/Onellan/tockrplatform/internal/domain"
	"github.com/Onellan/tockrplatform/internal/events"
	"github.com/Onellan/tockrplatform/internal/store"
)

func (s *Store) CreateUser(ctx context.Context, user domain.User, password string) (domain.User, error) {
	if err := user.Validate(); err != nil {
		return domain.User{}, err
	}
	if err := auth.ValidatePassword(password); err != nil {
		return domain.User{}, domain.ErrInvalidPassword
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return domain.User{}, fmt.Errorf("hash user password: %w", err)
	}
	now := time.Now().UTC()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.User{}, fmt.Errorf("begin user creation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `INSERT INTO users(public_id,email,display_name,password_hash,active,created_at) VALUES(?,?,?,?,?,?)`,
		user.ID, domain.NormalizeEmail(user.Email), strings.TrimSpace(user.DisplayName), hash, boolInt(user.Active), formatTime(user.CreatedAt)); err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}
	if err := appendPlatformEventTx(ctx, tx, events.EventUserCreated, user.ID, user.CreatedAt, map[string]any{"user_id": user.ID, "active": user.Active}); err != nil {
		return domain.User{}, fmt.Errorf("record user creation event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.User{}, fmt.Errorf("commit user creation: %w", err)
	}
	user.Email = domain.NormalizeEmail(user.Email)
	user.PasswordHash = ""
	return user, nil
}

func (s *Store) FindUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.findUser(ctx, `WHERE lower(email)=lower(?)`, domain.NormalizeEmail(email))
}

func (s *Store) FindUserByID(ctx context.Context, publicID string) (*domain.User, error) {
	return s.findUser(ctx, `WHERE public_id=?`, strings.TrimSpace(publicID))
}

func (s *Store) FindLoginCredential(ctx context.Context, email string) (*store.LoginCredential, error) {
	var credential store.LoginCredential
	var active, mfaEnabled int
	var created, lastLogin sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT public_id,email,display_name,password_hash,active,created_at,last_login_at,mfa_enabled FROM users WHERE lower(email)=lower(?)`, domain.NormalizeEmail(email)).
		Scan(&credential.User.ID, &credential.User.Email, &credential.User.DisplayName, &credential.PasswordHash, &active, &created, &lastLogin, &mfaEnabled)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find login credential: %w", err)
	}
	credential.User.Active = active == 1
	credential.User.MFAEnabled = mfaEnabled == 1
	credential.User.CreatedAt = parseTime(created.String)
	if lastLogin.Valid {
		value := parseTime(lastLogin.String)
		credential.User.LastLoginAt = &value
	}
	return &credential, nil
}

func (s *Store) findUser(ctx context.Context, predicate string, arg any) (*domain.User, error) {
	var user domain.User
	var active, mfaEnabled int
	var ciphertext []byte
	var created, lastLogin sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT public_id,email,display_name,password_hash,active,created_at,last_login_at,mfa_enabled,mfa_secret_ciphertext FROM users `+predicate, arg).
		Scan(&user.ID, &user.Email, &user.DisplayName, &user.PasswordHash, &active, &created, &lastLogin, &mfaEnabled, &ciphertext)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	user.Active = active == 1
	user.MFAEnabled = mfaEnabled == 1
	user.CreatedAt = parseTime(created.String)
	if lastLogin.Valid {
		value := parseTime(lastLogin.String)
		user.LastLoginAt = &value
	}
	return &user, nil
}

func (s *Store) SetUserActive(ctx context.Context, publicID string, active bool) error {
	publicID = strings.TrimSpace(publicID)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin user status change: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var current int
	if err := tx.QueryRowContext(ctx, `SELECT active FROM users WHERE public_id=?`, publicID).Scan(&current); errors.Is(err, sql.ErrNoRows) {
		return sql.ErrNoRows
	} else if err != nil {
		return fmt.Errorf("read user status: %w", err)
	}
	if current == boolInt(active) {
		return nil
	}
	result, err := tx.ExecContext(ctx, `UPDATE users SET active=? WHERE public_id=?`, boolInt(active), publicID)
	if err != nil {
		return fmt.Errorf("set user active: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("check user active update: %w", err)
	} else if rows != 1 {
		return sql.ErrNoRows
	}
	if err := appendPlatformEventTx(ctx, tx, events.EventUserStatusChanged, publicID, time.Now().UTC(), map[string]any{"user_id": publicID, "active": active}); err != nil {
		return fmt.Errorf("record user status event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit user status change: %w", err)
	}
	return nil
}

func (s *Store) TouchLogin(ctx context.Context, publicID string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE users SET last_login_at=? WHERE public_id=?`, formatTime(at.UTC()), strings.TrimSpace(publicID))
	return err
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func parseTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, value)
	return parsed
}
