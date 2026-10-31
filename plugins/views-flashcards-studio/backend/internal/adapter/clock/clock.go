// Package clock provides the tickers and the entropy for the use cases.
package clock

import (
	"crypto/rand"
	"time"
)

// NewTicker returns a ticker that sends the current time on its channel after each duration.
func NewTicker(d time.Duration) *time.Ticker {
	return time.NewTicker(d)
}

// ReadEntropy fills b with cryptographically secure random bytes. The runtime aborts the program
// when the system has no randomness, so reading never fails.
func ReadEntropy(b []byte) {
	rand.Read(b)
}
