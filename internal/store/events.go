package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrInvalidOutboxLimit = errors.New("outbox limit must be between 1 and 1000")

var (
	ErrReadAuthoritySourceGap     = errors.New("read-authority source cursor gap")
	ErrReadAuthoritySourceInvalid = errors.New("read-authority source event is invalid")
)

type OutboxEvent struct {
	GlobalCursor  int64
	EventID       string
	EventType     string
	AggregateType string
	AggregateID   string
	Sequence      int64
	SchemaVersion int
	OccurredAt    time.Time
	Payload       json.RawMessage
	PublishedAt   *time.Time
}

type PlatformEventFeed struct {
	Events           []OutboxEvent
	SourceCursor     int64
	RetentionHorizon int64
	NextCursor       int64
	HasMore          bool
}

type EventStore interface {
	ListPendingPlatformEvents(context.Context, int) ([]OutboxEvent, error)
	MarkPlatformEventsPublished(context.Context, []string, time.Time) error
	ListPlatformEventsAfter(context.Context, int64, int) (PlatformEventFeed, error)
	PlatformEventCursorBounds(context.Context) (int64, int64, error)
}
