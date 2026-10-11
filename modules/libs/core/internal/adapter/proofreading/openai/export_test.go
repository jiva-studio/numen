package openai

import (
	"context"
	"time"
)

// SetClock replaces the sleep, the instant and the jitter of the retry wait.
func (c *Client) SetClock(
	sleep func(context.Context, time.Duration) error,
	now func() time.Time,
	jitter func(ceiling time.Duration) time.Duration,
) {
	c.sleep, c.now, c.jitter = sleep, now, jitter
}
