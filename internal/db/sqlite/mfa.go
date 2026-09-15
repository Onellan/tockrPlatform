package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Onellan/tockrplatform/internal/auth"
)

const maxMFAEnrollmentAttempts = 5

func (s *Store) VerifyMFA(ctx context.Context, userID, code string) (bool, error) {
	var ciphertext []byte
	var enabled int
	var lastStep int64
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin MFA verification: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	err = tx.QueryRowContext(ctx, `SELECT mfa_enabled,mfa_secret_ciphertext,mfa_last_step FROM users WHERE public_id=? AND active=1`, userID).Scan(&enabled, &ciphertext, &lastStep)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read MFA credential: %w", err)
	}
	if enabled != 1 || len(ciphertext) == 0 {
		return false, nil
	}
	secret, err := auth.DecryptSecret(s.secretKey, ciphertext)
	if err != nil {
		return false, err
	}
	step, valid := auth.MatchTOTP(secret, code, time.Now().UTC())
	if !valid || step <= lastStep {
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("commit MFA replay check: %w", err)
		}
		return false, nil
	}
	result, err := tx.ExecContext(ctx, `UPDATE users SET mfa_last_step=? WHERE public_id=? AND active=1 AND mfa_last_step<?`, step, userID, step)
	if err != nil {
		return false, fmt.Errorf("record MFA step: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil {
		return false, fmt.Errorf("check MFA step: %w", err)
	} else if rows != 1 {
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("commit MFA replay rejection: %w", err)
		}
		return false, nil
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit MFA verification: %w", err)
	}
	return true, nil
}

func (s *Store) UseRecoveryCode(ctx context.Context, userID, code string, at time.Time) (bool, error) {
	code = auth.NormalizeRecoveryCode(code)
	if code == "" {
		return false, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin recovery code use: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var internalID int64
	var enabled int
	if err := tx.QueryRowContext(ctx, `SELECT id,mfa_enabled FROM users WHERE public_id=? AND active=1`, userID).Scan(&internalID, &enabled); errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("resolve recovery user: %w", err)
	}
	if enabled != 1 {
		return false, nil
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,code_hash FROM mfa_recovery_codes WHERE user_id=? AND used_at IS NULL ORDER BY id LIMIT 10`, internalID)
	if err != nil {
		return false, fmt.Errorf("read recovery codes: %w", err)
	}
	var matched int64
	for rows.Next() {
		var id int64
		var hash string
		if err := rows.Scan(&id, &hash); err != nil {
			_ = rows.Close()
			return false, fmt.Errorf("read recovery code: %w", err)
		}
		if matched == 0 && auth.CheckPassword(hash, code) {
			matched = id
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return false, fmt.Errorf("iterate recovery codes: %w", err)
	}
	if err := rows.Close(); err != nil {
		return false, fmt.Errorf("close recovery codes: %w", err)
	}
	if matched == 0 {
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("commit invalid recovery code: %w", err)
		}
		return false, nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE mfa_recovery_codes SET used_at=? WHERE id=? AND used_at IS NULL`, formatTime(at.UTC()), matched); err != nil {
		return false, fmt.Errorf("consume recovery code: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO security_events(user_id,event,details,occurred_at) VALUES(?,?,?,?)`, internalID, "mfa_recovery_used", "recovery code", formatTime(at.UTC())); err != nil {
		return false, fmt.Errorf("record recovery audit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit recovery code: %w", err)
	}
	return true, nil
}

func (s *Store) CreateMFAEnrollment(ctx context.Context, userID, secret string, expiresAt time.Time) (string, error) {
	if _, err := auth.TOTPCode(secret, time.Now().UTC()); err != nil {
		return "", err
	}
	ciphertext, err := auth.EncryptSecret(s.secretKey, secret)
	if err != nil {
		return "", err
	}
	token, err := auth.NewOpaqueToken(32)
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	if !expiresAt.After(now) || expiresAt.After(now.Add(30*time.Minute)) {
		return "", errors.New("MFA enrollment expiry is outside the bounded lifetime")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin MFA enrollment: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var internalID int64
	var enabled int
	if err := tx.QueryRowContext(ctx, `SELECT id,mfa_enabled FROM users WHERE public_id=? AND active=1`, userID).Scan(&internalID, &enabled); err != nil {
		return "", fmt.Errorf("resolve MFA enrollment user: %w", err)
	}
	if enabled == 1 {
		return "", errors.New("MFA is already enabled")
	}
	if _, err := tx.ExecContext(ctx, `UPDATE mfa_enrollments SET used_at=? WHERE user_id=? AND used_at IS NULL`, formatTime(now), internalID); err != nil {
		return "", fmt.Errorf("expire prior MFA enrollment: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO mfa_enrollments(user_id,token_hash,secret_ciphertext,expires_at,created_at) VALUES(?,?,?,?,?)`, internalID, auth.HashToken(token), ciphertext, formatTime(expiresAt.UTC()), formatTime(now)); err != nil {
		return "", fmt.Errorf("create MFA enrollment: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit MFA enrollment: %w", err)
	}
	return token, nil
}

func (s *Store) CompleteMFAEnrollment(ctx context.Context, userID, token, code string, recoveryHashes []string, at time.Time) (bool, error) {
	if len(recoveryHashes) != 10 {
		return false, errors.New("exactly 10 recovery codes are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin MFA confirmation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var internalID, attempts int64
	var ciphertext []byte
	var expires string
	var enabled int
	err = tx.QueryRowContext(ctx, `SELECT e.user_id,e.secret_ciphertext,e.expires_at,e.attempts,u.mfa_enabled
		FROM mfa_enrollments e JOIN users u ON u.id=e.user_id
		WHERE e.token_hash=? AND e.used_at IS NULL AND u.public_id=? AND u.active=1
		ORDER BY e.id DESC LIMIT 1`, auth.HashToken(token), userID).
		Scan(&internalID, &ciphertext, &expires, &attempts, &enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read MFA enrollment: %w", err)
	}
	if enabled == 1 || !at.Before(parseTime(expires)) || attempts >= maxMFAEnrollmentAttempts {
		_, _ = tx.ExecContext(ctx, `UPDATE mfa_enrollments SET used_at=? WHERE token_hash=? AND used_at IS NULL`, formatTime(at.UTC()), auth.HashToken(token))
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("close expired MFA enrollment: %w", err)
		}
		return false, nil
	}
	secret, err := auth.DecryptSecret(s.secretKey, ciphertext)
	if err != nil {
		return false, err
	}
	if !auth.VerifyTOTP(secret, code, at) {
		attempts++
		usedAt := any(nil)
		if attempts >= maxMFAEnrollmentAttempts {
			usedAt = formatTime(at.UTC())
		}
		if _, err := tx.ExecContext(ctx, `UPDATE mfa_enrollments SET attempts=?,used_at=? WHERE token_hash=?`, attempts, usedAt, auth.HashToken(token)); err != nil {
			return false, fmt.Errorf("record MFA confirmation failure: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("commit MFA confirmation failure: %w", err)
		}
		return false, nil
	}
	// The setup confirmation proves possession but is not a sign-in use. Leave
	// the current step available for the first authenticated login while still
	// preventing a second use of that step afterwards.
	currentStep := at.UTC().Unix()/30 - 1
	if _, err := tx.ExecContext(ctx, `UPDATE users SET mfa_enabled=1,mfa_secret_ciphertext=?,mfa_last_step=? WHERE id=?`, ciphertext, currentStep, internalID); err != nil {
		return false, fmt.Errorf("enable MFA: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM mfa_recovery_codes WHERE user_id=?`, internalID); err != nil {
		return false, fmt.Errorf("replace recovery codes: %w", err)
	}
	for _, hash := range recoveryHashes {
		if strings.TrimSpace(hash) == "" {
			return false, errors.New("recovery code hash is empty")
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO mfa_recovery_codes(user_id,code_hash,created_at) VALUES(?,?,?)`, internalID, hash, formatTime(at.UTC())); err != nil {
			return false, fmt.Errorf("store recovery code: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE mfa_enrollments SET used_at=? WHERE token_hash=?`, formatTime(at.UTC()), auth.HashToken(token)); err != nil {
		return false, fmt.Errorf("close MFA enrollment: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO security_events(user_id,event,details,occurred_at) VALUES(?,?,?,?)`, internalID, "mfa_enabled", "totp and recovery codes", formatTime(at.UTC())); err != nil {
		return false, fmt.Errorf("record MFA audit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit MFA confirmation: %w", err)
	}
	return true, nil
}
