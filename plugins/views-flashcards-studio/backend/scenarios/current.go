package scenarios

import (
	"sync/atomic"
	"time"
)

// Current holds the scenarios in force. Calls already running finish under the scenarios they
// began under.
type Current struct {
	inputs Inputs
	held   atomic.Pointer[Scenarios]
}

// NewCurrent builds the scenarios from the inputs and holds them.
func NewCurrent(in Inputs) *Current {
	c := &Current{inputs: in, held: atomic.Pointer[Scenarios]{}}
	c.held.Store(New(in))
	return c
}

// Get returns the scenarios in force.
func (c *Current) Get() *Scenarios { return c.held.Load() }

// SetDayStart builds the scenarios again with the time past midnight at which a day of review begins.
func (c *Current) SetDayStart(starts time.Duration) {
	in := c.inputs
	in.DayStart = func() time.Duration { return starts }
	c.held.Store(New(in))
}
