package machineport

import "time"

// Clock gets the current time. The domain has no time source of its own, so a
// scenario that stamps a note or queries what is due today receives a Clock,
// and the composition root binds it to the machine's clock.
type Clock interface {
	Now() time.Time
}
