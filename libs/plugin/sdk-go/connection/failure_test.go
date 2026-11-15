package connection

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestATryNextErrorCarriesItsReasonAndRetryAfter(t *testing.T) {
	err := fmt.Errorf("call: %w", NewTryNextError(ReasonRateLimited, 30*time.Second))
	retryAfter, isTryNext := IsTryNext(err)
	if !isTryNext || retryAfter != 30*time.Second {
		t.Fatalf("IsTryNext = %v, %v, want 30s, true", retryAfter, isTryNext)
	}
	if reason, ok := getReason(err); !ok || reason != ReasonRateLimited {
		t.Errorf("getReason = %v, %v, want rate limited", reason, ok)
	}
	if IsStop(err) {
		t.Error("a try-next error is a stop")
	}
	for _, d := range []time.Duration{0, -time.Nanosecond} {
		got, ok := IsTryNext(NewTryNextError(ReasonRateLimited, d))
		if !ok || got != 0 {
			t.Errorf("IsTryNext(%v) = %v, %v, want 0, true", d, got, ok)
		}
	}
}

func TestAStopErrorCarriesItsReason(t *testing.T) {
	err := fmt.Errorf("call: %w", NewStopError(ReasonInvalidRequest))
	if !IsStop(err) {
		t.Fatal("IsStop = false")
	}
	if reason, ok := getReason(err); !ok || reason != ReasonInvalidRequest {
		t.Errorf("getReason = %v, %v, want invalid request", reason, ok)
	}
	if retryAfter, isTryNext := IsTryNext(err); isTryNext || retryAfter != 0 {
		t.Errorf("IsTryNext(stop) = %v, %v, want 0, false", retryAfter, isTryNext)
	}
}

func TestAPlainErrorIsNeitherFailure(t *testing.T) {
	err := errors.New("HTTP 500: internal text of a service")
	if _, isTryNext := IsTryNext(err); isTryNext || IsStop(err) {
		t.Error("a plain error is a failure kind")
	}
	if _, ok := getReason(err); ok {
		t.Error("a plain error has a reason")
	}
}

func TestAFailureErrorTextNamesTheReasonAndNoServiceText(t *testing.T) {
	for reason, want := range map[Reason]string{
		ReasonRejectedKey:     "key rejected",
		ReasonPaymentRequired: "payment required",
		ReasonRateLimited:     "rate limited",
		ReasonUnavailable:     "unavailable",
		ReasonInvalidRequest:  "invalid request",
	} {
		if got := NewStopError(reason).Error(); got != want {
			t.Errorf("NewStopError(%d).Error() = %q, want %q", reason, got, want)
		}
	}
}
