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

// Wait holds the caller for d, or returns the context's error as soon as the
// context ends. A scenario that waits is handed one, and the composition root
// binds it to the machine's timers.
type Wait func(ctx context.Context, d time.Duration) error

// Delay calls f once after d and returns a cancel function. The cancel function
// reports whether it stopped f before f began.
type Delay func(d time.Duration, f func()) (cancel func() bool)
