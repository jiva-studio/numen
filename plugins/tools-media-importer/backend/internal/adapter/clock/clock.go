// Package clock is the plugin's source of the current time.
package clock

import "time"

// Now gets the current local time.
func Now() time.Time {
	return time.Now()
}
