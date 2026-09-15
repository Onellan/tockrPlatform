package store

import (
	"context"
	"time"

	"github.com/Onellan/tockrplatform/internal/domain"
)

type AuthenticatedSession struct {
	Session Session
	User    domain.User
}

type Session struct {
	UserID    string
	CSRFToken string
	ExpiresAt time.Time
}

// AuthenticationStore is the narrow caller contract for Platform identity
// and credential lifecycle. The implementation may retain credential fields
// internally, but authenticated projections must not expose them.
type AuthenticationStore interface {
	CreateUser(context.Context, domain.User, string) (domain.User, error)
	FindUserByEmail(context.Context, string) (*domain.User, error)
	FindUserByID(context.Context, string) (*domain.User, error)
	SetUserActive(context.Context, string, bool) error
	TouchLogin(context.Context, string, time.Time) error
}

type SessionStore interface {
	CreateSession(context.Context, string, string, string, time.Time) error
	EstablishSession(context.Context, string, string, string, time.Time, time.Time) error
	AuthenticatedSession(context.Context, string, time.Time) (AuthenticatedSession, error)
	VerifySessionCSRF(context.Context, string, string) (bool, error)
	RevokeSession(context.Context, string, time.Time) error
	RevokeSessionAndRecordLogout(context.Context, string, string, time.Time) error
}

type SecurityEventStore interface {
	RecordSecurityEvent(context.Context, *string, string, string, time.Time) error
}

type PlatformStore interface {
	AuthenticationStore
	SessionStore
	SecurityEventStore
}
