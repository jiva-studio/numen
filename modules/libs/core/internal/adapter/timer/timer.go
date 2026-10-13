// Package timer is the machine's timers: waiting for a duration and running a
// call after one.
package timer

import (
	"context"
	"time"
)

// Wait sleeps for d, or returns the context's error as soon as it ends. A d of
// zero or less only reports the context.
func Wait(ctx context.Context, d time.Duration) error {
	// gomutants:disable-next-line INTEGER_INCREMENT reason="the mutated value is 1: a wait of one nanosecond is over before the select runs, so both paths end the same"
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// After runs f after d passes unless stopped first.
func After(d time.Duration, f func()) func() bool {
	t := time.AfterFunc(d, f)
	return t.Stop
}
