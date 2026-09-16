package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

const MaxProjectionReasonLength = 500

var (
	ErrInvalidProjectionConsumer = errors.New("projection consumer key is invalid")
	ErrInvalidProjectionLimit    = errors.New("projection limit must be between 1 and 1000")
	ErrProjectionInboxNotFound   = errors.New("projection inbox event not found")
	ErrProjectionEventConflict   = errors.New("projection event identity conflict")
	ErrProjectionNotReady        = errors.New("projection event is not ready")
	ErrProjectionBlocked         = errors.New("projection is blocked")
	ErrProjectionUnavailable     = errors.New("projection is unavailable")
	ErrProjectionGap             = errors.New("projection has an unresolved gap")
)

type ProjectionInboxState string

const (
	ProjectionInboxPending ProjectionInboxState = "pending"
	ProjectionInboxApplied ProjectionInboxState = "applied"
	ProjectionInboxGap     ProjectionInboxState = "gap"
	ProjectionInboxStale   ProjectionInboxState = "stale"
	ProjectionInboxBlocked ProjectionInboxState = "blocked"
)

type ProjectionState string

const (
	ProjectionCurrent     ProjectionState = "current"
	ProjectionStale       ProjectionState = "stale"
	ProjectionGap         ProjectionState = "gap"
	ProjectionBlocked     ProjectionState = "blocked"
	ProjectionUnavailable ProjectionState = "unavailable"
)

type ProjectionIngestOutcome string

const (
	ProjectionOutcomeAccepted  ProjectionIngestOutcome = "accepted"
	ProjectionOutcomeDuplicate ProjectionIngestOutcome = "duplicate"
	ProjectionOutcomeStale     ProjectionIngestOutcome = "stale"
	ProjectionOutcomeBlocked   ProjectionIngestOutcome = "blocked"
	ProjectionOutcomeGapFound  ProjectionIngestOutcome = "gap"
)

type ProjectionInboxEvent struct {
	ConsumerKey   string
	EventID       string
	EventType     string
	AggregateType string
	AggregateID   string
	Sequence      int64
	SchemaVersion int
	OccurredAt    time.Time
	Payload       json.RawMessage
	State         ProjectionInboxState
	Reason        string
	ReceivedAt    time.Time
	AppliedAt     *time.Time
}

type ProjectionIngestResult struct {
	Outcome ProjectionIngestOutcome
	State   ProjectionInboxState
	Reason  string
}

type ProjectionCheckpoint struct {
	ConsumerKey   string
	AggregateType string
	AggregateID   string
	LastSequence  int64
	LastEventID   string
	State         ProjectionState
	Reason        string
	UpdatedAt     time.Time
}

type ProjectionReconciliation struct {
	Checkpoint ProjectionCheckpoint
	Promoted   int
}

type ProjectionStore interface {
	IngestPlatformProjectionEvent(context.Context, string, OutboxEvent, time.Time) (ProjectionIngestResult, error)
	ListPlatformProjectionInbox(context.Context, string, int) ([]ProjectionInboxEvent, error)
	MarkPlatformProjectionEventApplied(context.Context, string, string, time.Time) error
	ReconcilePlatformProjection(context.Context, string, string, string, int, time.Time) (ProjectionReconciliation, error)
	GetPlatformProjectionCheckpoint(context.Context, string, string, string) (ProjectionCheckpoint, error)
	MarkPlatformProjectionUnavailable(context.Context, string, string, string, string, time.Time) error
	ResumePlatformProjection(context.Context, string, string, string, time.Time) (ProjectionCheckpoint, error)
}
