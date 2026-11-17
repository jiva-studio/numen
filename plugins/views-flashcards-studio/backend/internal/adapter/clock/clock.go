// Package clock provides the tickers and the entropy for the use cases.
package clock

import (
	"crypto/rand"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// Tickers creates tickers of the machine clock.
type Tickers struct{}

// CreateTicker returns a ticker that sends the current time on its channel after each duration.
func (Tickers) CreateTicker(d time.Duration) ports.Ticker {
	return ticker{t: time.NewTicker(d)}
}

type ticker struct{ t *time.Ticker }

func (t ticker) GetChannel() <-chan time.Time { return t.t.C }

func (t ticker) Reset(d time.Duration) { t.t.Reset(d) }

func (t ticker) Stop() { t.t.Stop() }

// Entropy is the cryptographically secure randomness of the machine.
type Entropy struct{}

// Fill fills b with random bytes. The runtime aborts the program when the system has no
// randomness, so filling never fails.
func (Entropy) Fill(b []byte) {
	rand.Read(b)
}
