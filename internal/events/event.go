package events

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	SchemaVersion    = 1
	MaxPayloadBytes  = 4096
	MaxEventIDLength = 100
)

type AggregateType string

const (
	AggregateUser         AggregateType = "User"
	AggregateOrganisation AggregateType = "Organisation"
	AggregateWorkspace    AggregateType = "Workspace"
	AggregateProduct      AggregateType = "Product"
	AggregateAccess       AggregateType = "Access"
)

const (
	EventUserCreated                   = "platform.user.created"
	EventUserStatusChanged             = "platform.user.status_changed"
	EventOrganisationCreated           = "platform.organisation.created"
	EventOrganisationMembershipAdded   = "platform.organisation.membership_added"
	EventOrganisationRoleChanged       = "platform.organisation.membership_role_changed"
	EventOrganisationMembershipRemoved = "platform.organisation.membership_deactivated"
	EventOrganisationArchived          = "platform.organisation.archived"
	EventOrganisationRenamed           = "platform.organisation.renamed"
	EventWorkspaceCreated              = "platform.workspace.created"
	EventWorkspaceMembershipAdded      = "platform.workspace.membership_added"
	EventWorkspaceRoleChanged          = "platform.workspace.membership_role_changed"
	EventWorkspaceMembershipRemoved    = "platform.workspace.membership_deactivated"
	EventWorkspaceArchived             = "platform.workspace.archived"
	EventProductRetired                = "platform.product.retired"
	EventProductCreated                = "platform.product.created"
	EventEntitlementGranted            = "platform.access.entitlement_granted"
	EventEntitlementRevoked            = "platform.access.entitlement_revoked"
	EventAssignmentGranted             = "platform.access.assignment_granted"
	EventAssignmentRevoked             = "platform.access.assignment_revoked"
)

var (
	ErrInvalidEvent         = errors.New("invalid platform event")
	ErrInvalidEventPayload  = errors.New("invalid platform event payload")
	ErrUnknownEventType     = errors.New("unknown platform event type")
	ErrInvalidAggregateID   = errors.New("invalid platform event aggregate id")
	ErrInvalidEventSequence = errors.New("invalid platform event sequence")
)

type Event struct {
	EventID       string
	EventType     string
	AggregateType AggregateType
	AggregateID   string
	Sequence      int64
	SchemaVersion int
	OccurredAt    time.Time
	Payload       json.RawMessage
}

type eventSpec struct {
	aggregateType AggregateType
	fields        map[string]fieldType
}

type fieldType uint8

const (
	fieldString fieldType = iota + 1
	fieldBool
)

var specs = map[string]eventSpec{
	EventUserCreated:                   {AggregateUser, fields("user_id", fieldString, "active", fieldBool)},
	EventUserStatusChanged:             {AggregateUser, fields("user_id", fieldString, "active", fieldBool)},
	EventOrganisationCreated:           {AggregateOrganisation, fields("organisation_id", fieldString)},
	EventOrganisationMembershipAdded:   {AggregateOrganisation, fields("organisation_id", fieldString, "membership_id", fieldString, "user_id", fieldString, "role", fieldString, "active", fieldBool)},
	EventOrganisationRoleChanged:       {AggregateOrganisation, fields("organisation_id", fieldString, "membership_id", fieldString, "user_id", fieldString, "role", fieldString, "active", fieldBool)},
	EventOrganisationMembershipRemoved: {AggregateOrganisation, fields("organisation_id", fieldString, "membership_id", fieldString, "user_id", fieldString, "role", fieldString, "active", fieldBool)},
	EventOrganisationArchived:          {AggregateOrganisation, fields("organisation_id", fieldString, "active", fieldBool)},
	EventOrganisationRenamed:           {AggregateOrganisation, fields("organisation_id", fieldString, "name", fieldString)},
	EventWorkspaceCreated:              {AggregateWorkspace, fields("workspace_id", fieldString, "organisation_id", fieldString, "active", fieldBool)},
	EventWorkspaceMembershipAdded:      {AggregateWorkspace, fields("workspace_id", fieldString, "membership_id", fieldString, "user_id", fieldString, "role", fieldString, "active", fieldBool)},
	EventWorkspaceRoleChanged:          {AggregateWorkspace, fields("workspace_id", fieldString, "membership_id", fieldString, "user_id", fieldString, "role", fieldString, "active", fieldBool)},
	EventWorkspaceMembershipRemoved:    {AggregateWorkspace, fields("workspace_id", fieldString, "membership_id", fieldString, "user_id", fieldString, "role", fieldString, "active", fieldBool)},
	EventWorkspaceArchived:             {AggregateWorkspace, fields("workspace_id", fieldString, "active", fieldBool)},
	EventProductRetired:                {AggregateProduct, fields("product_key", fieldString, "status", fieldString)},
	EventProductCreated:                {AggregateProduct, fields("product_key", fieldString, "status", fieldString)},
	EventEntitlementGranted:            {AggregateAccess, fields("entitlement_id", fieldString, "organisation_id", fieldString, "product_key", fieldString, "status", fieldString)},
	EventEntitlementRevoked:            {AggregateAccess, fields("entitlement_id", fieldString, "organisation_id", fieldString, "product_key", fieldString, "status", fieldString)},
	EventAssignmentGranted:             {AggregateAccess, fields("assignment_id", fieldString, "organisation_id", fieldString, "user_id", fieldString, "product_key", fieldString, "status", fieldString)},
	EventAssignmentRevoked:             {AggregateAccess, fields("assignment_id", fieldString, "organisation_id", fieldString, "user_id", fieldString, "product_key", fieldString, "status", fieldString)},
}

func fields(values ...any) map[string]fieldType {
	result := make(map[string]fieldType, len(values)/2)
	for index := 0; index < len(values); index += 2 {
		result[values[index].(string)] = values[index+1].(fieldType)
	}
	return result
}

func New(eventType, aggregateID string, occurredAt time.Time, payload map[string]any) (Event, error) {
	spec, ok := specs[eventType]
	if !ok {
		return Event{}, ErrUnknownEventType
	}
	if occurredAt.IsZero() {
		return Event{}, ErrInvalidEvent
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return Event{}, fmt.Errorf("encode platform event payload: %w", err)
	}
	eventID, err := NewEventID()
	if err != nil {
		return Event{}, err
	}
	event := Event{
		EventID:       eventID,
		EventType:     eventType,
		AggregateType: spec.aggregateType,
		AggregateID:   strings.TrimSpace(aggregateID),
		SchemaVersion: SchemaVersion,
		OccurredAt:    occurredAt.UTC(),
		Payload:       payloadBytes,
	}
	if err := event.validate(false); err != nil {
		return Event{}, err
	}
	return event, nil
}

func NewEventID() (string, error) {
	var raw [18]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate platform event id: %w", err)
	}
	return "evt_" + base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func (e Event) WithSequence(sequence int64) (Event, error) {
	e.Sequence = sequence
	if err := e.validate(true); err != nil {
		return Event{}, err
	}
	return e, nil
}

func (e Event) Validate() error { return e.validate(true) }

func (e Event) MarshalJSON() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		EventID       string          `json:"event_id"`
		EventType     string          `json:"event_type"`
		AggregateType AggregateType   `json:"aggregate_type"`
		AggregateID   string          `json:"aggregate_id"`
		Sequence      int64           `json:"sequence"`
		SchemaVersion int             `json:"schema_version"`
		OccurredAt    time.Time       `json:"occurred_at"`
		Payload       json.RawMessage `json:"payload"`
	}{e.EventID, e.EventType, e.AggregateType, e.AggregateID, e.Sequence, e.SchemaVersion, e.OccurredAt.UTC(), e.Payload})
}

func (e Event) validate(requireSequence bool) error {
	spec, ok := specs[e.EventType]
	if !ok || e.AggregateType != spec.aggregateType {
		return ErrUnknownEventType
	}
	if !strings.HasPrefix(e.EventID, "evt_") || len(e.EventID) <= len("evt_") || len(e.EventID) > MaxEventIDLength {
		return ErrInvalidEvent
	}
	if !validAggregateID(e.AggregateType, e.AggregateID) {
		return ErrInvalidAggregateID
	}
	if e.SchemaVersion != SchemaVersion || e.OccurredAt.IsZero() {
		return ErrInvalidEvent
	}
	if requireSequence && e.Sequence <= 0 {
		return ErrInvalidEventSequence
	}
	if len(e.Payload) == 0 || len(e.Payload) > MaxPayloadBytes {
		return ErrInvalidEventPayload
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(e.Payload, &values); err != nil || values == nil {
		return ErrInvalidEventPayload
	}
	if len(values) != len(spec.fields) {
		return ErrInvalidEventPayload
	}
	for name, kind := range spec.fields {
		raw, exists := values[name]
		if !exists || string(raw) == "null" {
			return ErrInvalidEventPayload
		}
		switch kind {
		case fieldString:
			var value string
			if json.Unmarshal(raw, &value) != nil || !validPayloadString(name, value) {
				return ErrInvalidEventPayload
			}
		case fieldBool:
			var value bool
			if json.Unmarshal(raw, &value) != nil {
				return ErrInvalidEventPayload
			}
		default:
			return ErrInvalidEventPayload
		}
	}
	for name := range values {
		if _, exists := spec.fields[name]; !exists {
			return ErrInvalidEventPayload
		}
	}
	return nil
}

func validPayloadString(name, value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	switch name {
	case "user_id":
		return strings.HasPrefix(value, "usr_") && len(value) > len("usr_")
	case "organisation_id":
		return strings.HasPrefix(value, "org_") && len(value) > len("org_")
	case "workspace_id":
		return strings.HasPrefix(value, "wsp_") && len(value) > len("wsp_")
	case "membership_id":
		return (strings.HasPrefix(value, "omem_") && len(value) > len("omem_")) || (strings.HasPrefix(value, "wmem_") && len(value) > len("wmem_"))
	case "entitlement_id":
		return strings.HasPrefix(value, "ent_") && len(value) > len("ent_")
	case "assignment_id":
		return strings.HasPrefix(value, "upa_") && len(value) > len("upa_")
	case "product_key":
		return strings.HasPrefix(value, "product.") && len(value) > len("product.")
	case "role":
		return value == "owner" || value == "admin" || value == "member" || value == "viewer"
	case "status":
		return value == "active" || value == "archived" || value == "retired" || value == "revoked"
	case "name":
		return len(value) <= 200
	default:
		return false
	}
}

func validAggregateID(aggregateType AggregateType, value string) bool {
	value = strings.TrimSpace(value)
	switch aggregateType {
	case AggregateUser:
		return strings.HasPrefix(value, "usr_") && len(value) > len("usr_")
	case AggregateOrganisation:
		return strings.HasPrefix(value, "org_") && len(value) > len("org_")
	case AggregateWorkspace:
		return strings.HasPrefix(value, "wsp_") && len(value) > len("wsp_")
	case AggregateProduct:
		return strings.HasPrefix(value, "product.") && len(value) > len("product.")
	case AggregateAccess:
		return (strings.HasPrefix(value, "ent_") && len(value) > len("ent_")) || (strings.HasPrefix(value, "upa_") && len(value) > len("upa_"))
	default:
		return false
	}
}
