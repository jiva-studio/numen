package container

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port/machineport"
)

// GetWait is the wait every scenario assembled here is handed, bound to this
// machine's timers.
func (c Config) GetWait() machineport.Wait {
	return func(ctx context.Context, d time.Duration) error {
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
}

// GetDelay is the delayed call every scenario assembled here is handed, bound
// to this machine's timers.
func (c Config) GetDelay() machineport.Delay {
	return func(d time.Duration, f func()) func() bool {
		return time.AfterFunc(d, f).Stop
	}
}
