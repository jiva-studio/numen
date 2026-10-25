package retry_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/retry"
)

var now = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

func TestRetryAfterIsReadInSecondsAndAsADate(t *testing.T) {
	for name, tc := range map[string]struct {
		value string
		want  time.Duration
	}{
		"seconds":     {"7", 7 * time.Second},
		"padded":      {" 7 ", 7 * time.Second},
		"a date":      {now.Add(12 * time.Second).Format(http.TimeFormat), 12 * time.Second},
		"a past date": {now.Add(-time.Minute).Format(http.TimeFormat), 0},
		"negative":    {"-3", 0},
		"absent":      {"", 0},
		"unreadable":  {"soon", 0},
	} {
		t.Run(name, func(t *testing.T) {
			if got := retry.ParseAfter(tc.value, now); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAWaitEndsWithItsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	start := time.Now()
	if err := retry.Wait(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Error("the wait did not end with the context")
	}
}

func TestAWaitOfNothingReportsTheContext(t *testing.T) {
	if err := retry.Wait(t.Context(), 0); err != nil {
		t.Errorf("got %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := retry.Wait(ctx, 0); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v", err)
	}
}

func TestAWaitElapses(t *testing.T) {
	if err := retry.Wait(t.Context(), time.Millisecond); err != nil {
		t.Errorf("got %v", err)
	}
}

func TestRetryAfterOfMinusOneSecondIsNoWait(t *testing.T) {
	if got := retry.ParseAfter("-1", now); got != 0 {
		t.Errorf("got %v, want no wait", got)
	}
}

// A wait of nothing reports the context and never takes the timer's branch,
// whichever of the two is ready first.
func TestAWaitOfNothingNeverOutlastsAnEndedContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	for _, d := range []time.Duration{0, -time.Second} {
		for range 500 {
			if err := retry.Wait(ctx, d); !errors.Is(err, context.Canceled) {
				t.Fatalf("a wait of %v on an ended context = %v", d, err)
			}
		}
	}
}
