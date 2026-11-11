package supervision

import (
	"context"
	"sync"
	"time"
)

// idleStopper stops the plugins that have been idle for a while.
type idleStopper interface {
	EvictIdle(ctx context.Context, threshold, gracePeriod time.Duration) []string
}

// Evictor stops the idle plugins on a timer, until Stop.
type Evictor struct {
	stopper idleStopper

	mu         sync.Mutex
	ticker     *time.Ticker
	stopSignal chan struct{}
	wg         sync.WaitGroup
	isStopped  bool
}

// NewEvictor returns the evictor that starts the timer: every thirty seconds it asks stopper to stop the plugins idle for
// longer than IdleThreshold.
func NewEvictor(stopper idleStopper) *Evictor {
	e := &Evictor{stopper: stopper, ticker: time.NewTicker(evictInterval), stopSignal: make(chan struct{})}
	e.wg.Go(func() {
		for {
			select {
			case <-e.stopSignal:
				return
			case <-e.ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), evictTimeout)
				e.stopper.EvictIdle(ctx, IdleThreshold, stopPluginGracePeriod)
				cancel()
			}
		}
	})
	return e
}

// Stop ends the timer that stops the idle plugins.
func (e *Evictor) Stop() {
	e.mu.Lock()
	if !e.isStopped {
		e.isStopped = true
		e.ticker.Stop()
		close(e.stopSignal)
	}
	e.mu.Unlock()
	e.wg.Wait()
}

// IsRunning reports whether the timer that stops the idle plugins runs.
func (e *Evictor) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return !e.isStopped
}
