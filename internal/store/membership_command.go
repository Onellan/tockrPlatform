package store

import (
	"context"
	"errors"
	"time"
)

var (
	ErrMembershipCommandConflict       = errors.New("membership command conflict")
	ErrMembershipCommandReplayMismatch = errors.New("membership command idempotency key payload mismatch")
	ErrMembershipCommandUnsupported    = errors.New("membership command operation is unsupported")
)

type MembershipCommandRequest struct {
	Consumer        string
	Scope           string
	Operation       string
	OrganisationID  string
	WorkspaceID     string
	UserID          string
	Role            string
	Reason          string
	IdempotencyKey  string
	ExpectedVersion int64
	RequestHash     string
	ActorUserID     string
	OccurredAt      time.Time
}

type MembershipCommandResult struct {
	MembershipID      string
	OrganisationID    string
	WorkspaceID       string
	UserID            string
	Role              string
	Active            bool
	MembershipVersion int64
	Replay            bool
}

type MembershipCommandStore interface {
	ExecuteMembershipCommand(context.Context, MembershipCommandRequest) (MembershipCommandResult, error)
}
