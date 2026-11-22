// Package clock is the time of the machine.
package clock

import (
	"context"
	"time"

	sdk "github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
)

// System reads the clock of the machine.
type System struct{}

// Now returns the time of the machine.
func (System) Now() time.Time { return sdk.System{}.Now() }

// Wait returns after d, or with the error of ctx when it ends first.
func (System) Wait(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
