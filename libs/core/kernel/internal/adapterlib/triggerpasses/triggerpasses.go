// Package triggerpasses runs the passes of the triggers of the plugins of a vault over the files
// that changed.
package triggerpasses

import (
	"context"
	"maps"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/trigger"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Passes is the passes of the triggers asked for and not run yet: one over the whole vault, or one
// over the paths that changed. A new one holds a pass over the whole vault.
type Passes struct {
	mu    sync.Mutex
	isAll bool
	paths map[string]bool
	wake  chan struct{}
}

// NewPasses makes the passes of a vault just opened.
func NewPasses() *Passes {
	return &Passes{isAll: true, paths: map[string]bool{}, wake: make(chan struct{}, 1)}
}

// RequestAll asks for a pass over the whole vault.
func (q *Passes) RequestAll() {
	q.mu.Lock()
	q.isAll = true
	q.mu.Unlock()
	signal(q.wake)
}

// RequestPaths asks for a pass over the paths.
func (q *Passes) RequestPaths(paths []string) {
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

// Take returns what was asked for since the last Take and clears it.
func (q *Passes) Take() (isAll bool, paths []string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	isAll, paths = q.isAll, slices.Sorted(maps.Keys(q.paths))
	q.isAll, q.paths = false, map[string]bool{}
	return isAll, paths
}

// Run runs the passes asked for, and waits for the next request or for the plugins to change, until
// ctx ends. A failed pass is reported unless ctx ended.
func Run(
	ctx context.Context, triggers *trigger.Triggers, v domain.Vault, passes *Passes,
	changes <-chan struct{}, report func(error),
) {
	for {
		isAll, paths := passes.Take()
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
			passes.RequestAll()
		}
	}
}

func signal(wake chan<- struct{}) {
	select {
	case wake <- struct{}{}:
	default:
	}
}

// Source is where the passes of the triggers come from. Open returns nil where the installation runs
// no plugins. Watch reports each time the plugin host applied settings or forgot the failure of a
// plugin, and the function it returns stops the reports.
type Source struct {
	Open  func() *trigger.Triggers
	Watch func() (<-chan struct{}, func())
}

// Start returns the function that runs the passes over the vault until ctx ends, or nil where the
// installation runs no triggers.
func (s Source) Start(ctx context.Context, v domain.Vault, passes *Passes, report func(error)) func() {
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
		Run(ctx, triggers, v, passes, changes, report)
	}
}
