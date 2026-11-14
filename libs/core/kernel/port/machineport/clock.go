package machineport

import (
	"context"
	"time"
)

// Clock gets the current time. The domain has no time source of its own, so a
// scenario that stamps a note or queries what is due today receives a Clock,
// and the composition root binds it to the machine's clock.
type Clock interface {
	Now() time.Time
}

// Timer is a Clock that also waits, for work that waits out a quiet period or a delay before a retry.
type Timer interface {
	Clock
	After(d time.Duration) <-chan time.Time
	Wait(ctx context.Context, d time.Duration) error
}
