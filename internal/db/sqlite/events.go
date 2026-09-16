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

func appendPlatformEventTx(ctx context.Context, tx *sql.Tx, eventType, aggregateID string, occurredAt time.Time, payload map[string]any) error {
	event, err := events.New(eventType, aggregateID, occurredAt, payload)
	if err != nil {
		return err
	}
	var sequence int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence),0)+1 FROM platform_outbox WHERE aggregate_type=? AND aggregate_id=?`, event.AggregateType, event.AggregateID).Scan(&sequence); err != nil {
		return fmt.Errorf("next platform event sequence: %w", err)
	}
	event, err = event.WithSequence(sequence)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO platform_outbox(event_id,event_type,aggregate_type,aggregate_id,sequence,schema_version,occurred_at,payload) VALUES(?,?,?,?,?,?,?,?)`, event.EventID, event.EventType, event.AggregateType, event.AggregateID, event.Sequence, event.SchemaVersion, formatTime(event.OccurredAt), string(event.Payload)); err != nil {
		return fmt.Errorf("append platform outbox event: %w", err)
	}
	return nil
}

func appendOrganisationEventTx(ctx context.Context, tx *sql.Tx, eventName string, details organisationMutationDetails, at time.Time) error {
	payload := map[string]any{"organisation_id": details.OrganisationID}
	eventType := ""
	switch eventName {
	case eventOrganisationCreated:
		eventType = events.EventOrganisationCreated
	case eventMembershipAdded:
		eventType = events.EventOrganisationMembershipAdded
		payload["membership_id"] = details.MembershipID
		payload["user_id"] = details.UserID
		payload["role"] = details.Role
		payload["active"] = true
	case eventMembershipRoleChanged:
		eventType = events.EventOrganisationRoleChanged
		payload["membership_id"] = details.MembershipID
		payload["user_id"] = details.UserID
		payload["role"] = details.Role
		payload["active"] = true
	case eventMembershipDeactivated:
		eventType = events.EventOrganisationMembershipRemoved
		payload["membership_id"] = details.MembershipID
		payload["user_id"] = details.UserID
		payload["role"] = details.Role
		payload["active"] = false
	case eventOrganisationArchived:
		eventType = events.EventOrganisationArchived
		payload["active"] = false
	case eventOrganisationRenamed:
		eventType = events.EventOrganisationRenamed
		payload["name"] = details.Name
	}
	if eventType == "" {
		return fmt.Errorf("unsupported organisation audit event %q", eventName)
	}
	return appendPlatformEventTx(ctx, tx, eventType, details.OrganisationID, at, payload)
}

func appendWorkspaceEventTx(ctx context.Context, tx *sql.Tx, eventName string, details workspaceMutationDetails, organisationID string, at time.Time) error {
	payload := map[string]any{"workspace_id": details.WorkspaceID}
	eventType := ""
	switch eventName {
	case eventWorkspaceCreated:
		eventType = events.EventWorkspaceCreated
		payload["organisation_id"] = organisationID
		payload["active"] = true
	case eventWorkspaceMembershipAdded:
		eventType = events.EventWorkspaceMembershipAdded
		payload["membership_id"] = details.MembershipID
		payload["user_id"] = details.UserID
		payload["role"] = details.Role
		payload["active"] = true
	case eventWorkspaceRoleChanged:
		eventType = events.EventWorkspaceRoleChanged
		payload["membership_id"] = details.MembershipID
		payload["user_id"] = details.UserID
		payload["role"] = details.Role
		payload["active"] = true
	case eventWorkspaceMemberRemoved:
		eventType = events.EventWorkspaceMembershipRemoved
		payload["membership_id"] = details.MembershipID
		payload["user_id"] = details.UserID
		payload["role"] = details.Role
		payload["active"] = false
	case eventWorkspaceArchived:
		eventType = events.EventWorkspaceArchived
		payload["active"] = false
	}
	if eventType == "" {
		return fmt.Errorf("unsupported workspace audit event %q", eventName)
	}
	return appendPlatformEventTx(ctx, tx, eventType, details.WorkspaceID, at, payload)
}

func appendProductEventTx(ctx context.Context, tx *sql.Tx, eventName string, details productAuditDetails, at time.Time) error {
	if eventName != eventProductRetired {
		return fmt.Errorf("unsupported product audit event %q", eventName)
	}
	return appendPlatformEventTx(ctx, tx, events.EventProductRetired, details.ProductKey, at, map[string]any{
		"product_key": details.ProductKey,
		"status":      details.Status,
	})
}

func appendAccessEventTx(ctx context.Context, tx *sql.Tx, eventName string, details productAuditDetails, at time.Time) error {
	eventType := ""
	aggregateID := ""
	payload := map[string]any{
		"organisation_id": details.OrganisationID,
		"product_key":     details.ProductKey,
		"status":          details.Status,
	}
	switch eventName {
	case eventEntitlementGranted:
		eventType = events.EventEntitlementGranted
		aggregateID = details.EntitlementID
		payload["entitlement_id"] = details.EntitlementID
	case eventEntitlementRevoked:
		eventType = events.EventEntitlementRevoked
		aggregateID = details.EntitlementID
		payload["entitlement_id"] = details.EntitlementID
	case eventAssignmentGranted:
		eventType = events.EventAssignmentGranted
		aggregateID = details.AssignmentID
		payload["assignment_id"] = details.AssignmentID
		payload["user_id"] = details.UserID
	case eventAssignmentRevoked:
		eventType = events.EventAssignmentRevoked
		aggregateID = details.AssignmentID
		payload["assignment_id"] = details.AssignmentID
		payload["user_id"] = details.UserID
	default:
		return fmt.Errorf("unsupported access audit event %q", eventName)
	}
	return appendPlatformEventTx(ctx, tx, eventType, aggregateID, at, payload)
}

func (s *Store) ListPendingPlatformEvents(ctx context.Context, limit int) ([]store.OutboxEvent, error) {
	if limit <= 0 || limit > 1000 {
		return nil, store.ErrInvalidOutboxLimit
	}
	rows, err := s.db.QueryContext(ctx, `SELECT event_id,event_type,aggregate_type,aggregate_id,sequence,schema_version,occurred_at,payload,published_at FROM platform_outbox WHERE published_at IS NULL ORDER BY id LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending platform events: %w", err)
	}
	defer rows.Close()
	result := make([]store.OutboxEvent, 0, limit)
	for rows.Next() {
		var event store.OutboxEvent
		var occurredAt string
		var payload string
		var publishedAt sql.NullString
		if err := rows.Scan(&event.EventID, &event.EventType, &event.AggregateType, &event.AggregateID, &event.Sequence, &event.SchemaVersion, &occurredAt, &payload, &publishedAt); err != nil {
			return nil, fmt.Errorf("read pending platform event: %w", err)
		}
		event.OccurredAt = parseTime(occurredAt)
		event.Payload = json.RawMessage(payload)
		if !json.Valid(event.Payload) {
			return nil, errors.New("pending platform event payload is invalid")
		}
		if publishedAt.Valid {
			value := parseTime(publishedAt.String)
			event.PublishedAt = &value
		}
		result = append(result, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending platform events: %w", err)
	}
	return result, nil
}

func (s *Store) MarkPlatformEventsPublished(ctx context.Context, eventIDs []string, at time.Time) error {
	if at.IsZero() {
		return errors.New("platform event publication time is required")
	}
	if len(eventIDs) == 0 || len(eventIDs) > 1000 {
		return store.ErrInvalidOutboxLimit
	}
	seen := make(map[string]struct{}, len(eventIDs))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin platform event publication: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, eventID := range eventIDs {
		eventID = strings.TrimSpace(eventID)
		if eventID == "" {
			return errors.New("platform event ID is required")
		}
		if _, exists := seen[eventID]; exists {
			return errors.New("duplicate platform event ID")
		}
		seen[eventID] = struct{}{}
		result, err := tx.ExecContext(ctx, `UPDATE platform_outbox SET published_at=? WHERE event_id=? AND published_at IS NULL`, formatTime(at.UTC()), eventID)
		if err != nil {
			return fmt.Errorf("mark platform event published: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("check platform event publication: %w", err)
		}
		if rows != 1 {
			return errors.New("platform event is missing or already published")
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit platform event publication: %w", err)
	}
	return nil
}
