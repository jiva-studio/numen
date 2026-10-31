package testsupport

import (
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/clock"
)

// Epoch is the moment that the clock of a test reports.
var Epoch = time.Date(2026, time.March, 2, 12, 0, 0, 0, time.UTC)

// Machine reads the time of the machine. A test uses it where it compares a
// reading with the modification times of files or measures elapsed time.
var Machine clock.System

// NewClock creates a clock that reports Epoch until it is moved.
func NewClock() *clock.Fake {
	return clock.NewFake(Epoch)
}
