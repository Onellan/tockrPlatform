package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Onellan/tockrplatform/internal/events"
	"github.com/Onellan/tockrplatform/internal/store"
)

const (
	maxProjectionConsumerLength = 100
	maxProjectionEventType      = 200
	maxProjectionAggregateType  = 50
	maxProjectionAggregateID    = 200
)

func validateProjectionConsumer(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxProjectionConsumerLength {
		return store.ErrInvalidProjectionConsumer
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '.' && character != '_' && character != '-' {
			return store.ErrInvalidProjectionConsumer
		}
	}
	return nil
}

func validateProjectionAggregate(aggregateType, aggregateID string) error {
	if strings.TrimSpace(aggregateType) == "" || len(aggregateType) > maxProjectionAggregateType || strings.TrimSpace(aggregateID) == "" || len(aggregateID) > maxProjectionAggregateID {
		return errors.New("projection aggregate identity is invalid")
	}
	return nil
}

func validateProjectionSourceEvent(event store.OutboxEvent) (string, error) {
	if !strings.HasPrefix(event.EventID, "evt_") || len(event.EventID) <= len("evt_") || len(event.EventID) > events.MaxEventIDLength {
		return "", errors.New("projection event identity is invalid")
	}
	if strings.TrimSpace(event.EventType) == "" || len(event.EventType) > maxProjectionEventType || event.Sequence <= 0 || event.SchemaVersion <= 0 || event.OccurredAt.IsZero() {
		return "", errors.New("projection event envelope is invalid")
	}
	if err := validateProjectionAggregate(event.AggregateType, event.AggregateID); err != nil {
		return "", err
	}
	if len(event.Payload) == 0 || len(event.Payload) > events.MaxPayloadBytes || !json.Valid(event.Payload) {
		return "", errors.New("projection event payload is invalid")
	}
	if event.SchemaVersion != events.SchemaVersion {
		return "unknown_schema_version", nil
	}
	validated := events.Event{
		EventID:       event.EventID,
		EventType:     event.EventType,
		AggregateType: events.AggregateType(event.AggregateType),
		AggregateID:   event.AggregateID,
		Sequence:      event.Sequence,
		SchemaVersion: event.SchemaVersion,
		OccurredAt:    event.OccurredAt,
		Payload:       event.Payload,
	}
	if err := validated.Validate(); err != nil {
		return "invalid_event", nil
	}
	return "", nil
}

func (s *Store) IngestPlatformProjectionEvent(ctx context.Context, consumerKey string, event store.OutboxEvent, receivedAt time.Time) (store.ProjectionIngestResult, error) {
	consumerKey = strings.TrimSpace(consumerKey)
	if err := validateProjectionConsumer(consumerKey); err != nil {
		return store.ProjectionIngestResult{}, err
	}
	if receivedAt.IsZero() {
		return store.ProjectionIngestResult{}, errors.New("projection receive time is required")
	}
	blockedReason, err := validateProjectionSourceEvent(event)
	if err != nil {
		return store.ProjectionIngestResult{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.ProjectionIngestResult{}, fmt.Errorf("begin projection ingest: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	existing, exists, err := readProjectionInboxByEvent(ctx, tx, consumerKey, event.EventID)
	if err != nil {
		return store.ProjectionIngestResult{}, err
	}
	if exists {
		if !sameProjectionEvent(existing, event) {
			return store.ProjectionIngestResult{}, store.ErrProjectionEventConflict
		}
		return store.ProjectionIngestResult{Outcome: store.ProjectionOutcomeDuplicate, State: existing.State, Reason: existing.Reason}, nil
	}

	checkpoint, checkpointExists, err := readProjectionCheckpoint(ctx, tx, consumerKey, event.AggregateType, event.AggregateID)
	if err != nil {
		return store.ProjectionIngestResult{}, err
	}
	inboxState := store.ProjectionInboxPending
	outcome := store.ProjectionOutcomeAccepted
	reason := "awaiting_apply"
	if blockedReason != "" {
		inboxState = store.ProjectionInboxBlocked
		outcome = store.ProjectionOutcomeBlocked
		reason = blockedReason
	} else if checkpointExists && checkpoint.State == store.ProjectionBlocked {
		inboxState = store.ProjectionInboxBlocked
		outcome = store.ProjectionOutcomeBlocked
		reason = "prior_blocked_event"
	} else if !checkpointExists && event.Sequence > 1 {
		inboxState = store.ProjectionInboxGap
		outcome = store.ProjectionOutcomeGapFound
		reason = "missing_prior_sequence"
	} else if checkpointExists && event.Sequence <= checkpoint.LastSequence {
		inboxState = store.ProjectionInboxStale
		outcome = store.ProjectionOutcomeStale
		reason = "sequence_already_applied"
	} else if checkpointExists && event.Sequence > checkpoint.LastSequence+1 {
		inboxState = store.ProjectionInboxGap
		outcome = store.ProjectionOutcomeGapFound
		reason = "missing_prior_sequence"
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO platform_projection_inbox(
		consumer_key,event_id,event_type,aggregate_type,aggregate_id,sequence,
		schema_version,occurred_at,payload,state,reason,received_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, consumerKey, event.EventID, event.EventType,
		event.AggregateType, event.AggregateID, event.Sequence, event.SchemaVersion,
		formatTime(event.OccurredAt.UTC()), string(event.Payload), string(inboxState), reason,
		formatTime(receivedAt.UTC())); err != nil {
		return store.ProjectionIngestResult{}, fmt.Errorf("store projection inbox event: %w", err)
	}
	if err := upsertProjectionCheckpointAfterIngest(ctx, tx, consumerKey, event.AggregateType, event.AggregateID, checkpoint, checkpointExists, inboxState, reason, receivedAt.UTC()); err != nil {
		return store.ProjectionIngestResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return store.ProjectionIngestResult{}, fmt.Errorf("commit projection ingest: %w", err)
	}
	return store.ProjectionIngestResult{Outcome: outcome, State: inboxState, Reason: reason}, nil
}

func (s *Store) ListPlatformProjectionInbox(ctx context.Context, consumerKey string, limit int) ([]store.ProjectionInboxEvent, error) {
	consumerKey = strings.TrimSpace(consumerKey)
	if err := validateProjectionConsumer(consumerKey); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 1000 {
		return nil, store.ErrInvalidProjectionLimit
	}
	rows, err := s.db.QueryContext(ctx, `SELECT consumer_key,event_id,event_type,aggregate_type,aggregate_id,
		sequence,schema_version,occurred_at,payload,state,reason,received_at,applied_at
		FROM platform_projection_inbox WHERE consumer_key=? ORDER BY id LIMIT ?`, consumerKey, limit)
	if err != nil {
		return nil, fmt.Errorf("list projection inbox: %w", err)
	}
	defer rows.Close()
	result := make([]store.ProjectionInboxEvent, 0, limit)
	for rows.Next() {
		value, err := scanProjectionInbox(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projection inbox: %w", err)
	}
	return result, nil
}

func (s *Store) MarkPlatformProjectionEventApplied(ctx context.Context, consumerKey, eventID string, appliedAt time.Time) error {
	consumerKey = strings.TrimSpace(consumerKey)
	if err := validateProjectionConsumer(consumerKey); err != nil {
		return err
	}
	if strings.TrimSpace(eventID) == "" || appliedAt.IsZero() {
		return errors.New("projection event and apply time are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin projection apply: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	event, exists, err := readProjectionInboxByEvent(ctx, tx, consumerKey, strings.TrimSpace(eventID))
	if err != nil {
		return err
	}
	if !exists {
		return store.ErrProjectionInboxNotFound
	}
	if event.State == store.ProjectionInboxApplied {
		return nil
	}
	if event.State == store.ProjectionInboxBlocked {
		return store.ErrProjectionBlocked
	}
	if event.State == store.ProjectionInboxGap {
		return store.ErrProjectionGap
	}
	if event.State != store.ProjectionInboxPending {
		return store.ErrProjectionNotReady
	}
	checkpoint, exists, err := readProjectionCheckpoint(ctx, tx, consumerKey, event.AggregateType, event.AggregateID)
	if err != nil {
		return err
	}
	if !exists || checkpoint.State == store.ProjectionBlocked {
		return store.ErrProjectionBlocked
	}
	if checkpoint.State == store.ProjectionUnavailable {
		return store.ErrProjectionUnavailable
	}
	if event.Sequence != checkpoint.LastSequence+1 {
		if event.Sequence > checkpoint.LastSequence+1 {
			return store.ErrProjectionGap
		}
		return store.ErrProjectionNotReady
	}
	if _, err := tx.ExecContext(ctx, `UPDATE platform_projection_inbox SET state=?,reason='',applied_at=? WHERE consumer_key=? AND event_id=? AND state=?`, string(store.ProjectionInboxApplied), formatTime(appliedAt.UTC()), consumerKey, event.EventID, string(store.ProjectionInboxPending)); err != nil {
		return fmt.Errorf("mark projection event applied: %w", err)
	}
	checkpoint.LastSequence = event.Sequence
	checkpoint.LastEventID = event.EventID
	if err := refreshProjectionCheckpointTx(ctx, tx, &checkpoint, appliedAt.UTC()); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ReconcilePlatformProjection(ctx context.Context, consumerKey, aggregateType, aggregateID string, limit int, reconciledAt time.Time) (store.ProjectionReconciliation, error) {
	consumerKey = strings.TrimSpace(consumerKey)
	if err := validateProjectionConsumer(consumerKey); err != nil {
		return store.ProjectionReconciliation{}, err
	}
	if err := validateProjectionAggregate(aggregateType, aggregateID); err != nil {
		return store.ProjectionReconciliation{}, err
	}
	if limit <= 0 || limit > 1000 {
		return store.ProjectionReconciliation{}, store.ErrInvalidProjectionLimit
	}
	if reconciledAt.IsZero() {
		return store.ProjectionReconciliation{}, errors.New("projection reconciliation time is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.ProjectionReconciliation{}, fmt.Errorf("begin projection reconciliation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	checkpoint, exists, err := readProjectionCheckpoint(ctx, tx, consumerKey, aggregateType, aggregateID)
	if err != nil {
		return store.ProjectionReconciliation{}, err
	}
	if !exists {
		return store.ProjectionReconciliation{Checkpoint: unavailableProjectionCheckpoint(consumerKey, aggregateType, aggregateID, reconciledAt.UTC())}, nil
	}
	if checkpoint.State == store.ProjectionBlocked || checkpoint.State == store.ProjectionUnavailable {
		return store.ProjectionReconciliation{Checkpoint: checkpoint}, nil
	}
	promoted := 0
	for promoted < limit {
		var inboxID int64
		var sequence int64
		err := tx.QueryRowContext(ctx, `SELECT id,sequence FROM platform_projection_inbox
			WHERE consumer_key=? AND aggregate_type=? AND aggregate_id=? AND state=?
			ORDER BY sequence,id LIMIT 1`, consumerKey, aggregateType, aggregateID, string(store.ProjectionInboxGap)).Scan(&inboxID, &sequence)
		if errors.Is(err, sql.ErrNoRows) {
			break
		}
		if err != nil {
			return store.ProjectionReconciliation{}, fmt.Errorf("find projection gap: %w", err)
		}
		if sequence != checkpoint.LastSequence+1 {
			break
		}
		if _, err := tx.ExecContext(ctx, `UPDATE platform_projection_inbox SET state=?,reason=? WHERE id=? AND state=?`, string(store.ProjectionInboxPending), "awaiting_apply", inboxID, string(store.ProjectionInboxGap)); err != nil {
			return store.ProjectionReconciliation{}, fmt.Errorf("promote projection gap: %w", err)
		}
		promoted++
		break
	}
	if err := refreshProjectionCheckpointTx(ctx, tx, &checkpoint, reconciledAt.UTC()); err != nil {
		return store.ProjectionReconciliation{}, err
	}
	if err := tx.Commit(); err != nil {
		return store.ProjectionReconciliation{}, fmt.Errorf("commit projection reconciliation: %w", err)
	}
	return store.ProjectionReconciliation{Checkpoint: checkpoint, Promoted: promoted}, nil
}

func (s *Store) GetPlatformProjectionCheckpoint(ctx context.Context, consumerKey, aggregateType, aggregateID string) (store.ProjectionCheckpoint, error) {
	consumerKey = strings.TrimSpace(consumerKey)
	if err := validateProjectionConsumer(consumerKey); err != nil {
		return store.ProjectionCheckpoint{}, err
	}
	if err := validateProjectionAggregate(aggregateType, aggregateID); err != nil {
		return store.ProjectionCheckpoint{}, err
	}
	checkpoint, exists, err := readProjectionCheckpoint(ctx, s.db, consumerKey, aggregateType, aggregateID)
	if err != nil {
		return store.ProjectionCheckpoint{}, err
	}
	if !exists {
		return unavailableProjectionCheckpoint(consumerKey, aggregateType, aggregateID, time.Now().UTC()), nil
	}
	return checkpoint, nil
}

func (s *Store) MarkPlatformProjectionUnavailable(ctx context.Context, consumerKey, aggregateType, aggregateID, reason string, at time.Time) error {
	consumerKey = strings.TrimSpace(consumerKey)
	if err := validateProjectionConsumer(consumerKey); err != nil {
		return err
	}
	if err := validateProjectionAggregate(aggregateType, aggregateID); err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > store.MaxProjectionReasonLength || at.IsZero() {
		return errors.New("projection unavailable reason and time are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin projection unavailable state: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	checkpoint, exists, err := readProjectionCheckpoint(ctx, tx, consumerKey, aggregateType, aggregateID)
	if err != nil {
		return err
	}
	if !exists {
		checkpoint = store.ProjectionCheckpoint{ConsumerKey: consumerKey, AggregateType: aggregateType, AggregateID: aggregateID}
	}
	checkpoint.State = store.ProjectionUnavailable
	checkpoint.Reason = reason
	checkpoint.UpdatedAt = at.UTC()
	if err := upsertProjectionCheckpointTx(ctx, tx, checkpoint); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ResumePlatformProjection(ctx context.Context, consumerKey, aggregateType, aggregateID string, at time.Time) (store.ProjectionCheckpoint, error) {
	consumerKey = strings.TrimSpace(consumerKey)
	if err := validateProjectionConsumer(consumerKey); err != nil {
		return store.ProjectionCheckpoint{}, err
	}
	if err := validateProjectionAggregate(aggregateType, aggregateID); err != nil {
		return store.ProjectionCheckpoint{}, err
	}
	if at.IsZero() {
		return store.ProjectionCheckpoint{}, errors.New("projection resume time is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.ProjectionCheckpoint{}, fmt.Errorf("begin projection resume: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	checkpoint, exists, err := readProjectionCheckpoint(ctx, tx, consumerKey, aggregateType, aggregateID)
	if err != nil {
		return store.ProjectionCheckpoint{}, err
	}
	if !exists || checkpoint.State != store.ProjectionUnavailable {
		if !exists {
			checkpoint = unavailableProjectionCheckpoint(consumerKey, aggregateType, aggregateID, at.UTC())
		}
		return checkpoint, nil
	}
	if err := refreshProjectionCheckpointTx(ctx, tx, &checkpoint, at.UTC()); err != nil {
		return store.ProjectionCheckpoint{}, err
	}
	if err := tx.Commit(); err != nil {
		return store.ProjectionCheckpoint{}, fmt.Errorf("commit projection resume: %w", err)
	}
	return checkpoint, nil
}

type projectionScanner interface {
	Scan(...any) error
}

func scanProjectionInbox(scanner projectionScanner) (store.ProjectionInboxEvent, error) {
	var value store.ProjectionInboxEvent
	var occurredAt, receivedAt string
	var appliedAt sql.NullString
	var payload string
	if err := scanner.Scan(&value.ConsumerKey, &value.EventID, &value.EventType, &value.AggregateType, &value.AggregateID, &value.Sequence, &value.SchemaVersion, &occurredAt, &payload, &value.State, &value.Reason, &receivedAt, &appliedAt); err != nil {
		return store.ProjectionInboxEvent{}, fmt.Errorf("read projection inbox event: %w", err)
	}
	value.OccurredAt = parseTime(occurredAt)
	value.Payload = json.RawMessage(payload)
	value.ReceivedAt = parseTime(receivedAt)
	if appliedAt.Valid {
		applied := parseTime(appliedAt.String)
		value.AppliedAt = &applied
	}
	return value, nil
}

func readProjectionInboxByEvent(ctx context.Context, query interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, consumerKey, eventID string) (store.ProjectionInboxEvent, bool, error) {
	row := query.QueryRowContext(ctx, `SELECT consumer_key,event_id,event_type,aggregate_type,aggregate_id,
		sequence,schema_version,occurred_at,payload,state,reason,received_at,applied_at
		FROM platform_projection_inbox WHERE consumer_key=? AND event_id=?`, consumerKey, eventID)
	value, err := scanProjectionInbox(row)
	if errors.Is(err, sql.ErrNoRows) {
		return store.ProjectionInboxEvent{}, false, nil
	}
	if err != nil {
		return store.ProjectionInboxEvent{}, false, err
	}
	return value, true, nil
}

func sameProjectionEvent(existing store.ProjectionInboxEvent, event store.OutboxEvent) bool {
	return existing.EventID == event.EventID && existing.EventType == event.EventType && existing.AggregateType == event.AggregateType && existing.AggregateID == event.AggregateID && existing.Sequence == event.Sequence && existing.SchemaVersion == event.SchemaVersion && existing.OccurredAt.Equal(event.OccurredAt) && string(existing.Payload) == string(event.Payload)
}

func readProjectionCheckpoint(ctx context.Context, query interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, consumerKey, aggregateType, aggregateID string) (store.ProjectionCheckpoint, bool, error) {
	var value store.ProjectionCheckpoint
	var updatedAt string
	err := query.QueryRowContext(ctx, `SELECT consumer_key,aggregate_type,aggregate_id,last_sequence,last_event_id,state,reason,updated_at
		FROM platform_projection_checkpoints WHERE consumer_key=? AND aggregate_type=? AND aggregate_id=?`, consumerKey, aggregateType, aggregateID).Scan(&value.ConsumerKey, &value.AggregateType, &value.AggregateID, &value.LastSequence, &value.LastEventID, &value.State, &value.Reason, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return store.ProjectionCheckpoint{}, false, nil
	}
	if err != nil {
		return store.ProjectionCheckpoint{}, false, fmt.Errorf("read projection checkpoint: %w", err)
	}
	value.UpdatedAt = parseTime(updatedAt)
	return value, true, nil
}

func upsertProjectionCheckpointAfterIngest(ctx context.Context, tx *sql.Tx, consumerKey, aggregateType, aggregateID string, checkpoint store.ProjectionCheckpoint, exists bool, inboxState store.ProjectionInboxState, reason string, at time.Time) error {
	if !exists {
		checkpoint = store.ProjectionCheckpoint{ConsumerKey: consumerKey, AggregateType: aggregateType, AggregateID: aggregateID, LastSequence: 0, LastEventID: ""}
	}
	if checkpoint.State != store.ProjectionUnavailable && checkpoint.State != store.ProjectionBlocked {
		switch inboxState {
		case store.ProjectionInboxBlocked:
			checkpoint.State = store.ProjectionBlocked
			checkpoint.Reason = reason
		case store.ProjectionInboxGap:
			checkpoint.State = store.ProjectionGap
			checkpoint.Reason = reason
		case store.ProjectionInboxPending:
			if checkpoint.State != store.ProjectionGap {
				checkpoint.State = store.ProjectionStale
				checkpoint.Reason = reason
			}
		}
	}
	checkpoint.UpdatedAt = at
	return upsertProjectionCheckpointTx(ctx, tx, checkpoint)
}

func upsertProjectionCheckpointTx(ctx context.Context, tx *sql.Tx, checkpoint store.ProjectionCheckpoint) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO platform_projection_checkpoints(
		consumer_key,aggregate_type,aggregate_id,last_sequence,last_event_id,state,reason,updated_at)
		VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(consumer_key,aggregate_type,aggregate_id) DO UPDATE SET
		last_sequence=excluded.last_sequence,last_event_id=excluded.last_event_id,state=excluded.state,
		reason=excluded.reason,updated_at=excluded.updated_at`, checkpoint.ConsumerKey, checkpoint.AggregateType,
		checkpoint.AggregateID, checkpoint.LastSequence, checkpoint.LastEventID, string(checkpoint.State),
		checkpoint.Reason, formatTime(checkpoint.UpdatedAt.UTC())); err != nil {
		return fmt.Errorf("upsert projection checkpoint: %w", err)
	}
	return nil
}

func refreshProjectionCheckpointTx(ctx context.Context, tx *sql.Tx, checkpoint *store.ProjectionCheckpoint, at time.Time) error {
	var state, reason string
	var sequence int64
	err := tx.QueryRowContext(ctx, `SELECT state,reason,sequence FROM platform_projection_inbox
		WHERE consumer_key=? AND aggregate_type=? AND aggregate_id=? AND state IN (?,?,?)
		ORDER BY sequence,id LIMIT 1`, checkpoint.ConsumerKey, checkpoint.AggregateType, checkpoint.AggregateID,
		string(store.ProjectionInboxBlocked), string(store.ProjectionInboxGap), string(store.ProjectionInboxPending)).Scan(&state, &reason, &sequence)
	if errors.Is(err, sql.ErrNoRows) {
		checkpoint.State = store.ProjectionCurrent
		checkpoint.Reason = ""
	} else if err != nil {
		return fmt.Errorf("refresh projection checkpoint: %w", err)
	} else {
		switch store.ProjectionInboxState(state) {
		case store.ProjectionInboxBlocked:
			checkpoint.State = store.ProjectionBlocked
			checkpoint.Reason = reason
		case store.ProjectionInboxGap:
			checkpoint.State = store.ProjectionGap
			checkpoint.Reason = reason
		case store.ProjectionInboxPending:
			if sequence == checkpoint.LastSequence+1 {
				checkpoint.State = store.ProjectionStale
				checkpoint.Reason = reason
			} else {
				checkpoint.State = store.ProjectionGap
				checkpoint.Reason = "missing_prior_sequence"
			}
		}
	}
	checkpoint.UpdatedAt = at
	return upsertProjectionCheckpointTx(ctx, tx, *checkpoint)
}

func unavailableProjectionCheckpoint(consumerKey, aggregateType, aggregateID string, at time.Time) store.ProjectionCheckpoint {
	return store.ProjectionCheckpoint{ConsumerKey: consumerKey, AggregateType: aggregateType, AggregateID: aggregateID, State: store.ProjectionUnavailable, Reason: "not_initialized", UpdatedAt: at.UTC()}
}
