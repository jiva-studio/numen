// Package clock reads the day from the system clock.
package clock

import "time"

// System is the clock of the machine the plugin runs on.
type System struct{}

// GetToday returns the local date as YYYY-MM-DD.
func (System) GetToday() string {
	return time.Now().Format(time.DateOnly)
}
