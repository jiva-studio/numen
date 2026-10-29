// Package clock provides the time source of the plugin.
package clock

import "time"

// NewTicker returns a ticker that sends the current time on its channel after each duration.
func NewTicker(d time.Duration) *time.Ticker {
	return time.NewTicker(d)
}
