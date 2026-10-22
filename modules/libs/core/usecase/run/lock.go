package run

import (
	"context"
	"slices"
	"sync"
)

// The priorities a run waits at, the first of them served first. Work a person
// is sitting in front of goes before work the vault set itself.
const (
	priorityAsked = iota
	priorityUnasked
	priorities
)

// A Lock is the models and the processor this machine reads with, held by one
// run at a time. A run waits at the priority its work belongs to and is served
// in the order it arrived there.
type Lock struct {
	mu      sync.Mutex
	isHeld  bool
	waiting [priorities][]chan struct{}
}

// Acquire waits for the lock and returns what releases it. waiting is called
// where the lock is not free, so that a person watching is told why nothing is
// moving. A context that ends while waiting acquires nothing.
func (g *Lock) Acquire(ctx context.Context, asked bool, waiting func()) (func(), error) {
	at := priorityUnasked
	if asked {
		at = priorityAsked
	}

	g.mu.Lock()
	if !g.isHeld {
		g.isHeld = true
		g.mu.Unlock()
		return g.release, nil
	}
	stand := make(chan struct{})
	g.waiting[at] = append(g.waiting[at], stand)
	g.mu.Unlock()

	if waiting != nil {
		waiting()
	}
	select {
	case <-stand:
		return g.release, nil
	case <-ctx.Done():
		g.leave(at, stand)
		return nil, ctx.Err()
	}
}

// release hands the lock to whoever has waited longest at the first priority
func (g *Lock) release() {
	g.mu.Lock()
	defer g.mu.Unlock()
	for at := range g.waiting {
		if len(g.waiting[at]) > 0 {
			next := g.waiting[at][0]
			g.waiting[at] = g.waiting[at][1:]
			close(next)
			return
		}
	}
	g.isHeld = false
}

// holds it, and hands it on.
func (g *Lock) leave(at int, stand chan struct{}) {
	g.mu.Lock()
	for i, one := range g.waiting[at] {
		if one == stand {
			g.waiting[at] = slices.Delete(g.waiting[at], i, i+1)
			g.mu.Unlock()
			return
		}
	}
	g.mu.Unlock()
	<-stand
	g.release()
}

func (g *Lock) HasWaiting() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, line := range g.waiting {
		if len(line) > 0 {
			return true
		}
	}
	return false
}
