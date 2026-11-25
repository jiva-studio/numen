package machineport

import (
	"time"
)

// Clock gets the current time. The domain has no time source of its own, so a
// scenario that stamps a note or queries what is due today receives a Clock,
// and the composition root binds it to the machine's clock.
type Clock interface {
	Now() time.Time
}

// InstallationClock is the time source of an installation: the time now and a signal after a delay.
type InstallationClock interface {
	Clock
	After(d time.Duration) <-chan time.Time
}
