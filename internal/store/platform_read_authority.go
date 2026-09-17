package store

import (
	"context"
	"errors"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
)

var (
	ErrInvalidReadAuthorityConsumer    = errors.New("read-authority consumer is invalid")
	ErrInvalidReadAuthorityRequest     = errors.New("read-authority snapshot request is invalid")
	ErrReadAuthoritySnapshotNotFound   = errors.New("read-authority snapshot not found")
	ErrReadAuthoritySnapshotExpired    = errors.New("read-authority snapshot expired")
	ErrReadAuthoritySnapshotIncomplete = errors.New("read-authority snapshot is incomplete")
	ErrReadAuthorityChecksumMismatch   = errors.New("read-authority snapshot checksum mismatch")
	ErrReadAuthorityCursorInvalid      = errors.New("read-authority snapshot cursor is invalid")
)

const (
	MaxReadAuthoritySnapshotRecords = 10000
	MaxReadAuthoritySnapshotPage    = 500
	MaxReadAuthoritySnapshotTTL     = 24 * time.Hour
)

// ReadAuthoritySnapshotStore is the narrow capability seam for immutable
// Platform bootstrap snapshots. Callers know records and cursors, never SQL
// tables or migration representation.
type ReadAuthoritySnapshotStore interface {
	CreateReadAuthoritySnapshot(context.Context, string, []string, time.Duration, int, time.Time) (domain.ReadAuthoritySnapshot, error)
	GetReadAuthoritySnapshot(context.Context, string, string, time.Time) (domain.ReadAuthoritySnapshot, error)
	ListReadAuthoritySnapshotRecords(context.Context, string, string, string, string, int, time.Time) (domain.ReadAuthoritySnapshotPage, error)
	CleanupReadAuthoritySnapshots(context.Context, string, time.Time, int) (int64, error)
}
