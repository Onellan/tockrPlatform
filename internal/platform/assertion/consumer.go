package assertion

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type FailureClass string

const (
	FailureUnauthenticated FailureClass = "unauthenticated"
	FailureForbidden       FailureClass = "forbidden"
	FailureStale           FailureClass = "stale"
	FailureUnavailable     FailureClass = "unavailable"
	FailureVersionMismatch FailureClass = "version_mismatch"
)

var ErrAssertionVersionMismatch = errors.New("platform assertion version is not supported")

type ConsumerValidationError struct {
	Class FailureClass
}

func (e ConsumerValidationError) Error() string {
	return fmt.Sprintf("consumer assertion validation failed: %s", e.Class)
}

func (e ConsumerValidationError) Unwrap() error {
	return nil
}

func FailureClassOf(err error) FailureClass {
	var validationErr ConsumerValidationError
	if errors.As(err, &validationErr) {
		return validationErr.Class
	}
	return FailureUnauthenticated
}

func ProductForAudience(audience string) (string, bool) {
	switch strings.TrimSpace(audience) {
	case "tockrctrl":
		return "product.tockrctrl", true
	case "tockrims":
		return "product.tockrims", true
	default:
		return "", false
	}
}

// VerifyForProduct is the bounded consumer contract. Product adapters provide
// their expected audience and stable Platform product key; Platform does not
// inspect or issue product-specific roles.
func (i *Issuer) VerifyForProduct(token, expectedAudience, expectedProductKey string, now time.Time) (Claims, error) {
	productKey, ok := ProductForAudience(expectedAudience)
	if !ok || productKey != strings.TrimSpace(expectedProductKey) {
		return Claims{}, ConsumerValidationError{Class: FailureForbidden}
	}
	claims, err := i.Verify(token, now)
	if err != nil {
		return Claims{}, ConsumerValidationError{Class: classifyVerificationFailure(err)}
	}
	if claims.Audience != strings.TrimSpace(expectedAudience) {
		return Claims{}, ConsumerValidationError{Class: FailureForbidden}
	}
	return claims, nil
}

func classifyVerificationFailure(err error) FailureClass {
	switch {
	case errors.Is(err, ErrConfiguration):
		return FailureUnavailable
	case errors.Is(err, ErrAssertionVersionMismatch):
		return FailureVersionMismatch
	case errors.Is(err, ErrAssertionExpired), errors.Is(err, ErrAssertionNotYetValid), errors.Is(err, ErrAssertionReplay):
		return FailureStale
	case errors.Is(err, ErrInvalidAudience), errors.Is(err, ErrInvalidScope):
		return FailureForbidden
	default:
		return FailureUnauthenticated
	}
}
