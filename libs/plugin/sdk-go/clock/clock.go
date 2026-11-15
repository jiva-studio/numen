// Package clock provides the time source plugins receive through their
// constructors: a system clock for the plugin entry point. Tests take the
// settable clock of clocktest.
package clock

import "time"

// System reads the time of the machine. The plugin entry point passes it to
// the packages it constructs.
type System struct{}

// Now gets the current wall-clock time. A plugin may take this clock or
// declare its own; TestTheTypeCheckedWorkspaceObeysItsRules/TheMachineClockIsReadOnlyByAClockPortImplementation
// accepts a read of the machine clock only inside a method that implements a
// clock port.
func (System) Now() time.Time {
	return time.Now()
}
