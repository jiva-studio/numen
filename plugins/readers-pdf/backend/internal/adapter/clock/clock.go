// Package clock reads the current time.
package clock

import (
	"time"

	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

// AfterFunc waits for the duration to elapse and then calls f in its own goroutine.
func AfterFunc(d time.Duration, f func()) pdf.Timer {
	return time.AfterFunc(d, f)
}
