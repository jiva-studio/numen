package provision

import (
	"context"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

func getFlightKey(req pluginport.DependencyRequirement) string {
	return strings.Join([]string{string(req.Kind), req.ID, req.Version, req.SourceURL, req.ChecksumSHA256, req.PluginDir}, "\x00")
}

// flightGroup runs one call per key at a time. A caller that arrives while a
// call runs joins it and receives the last reported status and every later
// one. A call outlives the contexts of its callers and ends only when it returns.
type flightGroup struct {
	mu      sync.Mutex
	flights map[string]*flight
}

type flight struct {
	cancel      context.CancelFunc
	completion  chan struct{}
	subscribers map[int]func(pluginport.DependencyStatus)
	next        int
	last        *pluginport.DependencyStatus
	resolved    pluginport.ResolvedDependency
	err         error
}

func (g *flightGroup) isRunning(key string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	_, ok := g.flights[key]
	return ok
}

func (g *flightGroup) do(
	ctx context.Context,
	key string,
	onProgress func(pluginport.DependencyStatus),
	run func(ctx context.Context, report func(pluginport.DependencyStatus)) (pluginport.ResolvedDependency, error),
) (pluginport.ResolvedDependency, error) {
	g.mu.Lock()
	if g.flights == nil {
		g.flights = make(map[string]*flight)
	}
	current, joined := g.flights[key]
	if !joined {
		runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
		current = &flight{cancel: cancel, completion: make(chan struct{}), subscribers: make(map[int]func(pluginport.DependencyStatus))}
		g.flights[key] = current
		go g.execute(runCtx, key, current, run)
	}
	id := current.next
	current.next++
	if onProgress != nil {
		current.subscribers[id] = onProgress
		if current.last != nil {
			onProgress(*current.last)
		}
	}
	g.mu.Unlock()

	select {
	case <-current.completion:
		return current.resolved, current.err
	case <-ctx.Done():
		g.mu.Lock()
		delete(current.subscribers, id)
		g.mu.Unlock()
		return pluginport.ResolvedDependency{}, ctx.Err()
	}
}

func (g *flightGroup) execute(
	ctx context.Context,
	key string,
	current *flight,
	run func(ctx context.Context, report func(pluginport.DependencyStatus)) (pluginport.ResolvedDependency, error),
) {
	report := func(status pluginport.DependencyStatus) {
		g.mu.Lock()
		current.last = &status
		listeners := make([]func(pluginport.DependencyStatus), 0, len(current.subscribers))
		for _, listener := range current.subscribers {
			listeners = append(listeners, listener)
		}
		g.mu.Unlock()
		for _, listener := range listeners {
			listener(status)
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
