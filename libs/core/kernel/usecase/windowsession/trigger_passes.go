package windowsession

import (
	"context"
	"maps"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// triggerPasses is the passes of the triggers asked for and not run yet: one over the whole vault, or
// one over the paths that changed. A new one holds a pass over the whole vault.
type triggerPasses struct {
	mu    sync.Mutex
	isAll bool
	paths map[string]bool
	wake  chan struct{}
}

func newTriggerPasses() *triggerPasses {
	return &triggerPasses{isAll: true, paths: map[string]bool{}, wake: make(chan struct{}, 1)}
}

func (q *triggerPasses) requestAll() {
	q.mu.Lock()
	q.isAll = true
	q.mu.Unlock()
	signal(q.wake)
}

func (q *triggerPasses) requestPaths(paths []string) {
	if len(paths) == 0 {
		return
	}
	q.mu.Lock()
	for _, path := range paths {
		q.paths[path] = true
	}
	q.mu.Unlock()
	signal(q.wake)
}

func (q *triggerPasses) take() (isAll bool, paths []string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	isAll, paths = q.isAll, slices.Sorted(maps.Keys(q.paths))
	q.isAll, q.paths = false, map[string]bool{}
	return isAll, paths
}

func runTriggers(
	ctx context.Context, triggers TriggerRunner, v domain.Vault, passes *triggerPasses,
	changes <-chan struct{}, report func(error),
) {
	for {
		isAll, paths := passes.take()
		var err error
		switch {
		case isAll:
			err = triggers.ExecuteAll(ctx, v)
		case len(paths) > 0:
			err = triggers.Execute(ctx, v, paths)
		}
		if err != nil && ctx.Err() == nil {
			report(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-passes.wake:
		case <-changes:
			passes.requestAll()
		}
	}
}

func signal(wake chan<- struct{}) {
	select {
	case wake <- struct{}{}:
	default:
	}
}

func (s TriggerSource) start(ctx context.Context, v domain.Vault, passes *triggerPasses, report func(error)) func() {
	if s.Open == nil {
		return nil
	}
	triggers := s.Open()
	if triggers == nil {
		return nil
	}
	return func() {
		changes, stop := s.Watch()
		defer stop()
		runTriggers(ctx, triggers, v, passes, changes, report)
	}
}
