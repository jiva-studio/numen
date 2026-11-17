package windowsession

import (
	"context"
	"maps"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// triggerPasses is the passes of the triggers asked for and not run yet: one over the whole vault, or
// one over the paths that changed, and one over the notes whose frontmatter changed. A new one holds
// a pass over the whole vault. A note a frontmatter pass left undone is kept for the next pass, which
// it does not start.
type triggerPasses struct {
	mu          sync.Mutex
	isAll       bool
	paths       map[string]bool
	frontmatter map[string]bool
	wake        chan struct{}
}

func newTriggerPasses() *triggerPasses {
	return &triggerPasses{isAll: true, paths: map[string]bool{}, frontmatter: map[string]bool{}, wake: make(chan struct{}, 1)}
}

func (q *triggerPasses) requestAll() {
	q.mu.Lock()
	q.isAll = true
	q.mu.Unlock()
	signal(q.wake)
}

func (q *triggerPasses) requestPaths(paths []string) {
	q.request(func() map[string]bool { return q.paths }, paths)
}

func (q *triggerPasses) requestFrontmatter(paths []string) {
	q.request(func() map[string]bool { return q.frontmatter }, paths)
}

func (q *triggerPasses) keepFrontmatter(paths []string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, path := range paths {
		q.frontmatter[path] = true
	}
}

func (q *triggerPasses) request(getSet func() map[string]bool, paths []string) {
	if len(paths) == 0 {
		return
	}
	q.mu.Lock()
	into := getSet()
	for _, path := range paths {
		into[path] = true
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

func (q *triggerPasses) takeFrontmatter() []string {
	q.mu.Lock()
	defer q.mu.Unlock()
	paths := slices.Sorted(maps.Keys(q.frontmatter))
	q.frontmatter = map[string]bool{}
	return paths
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
		if frontmatter := passes.takeFrontmatter(); len(frontmatter) > 0 {
			retry, err := triggers.ExecuteFrontmatter(ctx, v, frontmatter)
			if err != nil && ctx.Err() == nil {
				report(err)
			}
			passes.keepFrontmatter(retry)
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
		if s.WatchFrontmatter != nil {
			defer s.WatchFrontmatter(v.ID, passes.requestFrontmatter)()
		}
		runTriggers(ctx, triggers, v, passes, changes, report)
	}
}
