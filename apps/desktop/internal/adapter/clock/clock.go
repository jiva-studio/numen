// Package clock provides the application time source.
package clock

import "time"

// Now gets the current time.
func Now() time.Time {
	return time.Now()
}
