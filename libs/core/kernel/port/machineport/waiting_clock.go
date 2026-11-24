package machineport

import (
	"context"
	"time"
)

// WaitingClock is a Clock that also waits, for work that waits out a quiet period or a delay before a retry.
type WaitingClock interface {
	Clock
	After(d time.Duration) <-chan time.Time
	Wait(ctx context.Context, d time.Duration) error
}
