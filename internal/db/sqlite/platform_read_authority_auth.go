package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Onellan/tockrplatform/internal/store"
)

func (s *Store) ReserveReadAuthorityNonce(ctx context.Context, consumer, nonce string, reservedAt, expiresAt time.Time) error {
	consumer = strings.TrimSpace(consumer)
	nonce = strings.TrimSpace(nonce)
	if consumer == "" || len(consumer) > 100 || nonce == "" || len(nonce) > 128 || reservedAt.IsZero() || expiresAt.IsZero() || !expiresAt.After(reservedAt) {
		return store.ErrInvalidReadAuthorityNonce
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin read-authority nonce reservation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM platform_read_authority_nonces WHERE expires_at<=?`, formatTime(reservedAt.UTC())); err != nil {
		return fmt.Errorf("cleanup read-authority nonces: %w", err)
	}
	var exists int
	err = tx.QueryRowContext(ctx, `SELECT 1 FROM platform_read_authority_nonces WHERE consumer_key=? AND nonce=?`, consumer, nonce).Scan(&exists)
	if err == nil && exists == 1 {
		return store.ErrReadAuthorityNonceReplay
	}
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("check read-authority nonce: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO platform_read_authority_nonces(consumer_key,nonce,reserved_at,expires_at) VALUES(?,?,?,?)`, consumer, nonce, formatTime(reservedAt.UTC()), formatTime(expiresAt.UTC())); err != nil {
		return fmt.Errorf("reserve read-authority nonce: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit read-authority nonce: %w", err)
	}
	return nil
}
