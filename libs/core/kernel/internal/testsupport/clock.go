package testsupport

import (
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
)

// Epoch is the moment that the clock of a test reports.
func GetEpoch() time.Time { return time.Date(2026, time.March, 2, 12, 0, 0, 0, time.UTC) }

// Machine reads the time of the machine. A test uses it where it compares a
// reading with the modification times of files or measures elapsed time.
func GetMachine() clock.System { return clock.System{} }

// NewClock creates a clock that reports Epoch until it is moved.
func NewClock() *clock.Fake {
	return clock.NewFake(GetEpoch())
}
