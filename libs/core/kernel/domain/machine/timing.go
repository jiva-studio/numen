package machine

import (
	"context"
	"time"
)

// Wait holds the caller for d, or returns the context's error as soon as the
// context ends. A scenario that waits is handed one, and the composition root
// binds it to the machine's timers.
type Wait func(ctx context.Context, d time.Duration) error
