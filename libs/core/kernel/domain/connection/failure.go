package connection

import (
	"errors"
	"strings"
	"time"
)

// ErrNoConnection is a plugin with no row that can serve the operation.
var ErrNoConnection = errors.New("no connection serves the operation")

// Failure is one row that did not answer: the name of its connection, the reason in words and the reason as
// a code. A row set aside for a reason outside the enum has the code FailureReasonUnavailable.
type Failure struct {
	ConnectionName string
	Reason         string
	Code           FailureReason
}

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

// StoppedError is a connection that found the request itself wrong, so no other row was tried. Reason is
// the words of Code.
type StoppedError struct {
	ConnectionName string
	Reason         string
	Code           FailureReason
}

// Error names the connection and the reason.
func (e *StoppedError) Error() string { return e.ConnectionName + ": " + e.Reason }

// FailureReason is why a call to a provider failed. The provider names it from an enum; the core sees
// no HTTP code and no text from a service. The zero value is FailureReasonUnavailable.
type FailureReason int

// The reasons of a failure.
const (
	FailureReasonUnavailable FailureReason = iota
	FailureReasonRejectedKey
	FailureReasonPaymentRequired
	FailureReasonRateLimited
	FailureReasonInvalidRequest
)

// String returns the words the person reads for the reason.
func (r FailureReason) String() string {
	switch r {
	case FailureReasonRejectedKey:
		return "key rejected"
	case FailureReasonPaymentRequired:
		return "payment required"
	case FailureReasonRateLimited:
		return "rate limited"
	case FailureReasonInvalidRequest:
		return "invalid request"
	case FailureReasonUnavailable:
		return "unavailable"
	}
	return "unavailable"
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

// SetAside is a connection a call set aside for a reason, and NextName, the name of the connection of a later
// row that answered the call.
type SetAside struct {
	ConnectionID   string
	ConnectionName string
	Reason         FailureReason
	NextName       string
}
