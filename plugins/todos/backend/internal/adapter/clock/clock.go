// Package clock reads the day from the system clock.
package clock

import "time"

// System is the clock of the machine the plugin runs on.
type System struct{}

// Now returns the time of the machine.
func (System) Now() time.Time {
	return time.Now()
}
