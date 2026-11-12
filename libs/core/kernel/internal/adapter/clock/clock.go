// Package clock provides the system clock of the core, for the composition root.
package clock

import (
	"context"
	"time"
)

// System reads the time of the machine. The composition root passes it to the
// packages it constructs.
type System struct{}

// Now gets the current wall-clock time. This is the only read of the machine
// clock in the workspace; TestTheTypeCheckedWorkspaceObeysItsRules/TheMachineClockIsReadOnlyByAClockPortImplementation
// enforces it.
func (System) Now() time.Time {
	return time.Now()
}

// After returns a channel that receives the time once d has passed.
func (System) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// Wait waits for d, and returns early with the error of ctx where ctx ends first. A d of zero or less returns
// the error of ctx, which is nil while ctx is live.
func (System) Wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
