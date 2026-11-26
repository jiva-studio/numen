package synctest

import (
	"sync"
	"time"
)

// Jitter is a source of random durations set by hand: the n-th pick is the n-th fraction of its limit, and the
// fractions run round again after the last.
type Jitter struct {
	mu        sync.Mutex
	fractions []float64
	next      int
}

// NewJitter returns the source that picks the fractions of the limits in turn. A fraction of 1 picks the limit and a
// fraction of 0 picks nothing.
func NewJitter(fractions ...float64) *Jitter {
	return &Jitter{mu: sync.Mutex{}, fractions: fractions, next: 0}
}

func (j *Jitter) Pick(limit time.Duration) time.Duration {
	j.mu.Lock()
	defer j.mu.Unlock()
	fraction := j.fractions[j.next%len(j.fractions)]
	j.next++
	return time.Duration(float64(limit) * fraction)
}
