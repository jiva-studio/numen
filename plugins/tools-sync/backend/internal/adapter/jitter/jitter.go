// Package jitter is the randomness of the waits of the plugin.
package jitter

import (
	"math/rand/v2"
	"sync"
	"time"
)

// Random picks durations from a source of random numbers.
type Random struct {
	mu     sync.Mutex
	source *rand.Rand
}

// NewRandom returns the picker over the source.
func NewRandom(source *rand.Rand) *Random { return &Random{mu: sync.Mutex{}, source: source} }

// NewSystem returns the picker over a source seeded at random.
func NewSystem() *Random { return NewRandom(rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))) }

// Pick returns a duration from 0 to limit, both included.
func (r *Random) Pick(limit time.Duration) time.Duration {
	if limit <= 0 {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return time.Duration(r.source.Int64N(int64(limit) + 1))
}
