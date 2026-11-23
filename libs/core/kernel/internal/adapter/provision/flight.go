package provision

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// flightGroup runs one call per key at a time. A caller that arrives while a
// call runs joins it and receives the last reported status and every later
// one. A call outlives the contexts of its callers and ends only when it returns.
type flightGroup struct {
	mu      sync.Mutex
	flights map[string]*flight
}

// subscriber delivers statuses to one caller one at a time, so that a replay completes before any later status.
type subscriber struct {
	mu      sync.Mutex
	deliver func(plugin.DependencyStatus)
}

type flight struct {
	cancel      context.CancelFunc
	canceled    bool
	completion  chan struct{}
	subscribers map[int]*subscriber
	next        int
	last        *plugin.DependencyStatus
	resolved    plugin.ResolvedDependency
	err         error
}

func (g *flightGroup) do(
	ctx context.Context,
	key string,
	handleProgress func(plugin.DependencyStatus),
	run func(ctx context.Context, report func(plugin.DependencyStatus)) (plugin.ResolvedDependency, error),
) (plugin.ResolvedDependency, error) {
	g.mu.Lock()
	if g.flights == nil {
		g.flights = make(map[string]*flight)
	}
	current, joined := g.flights[key]
	if joined && current.canceled {
		g.mu.Unlock()
		select {
		case <-current.completion:
			return g.do(ctx, key, handleProgress, run)
		case <-ctx.Done():
			return plugin.ResolvedDependency{}, ctx.Err()
		}
	}
	if !joined {
		runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
		current = &flight{cancel: cancel, completion: make(chan struct{}), subscribers: make(map[int]*subscriber)}
		g.flights[key] = current
		go g.execute(runCtx, key, current, run)
	}
	id := current.next
	current.next++
	var joiner *subscriber
	var replay *plugin.DependencyStatus
	if handleProgress != nil {
		joiner = &subscriber{deliver: handleProgress}
		joiner.mu.Lock()
		current.subscribers[id] = joiner
		replay = current.last
	}
	g.mu.Unlock()
	if joiner != nil {
		if replay != nil {
			joiner.deliver(*replay)
		}
		joiner.mu.Unlock()
	}

	select {
	case <-current.completion:
		return current.resolved, current.err
	case <-ctx.Done():
		g.mu.Lock()
		delete(current.subscribers, id)
		g.mu.Unlock()
		return plugin.ResolvedDependency{}, ctx.Err()
	}
}

func (g *flightGroup) cancel(key string) bool {
	g.mu.Lock()
	current, ok := g.flights[key]
	ok = ok && !current.canceled
	if ok {
		current.canceled = true
	}
	g.mu.Unlock()
	if ok {
		current.cancel()
	}
	return ok
}

func (g *flightGroup) cancelAll() {
	g.mu.Lock()
	running := make([]*flight, 0, len(g.flights))
	for _, current := range g.flights {
		current.canceled = true
		running = append(running, current)
	}
	g.mu.Unlock()
	for _, current := range running {
		current.cancel()
	}
}

func (g *flightGroup) execute(
	ctx context.Context,
	key string,
	current *flight,
	run func(ctx context.Context, report func(plugin.DependencyStatus)) (plugin.ResolvedDependency, error),
) {
	report := func(status plugin.DependencyStatus) {
		g.mu.Lock()
		current.last = &status
		listeners := make([]*subscriber, 0, len(current.subscribers))
		for _, listener := range current.subscribers {
			listeners = append(listeners, listener)
		}
		g.mu.Unlock()
		for _, listener := range listeners {
			listener.mu.Lock()
			listener.deliver(status)
			listener.mu.Unlock()
		}
	}
	current.resolved, current.err = run(ctx, report)
	current.cancel()
	g.mu.Lock()
	if g.flights[key] == current {
		delete(g.flights, key)
	}
	g.mu.Unlock()
	close(current.completion)
}
