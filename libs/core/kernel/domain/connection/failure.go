package connection

import (
	"errors"
	"strings"
	"time"
)

// ErrNoConnection is a plugin with no row that can serve the operation.
var ErrNoConnection = errors.New("no connection serves the operation")

// Failure is one row that did not answer: the name of its connection and the reason.
type Failure struct{ ConnectionName, Reason string }

// AllFailedError is every row of a plugin set aside or failed.
type AllFailedError struct{ Failures []Failure }

// Error names each connection and its reason, in the order of the rows.
func (e *AllFailedError) Error() string {
	parts := make([]string, len(e.Failures))
	for i, one := range e.Failures {
		parts[i] = one.ConnectionName + ": " + one.Reason
	}
	return strings.Join(parts, "; ")
}

// RejectedError is a connection that found the request itself wrong.
type RejectedError struct{ ConnectionName, Reason string }

// Error names the connection and the reason.
func (e *RejectedError) Error() string { return e.ConnectionName + ": " + e.Reason }

// FailureReason is why a call to a provider failed. The provider names it from an enum; the core sees
// no HTTP code and no text from a service. The zero value is FailureReasonUnknown.
type FailureReason int

// The reasons of a failure.
const (
	FailureReasonUnknown FailureReason = iota
	FailureReasonKeyRejected
	FailureReasonPaymentRequired
	FailureReasonRateLimited
	FailureReasonUnreachable
	FailureReasonTimeout
	FailureReasonBadRequest
)

// String returns the words the person reads for the reason.
func (r FailureReason) String() string {
	switch r {
	case FailureReasonKeyRejected:
		return "key rejected"
	case FailureReasonPaymentRequired:
		return "payment required"
	case FailureReasonRateLimited:
		return "rate limited"
	case FailureReasonUnreachable:
		return "unreachable"
	case FailureReasonTimeout:
		return "timeout"
	case FailureReasonBadRequest:
		return "bad request"
	case FailureReasonUnknown:
		return "unknown"
	}
	return "unknown"
}

// TryNextError is a connection that cannot serve now. RetryAfter is zero when the service gave no wait.
type TryNextError struct {
	Reason     FailureReason
	RetryAfter time.Duration
}

// Error returns the reason.
func (e *TryNextError) Error() string { return e.Reason.String() }

// StopError is a request that is wrong in itself, so no other connection is tried.
type StopError struct{ Reason FailureReason }

// Error returns the reason.
func (e *StopError) Error() string { return e.Reason.String() }

var (
	// ErrNotFound is a request for a connection that is not in the list.
	ErrNotFound = errors.New("no such connection")
	// ErrTypeUnavailable is a type that no installed and enabled plugin provides.
	ErrTypeUnavailable = errors.New("no installed plugin provides the connection type")
	// ErrKeyStore is a key store that fails or that this machine does not have.
	ErrKeyStore = errors.New("the key store failed")
	// ErrStaleRevision is an update that carries an older revision than the connection has.
	ErrStaleRevision = errors.New("the connection changed since it was read")
	// ErrSecretMissing is a required secret field with no value stored for the address the connection has.
	ErrSecretMissing = errors.New("a required secret has no value")
)
