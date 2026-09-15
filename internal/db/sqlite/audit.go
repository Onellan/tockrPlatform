package sqlite

import (
	"context"
	"fmt"
	"time"
)

func (s *Store) RecordSecurityEvent(ctx context.Context, publicUserID *string, event, details string, at time.Time) error {
	var internalID any
	if publicUserID != nil && *publicUserID != "" {
		var id int64
		if err := s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE public_id=?`, *publicUserID).Scan(&id); err != nil {
			return fmt.Errorf("resolve audit user: %w", err)
		}
		internalID = id
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO security_events(user_id,event,details,occurred_at) VALUES(?,?,?,?)`, internalID, event, details, formatTime(at.UTC()))
	return err
}

func (s *Store) SecurityEventCount(ctx context.Context, event string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM security_events WHERE event=?`, event).Scan(&count)
	return count, err
}

func (s *Store) UserCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}
