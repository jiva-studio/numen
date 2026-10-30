// Package clock reads the current time and random bytes.
package clock

import (
	"crypto/rand"
	"time"
)

// Now returns the current time.
func Now() time.Time {
	return time.Now()
}

// NewTicker returns a ticker that sends the current time on its channel after each duration.
func NewTicker(d time.Duration) *time.Ticker {
	return time.NewTicker(d)
}

// ReadEntropy fills b with cryptographically secure random bytes. The runtime aborts the program
// when the system has no randomness, so reading never fails.
func ReadEntropy(b []byte) {
	rand.Read(b)
}
