package machineport

import (
	"context"
	"time"
)

// Clock is what time it is. The domain knows nothing that has a lifetime, so a
// scenario that stamps a note or asks what is due today is handed one, and the
// composition root binds it to the machine's.
//
// A scenario that takes no clock has none: there is no reading of the machine
// behind a clock nobody gave, because a note stamped an hour out and a card due
// on the wrong day are both wrong quietly.
type Clock func() time.Time

// Wait holds the caller for d, or returns the context's error as soon as the
// context ends. A scenario that waits is handed one, and the composition root
// binds it to the machine's timers.
type Wait func(ctx context.Context, d time.Duration) error

// Delay calls f once after d and returns what cancels the call. The cancel says
// whether it stopped f before f began.
type Delay func(d time.Duration, f func()) (cancel func() bool)
