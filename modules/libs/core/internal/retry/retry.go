// Package retry holds what every client that sends a request again shares: how
// a service's Retry-After is read, and a wait that ends with its context.
package retry

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// ParseAfter reads a Retry-After value, in seconds or as an HTTP date, into a
// wait measured from now. A value that is absent, negative, past or unreadable
// is zero.
func ParseAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if at, ok := parseDate(value); ok {
		return max(0, at.Sub(now))
	}
	return 0
}

// Wait sleeps for d, or returns the context's error as soon as it ends. A d of
// zero or less only reports the context.
func Wait(ctx context.Context, d time.Duration) error {
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

// dateLayouts are the three forms of an HTTP date.
var dateLayouts = []string{
	"Mon, 02 Jan 2006 15:04:05 GMT",
	time.RFC850,
	time.ANSIC,
}

func parseDate(value string) (time.Time, bool) {
	for _, layout := range dateLayouts {
		if at, err := time.Parse(layout, value); err == nil {
			return at, true
		}
	}
	return time.Time{}, false
}
