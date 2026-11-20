package container

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// installationClock is a Clock that also waits, for work that waits out a quiet period or a delay before a retry.
type installationClock interface {
	machineport.Clock
	After(d time.Duration) <-chan time.Time
	Wait(ctx context.Context, d time.Duration) error
}
