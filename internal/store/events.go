package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrInvalidOutboxLimit = errors.New("outbox limit must be between 1 and 1000")

type OutboxEvent struct {
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

type EventStore interface {
	ListPendingPlatformEvents(context.Context, int) ([]OutboxEvent, error)
	MarkPlatformEventsPublished(context.Context, []string, time.Time) error
}
