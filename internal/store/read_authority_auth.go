package store

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidReadAuthorityNonce = errors.New("read-authority nonce is invalid")
	ErrReadAuthorityNonceReplay  = errors.New("read-authority nonce was already used")
)

// ReadAuthorityNonceStore durably reserves a verified machine nonce. Signature
// verification remains at the HTTP boundary; persistence owns replay safety.
type ReadAuthorityNonceStore interface {
	ReserveReadAuthorityNonce(context.Context, string, string, time.Time, time.Time) error
}
