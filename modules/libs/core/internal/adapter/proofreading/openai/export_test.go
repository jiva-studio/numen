package openai

import (
	"context"
	"net/http"
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

// GetHTTPClient is the client every request is sent with.
func (c *Client) GetHTTPClient() *http.Client { return c.http }

// SetSleep replaces the sleep of the retry wait, keeping the instant and the
// jitter.
func (c *Client) SetSleep(sleep func(context.Context, time.Duration) error) { c.sleep = sleep }
