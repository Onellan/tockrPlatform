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

type LoginCredential struct {
	User         domain.User
	PasswordHash string
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
	FindLoginCredential(context.Context, string) (*LoginCredential, error)
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
	CleanupExpiredSessions(context.Context, time.Time, int) (int64, error)
	VerifyMFA(context.Context, string, string) (bool, error)
	UseRecoveryCode(context.Context, string, string, time.Time) (bool, error)
	CreateMFAEnrollment(context.Context, string, string, time.Time) (string, error)
	CompleteMFAEnrollment(context.Context, string, string, string, []string, time.Time) (bool, error)
}

type SecurityEventStore interface {
	RecordSecurityEvent(context.Context, *string, string, string, time.Time) error
}

type PlatformStore interface {
	AuthenticationStore
	SessionStore
	SecurityEventStore
	OrganisationStore
	WorkspaceStore
	AccessScopeStore
}
