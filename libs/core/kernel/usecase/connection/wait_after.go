package connection

import (
	"context"
	"time"
)

// NewWait returns a Wait that waits for d on the timer channel after gives, and returns early with the error of ctx
// where ctx ends first. A d of zero or less returns the error of ctx, which is nil while ctx is live.
func NewWait(after func(d time.Duration) <-chan time.Time) Wait {
	return func(ctx context.Context, d time.Duration) error {
		if d <= 0 {
			return ctx.Err()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-after(d):
			return nil
		}
	}
}
