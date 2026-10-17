package completion

import (
	"context"
	"sync"
)

// limits holds, for each route, the places its questions wait for.
type limits struct {
	mu    sync.Mutex
	gates map[string]*gate
}

// gate is the places of one route, as many as the limit it was made with.
type gate struct {
	size  int
	slots chan struct{}
}

func newLimits() *limits {
	return &limits{gates: make(map[string]*gate)}
}

// acquire waits for a place under the route's limit and returns the call that
// gives it back. A limit changed since the last question applies to the
// questions that come after; those already answering keep their place under
// the old one. A cancelled context ends the wait.
func (l *limits) acquire(ctx context.Context, route string, size int) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	slots := l.getGate(route, size).slots
	select {
	case slots <- struct{}{}:
		return func() { <-slots }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (l *limits) getGate(route string, size int) *gate {
	l.mu.Lock()
	defer l.mu.Unlock()
	held, found := l.gates[route]
	if !found || held.size != size {
		held = &gate{size: size, slots: make(chan struct{}, size)}
		l.gates[route] = held
	}
	return held
}
