package connection

import (
	"errors"
	"time"
)

// Reason is why a call failed. The core shows a reason and sees no HTTP code and no text from a service.
type Reason int

// The reasons of a failure. The zero value is ReasonUnavailable.
const (
	ReasonUnavailable Reason = iota
	ReasonRejectedKey
	ReasonPaymentRequired
	ReasonRateLimited
	ReasonInvalidRequest
)

// String returns the words the person reads for the reason.
func (r Reason) String() string {
	switch r {
	case ReasonRejectedKey:
		return "key rejected"
	case ReasonPaymentRequired:
		return "payment required"
	case ReasonRateLimited:
		return "rate limited"
	case ReasonInvalidRequest:
		return "invalid request"
	case ReasonUnavailable:
		return "unavailable"
	}
	return "unavailable"
}

type failure struct {
	reason     Reason
	isTryNext  bool
	retryAfter time.Duration
}

func (f *failure) Error() string { return f.reason.String() }

// NewTryNextError is "this connection cannot serve now, try the next": a rejected key, a payment
// required, a limit of rate with its Retry-After, or a service that is unreachable, slow or failing on
// its side. retryAfter is zero when the service gave none.
func NewTryNextError(reason Reason, retryAfter time.Duration) error {
	return &failure{reason: reason, isTryNext: true, retryAfter: max(retryAfter, 0)}
}

// NewStopError is "the request itself is wrong": the core goes to no other connection.
func NewStopError(reason Reason) error { return &failure{reason: reason} }

// IsTryNext reports whether err is, or wraps, a try-next failure, and the wait the service asked for.
func IsTryNext(err error) (retryAfter time.Duration, ok bool) {
	var one *failure
	if errors.As(err, &one) && one.isTryNext {
		return one.retryAfter, true
	}
	return 0, false
}

// IsStop reports whether err is, or wraps, a stop failure.
func IsStop(err error) bool {
	var one *failure
	return errors.As(err, &one) && !one.isTryNext
}

// getReason returns the reason of err where err is, or wraps, a try-next or a stop failure.
func getReason(err error) (Reason, bool) {
	if one, ok := errors.AsType[*failure](err); ok {
		return one.reason, true
	}
	return ReasonUnavailable, false
}
