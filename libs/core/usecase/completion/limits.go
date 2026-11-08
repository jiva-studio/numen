package completion

import (
	"context"
	"sync"
)

// limits holds, for each agent, the places its questions wait for.
type limits struct {
	mu    sync.Mutex
	gates map[string]*gate
}

// gate is the places of one agent, as many as the limit it was made with.
type gate struct {
	size  int
	slots chan struct{}
}

func newLimits() *limits {
	return &limits{gates: make(map[string]*gate)}
}

func (l *limits) acquire(ctx context.Context, agent string, size int) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	slots := l.getGate(agent, size).slots
	select {
	case slots <- struct{}{}:
		return func() { <-slots }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (l *limits) getGate(agent string, size int) *gate {
	l.mu.Lock()
	defer l.mu.Unlock()
	current, found := l.gates[agent]
	if !found || current.size != size {
		current = &gate{size: size, slots: make(chan struct{}, size)}
		l.gates[agent] = current
	}
	return current
}
