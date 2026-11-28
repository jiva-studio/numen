package synctest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/loops"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/watchchanges"
)

func (c *Computer) WatchDeps(waker ports.Waker) watchchanges.Deps {
	return watchchanges.Deps{Hints: c.Cloud, Host: c.Host, Links: c.State, Waker: waker, Clock: c.Clock, Jitter: NewJitter(1)}
}

// StartPush runs the sync loop of this computer with the given interval, fed by the stream of change hints as the
// plugin runs them, until the test ends.
func (c *Computer) StartPush(t *testing.T, interval time.Duration) {
	t.Helper()
	wakes := loops.NewWakes(16)
	ctx, cancel := context.WithCancel(t.Context())
	run := syncvault.NewRunSync(c.RunDeps(interval, wakes.WatchVaults()))
	watch := watchchanges.NewWatchChanges(c.WatchDeps(wakes))
	done := make(chan error, 2)
	go func() { done <- run.Execute(ctx) }()
	go func() { done <- watch.Execute(ctx) }()
	t.Cleanup(func() {
		cancel()
		for range 2 {
			if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
				t.Errorf("a loop ended with %v, want the error of its context or none", err)
			}
		}
	})
}
