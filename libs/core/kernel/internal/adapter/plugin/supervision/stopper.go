package supervision

import (
	"context"
	"errors"
	"fmt"
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

// stdinHolder closes the pipes the starts of the processes hold open.
type stdinHolder interface {
	CloseStdin(pluginID string, pid int)
	CloseAllStdins() error
}

// serviceEndpoint forgets what the host endpoint holds for a plugin, and closes the endpoint.
type serviceEndpoint interface {
	ClearPlugin(pluginID string)
	CloseHostEndpoint(ctx context.Context) error
}

// Stopper stops the processes of the plugins: one on request, the idle ones when asked, and the ones
// that still run when the host shuts down.
type Stopper struct {
	sups      *Supervisors
	processes stdinHolder
	services  serviceEndpoint
	clock     machineport.Clock
	report    func(error)
}

// NewStopper returns the stopper of the processes the supervisors track.
func NewStopper(sups *Supervisors, processes stdinHolder, services serviceEndpoint, clock machineport.Clock, report func(error)) *Stopper {
	return &Stopper{sups: sups, processes: processes, services: services, clock: clock, report: report}
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
	s.sups.NotifyChange()
	return err
}

// EvictIdle stops the plugins that hold no lease and have been idle for longer than threshold.
func (s *Stopper) EvictIdle(ctx context.Context, threshold, gracePeriod time.Duration) []string {
	return s.EvictTargets(ctx, takeIdleTargets(s.sups, threshold, s.clock.Now()), gracePeriod)
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

// TakeRunning returns, as targets to stop, the plugins whose supervisor is not dormant, and forgets
// their endpoints.
func (s *Stopper) TakeRunning() []Target {
	return takeRunningTargets(s.sups)
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
