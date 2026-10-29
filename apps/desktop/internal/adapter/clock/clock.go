// Package clock provides the application time source.
package clock

import "time"

// Now returns the current time.
func Now() time.Time {
	return time.Now()
}
