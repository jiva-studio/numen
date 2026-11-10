package supervision

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

const (
	stopPluginGracePeriod = 2 * time.Second
	stopPluginTimeout     = 5 * time.Second
	evictInterval         = 30 * time.Second
	evictTimeout          = 5 * time.Second
)

// IdleThreshold is how long a plugin without a lease may stay idle before the host stops it.
const IdleThreshold = 180 * time.Second

// Stopper stops the processes of the plugins: one on request, the idle ones on a timer, and the ones
// that still run when the host shuts down.
type Stopper struct {
	sups      *Supervisors
	processes *Processes
	services  *Services
	clock     machineport.Clock
	report    func(error)

	evictMu        sync.Mutex
	evictTicker    *time.Ticker
	evictDone      chan struct{}
	evictWg        sync.WaitGroup
	isEvictStopped bool
}

// NewStopper returns the stopper of the processes the supervisors track, which stops the idle ones
// on a timer until StopEvicting.
func NewStopper(sups *Supervisors, processes *Processes, services *Services, clock machineport.Clock, report func(error)) *Stopper {
	s := &Stopper{
		sups: sups, processes: processes, services: services, clock: clock, report: report,
		evictTicker: time.NewTicker(evictInterval),
		evictDone:   make(chan struct{}),
	}
	s.startEvictLoop()
	return s
}

// StopPlugin stops the process of the plugin, if one runs, and forgets its endpoint.
func (s *Stopper) StopPlugin(ctx context.Context, pluginID string) error {
	target, ok := takeTarget(s.sups, pluginID)
	if !ok {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, stopPluginTimeout)
	defer cancel()
	if err := s.stopTarget(ctx, target, stopPluginGracePeriod); err != nil && !errors.Is(err, lifecycle.ErrProcessNotRunning) {
		return fmt.Errorf("stop plugin %s: %w", pluginID, err)
	}
	return nil
}

func (s *Stopper) stopTarget(ctx context.Context, target Target, gracePeriod time.Duration) error {
	rpcShutdown := createRPCShutdown(target.Endpoint, s.sups.Clients().Get(target.Endpoint), gracePeriod)
	err := target.Sup.Teardown(ctx, gracePeriod, rpcShutdown)
	s.sups.Clients().Remove(target.Endpoint.Token)
	s.processes.CloseStdin(target.ID, target.PID)
	s.services.ClearPlugin(target.ID)
	return err
}

// EvictIdle stops the plugins that hold no lease and have been idle for longer than threshold.
func (s *Stopper) EvictIdle(ctx context.Context, threshold, gracePeriod time.Duration) []string {
	return s.EvictTargets(ctx, takeIdle(s.sups, threshold, s.clock.Now()), gracePeriod)
}

// EvictTargets stops the targets and returns the ids of the ones it stopped. It reports a failure to stop one.
func (s *Stopper) EvictTargets(ctx context.Context, targets []Target, gracePeriod time.Duration) []string {
	evicted := make([]string, 0, len(targets))
	for _, target := range targets {
		if err := s.stopTarget(ctx, target, gracePeriod); err != nil && !errors.Is(err, lifecycle.ErrProcessNotRunning) {
			s.report(fmt.Errorf("evict plugin %s: %w", target.ID, err))
			continue
		}
		evicted = append(evicted, target.ID)
	}
	return evicted
}

func (s *Stopper) startEvictLoop() {
	s.evictWg.Go(func() {
		for {
			select {
			case <-s.evictDone:
				return
			case <-s.evictTicker.C:
				ctx, cancel := context.WithTimeout(context.Background(), evictTimeout)
				_ = s.EvictIdle(ctx, IdleThreshold, stopPluginGracePeriod)
				cancel()
			}
		}
	})
}

// StopEvicting ends the timer that stops the idle plugins.
func (s *Stopper) StopEvicting() {
	s.evictMu.Lock()
	if !s.isEvictStopped {
		s.isEvictStopped = true
		s.evictTicker.Stop()
		close(s.evictDone)
	}
	s.evictMu.Unlock()
	s.evictWg.Wait()
}

// IsEvicting reports whether the timer that stops the idle plugins runs.
func (s *Stopper) IsEvicting() bool {
	s.evictMu.Lock()
	defer s.evictMu.Unlock()
	return !s.isEvictStopped
}

// TakeRunning returns, as targets to stop, the plugins whose supervisor is not dormant, and forgets
// their endpoints.
func (s *Stopper) TakeRunning() []Target {
	return takeRunning(s.sups)
}

// StopTargets stops the targets, closes the pipes held and the endpoint of the host, and returns the
// first failure.
func (s *Stopper) StopTargets(ctx context.Context, targets []Target, gracePeriod time.Duration) error {
	var firstErr error
	for _, target := range targets {
		if err := s.stopTarget(ctx, target, gracePeriod); err != nil && firstErr == nil && !errors.Is(err, lifecycle.ErrProcessNotRunning) {
			firstErr = err
		}
	}
	if err := s.processes.CloseAllStdins(); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := s.services.CloseHostEndpoint(ctx); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}
