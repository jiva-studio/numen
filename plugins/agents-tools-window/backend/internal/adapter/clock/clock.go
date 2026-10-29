// Package clock provides the wall-clock time source of the plugin.
package clock

import "time"

// Now gets the current time.
func Now() time.Time {
	return time.Now()
}
