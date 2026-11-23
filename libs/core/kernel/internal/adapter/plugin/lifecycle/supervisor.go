package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/process"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/security"
)

var (
	ErrProcessNotRunning = errors.New("plugin process is not running")
	errNilProcess        = errors.New("process handle is nil")
)

// ProcessState represents the lifecycle phase of a managed plugin process.
type ProcessState int

const (
	// StateDormant indicates the plugin is discovered on disk but not executing.
	StateDormant ProcessState = iota
	// StateActivating indicates the plugin is in the process of starting up.
	StateActivating
	// StateActive indicates the plugin is executing and handling requests.
	StateActive
	// StateIdle indicates the plugin is running but has no active consumers.
	StateIdle
	// StateDeactivating indicates the plugin is executing its graceful shutdown sequence.
	StateDeactivating
)

// LeaseKind says what a lease holds a plugin for: a use of the plugin, a question the core asks the
// plugin for itself, or work the plugin does on its own. A question lease holds the process as a use lease
// does, but neither marks the plugin used nor moves its idle countdown. An own lease counts as a use but
// does not cancel a release of the process.
type LeaseKind int

const (
	// UseLease holds the plugin for a use of it.
	UseLease LeaseKind = iota
	// QuestionLease holds the plugin for a question the core asks it for itself.
	QuestionLease
	// OwnLease holds the plugin for work it started on its own, such as an open task.
	OwnLease
)

// Lease is one hold on a plugin: its kind and its holder.
type Lease struct {
	Kind   LeaseKind
	Holder plugin.Holder
}

func (l Lease) isUse() bool {
	return l.Kind != QuestionLease
}

// Supervisor monitors and controls the execution lifecycle of a single plugin process. Failure keeps why
// the last start failed, and Permissions what the manifest of the plugin lets it do.
type Supervisor struct {
	Failure     process.StartFailure
	Permissions security.Permissions
	Child       process.Child

	mu            sync.Mutex
	state         ProcessState
	inProcess     bool
	teardownBegun bool
	leases        plugin.Leases
	exitCh        chan struct{}
	handleExit        func()
	activatingCh  chan struct{}
	activationErr error
	callShutdown   func(ctx context.Context) error
	handleError       func(error)
}

// New creates an idle supervisor for a plugin. onError receives the errors the supervisor meets outside
// a call, such as a failed wait for the process; it may be nil.
func New(handleError func(error)) *Supervisor {
	return &Supervisor{Permissions: security.NewPermissions(), state: StateDormant, handleError: handleError}
}

// Usage is what a running plugin holds: when it started or a use lease was last taken or returned, the
// leases, whether it is served inside the host, whether it is idle, and whether a use lease was taken
// since it started.
type Usage struct {
	LastActiveTime time.Time
	LeaseCount     int
	IsInProcess    bool
	IsIdle         bool
	IsUsed         bool
	StartedBy      plugin.Holder
	Holders        []plugin.Holder
}

// GetUsage returns what the plugin holds, and false where it is not running.
func (s *Supervisor) GetUsage() (Usage, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !isRunning(s) {
		return Usage{}, false
	}
	leases := s.leases.GetState()
	return Usage{
		LastActiveTime: leases.LastUseTime,
		LeaseCount:     len(leases.Holders),
		IsInProcess:    s.inProcess,
		IsIdle:         s.state == StateIdle && len(leases.Holders) == 0,
		IsUsed:         leases.IsUsed,
		StartedBy:      leases.StartedBy,
		Holders:        leases.Holders,
	}, true
}

// SetOnExit sets a callback invoked when the plugin process terminates, before the supervisor goes Dormant.
func (s *Supervisor) SetOnExit(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handleExit = fn
}

// GetState returns the current lifecycle state of the supervised plugin.
func (s *Supervisor) GetState() ProcessState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// SetActivating transitions the supervisor state to StateActivating, started by the starter.
func (s *Supervisor) SetActivating(starter plugin.Holder) <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.leases.Begin(starter)
	s.state = StateActivating
	s.activationErr = nil
	s.activatingCh = make(chan struct{})
	return s.activatingCh
}

// GetActivatingChan returns a channel closed when the pending activation completes.
func (s *Supervisor) GetActivatingChan() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activatingCh == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return s.activatingCh
}

// GetActivationError returns the error from the most recent failed activation attempt.
func (s *Supervisor) GetActivationError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activationErr
}

// FinishActivating transitions the supervisor state to StateActive and unblocks waiters.
func (s *Supervisor) FinishActivating() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = StateActive
	s.activationErr = nil
	closeActivationSignal(&s.activatingCh)
}

// SetDormant transitions the supervisor state to StateDormant and clears activation state.
func (s *Supervisor) SetDormant(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = StateDormant
	s.Child.SetProcess(nil)
	s.inProcess = false
	s.teardownBegun = false
	s.leases.Clear()
	s.activationErr = err
	closeActivationSignal(&s.activatingCh)
}

func closeActivationSignal(ch *chan struct{}) {
	if *ch != nil {
		close(*ch)
		*ch = nil
	}
}

// Attach registers a newly started OS process under this supervisor.
func (s *Supervisor) Attach(ctx context.Context, p *os.Process, now time.Time) error {
	if p == nil {
		return errNilProcess
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Child.SetProcess(p)
	s.teardownBegun = false
	s.state = StateActive
	s.leases.Start(now)
	s.activationErr = nil
	closeActivationSignal(&s.activatingCh)
	exitCh := make(chan struct{})
	s.exitCh = exitCh

	go func() {
		_, waitErr := p.Wait()
		if waitErr != nil && !errors.Is(waitErr, os.ErrProcessDone) && s.handleError != nil {
			s.handleError(fmt.Errorf("wait for plugin process %d: %w", p.Pid, waitErr))
		}
		s.mu.Lock()
		var handleExit func()
		if s.Child.GetProcess() == p {
			handleExit = s.handleExit
		}
		s.mu.Unlock()
		// The supervisor holds p until onExit returns, so no replacement starts before the host has
		// forgotten the state of the ended process.
		if handleExit != nil {
			handleExit()
		}
		s.mu.Lock()
		if s.Child.GetProcess() == p {
			s.Child.SetProcess(nil)
			s.inProcess = false
			s.teardownBegun = false
			s.state = StateDormant
			s.leases.Clear()
		}
		s.mu.Unlock()
		close(exitCh)
	}()
	return nil
}

// ActivateInProcess marks a plugin served inside the host as running and held by the lease.
func (s *Supervisor) ActivateInProcess(lease Lease, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inProcess = true
	s.state = StateActive
	s.leases.StartHeld(lease.Holder, lease.isUse(), now)
	s.activationErr = nil
	closeActivationSignal(&s.activatingCh)
}

func isRunning(s *Supervisor) bool {
	if s.state != StateActive && s.state != StateIdle {
		return false
	}
	return s.Child.GetProcess() != nil || s.inProcess
}

// Acquire takes the lease on a running plugin and reports whether it did. A use lease marks the plugin
// used and restarts its idle countdown.
func (s *Supervisor) Acquire(lease Lease, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !isRunning(s) {
		return false
	}
	s.leases.Add(lease.Holder, lease.isUse(), now)
	s.state = StateActive
	return true
}

// Release returns the lease and marks a running plugin idle when it holds none. A use lease restarts the
// idle countdown.
func (s *Supervisor) Release(lease Lease, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !isRunning(s) {
		return
	}
	if s.leases.Remove(lease.Holder, lease.isUse(), now) {
		s.state = StateIdle
	}
}

// BeginDeactivation moves an idle plugin to StateDeactivating and reports whether it did: a plugin used
// since it started once it has been idle for the threshold, and a plugin never used at once.
func (s *Supervisor) BeginDeactivation(threshold time.Duration, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != StateIdle || !s.leases.IsDueToStop(threshold, now) {
		return false
	}
	s.state = StateDeactivating
	return true
}

// WaitWhileDeactivating blocks until the supervisor leaves StateDeactivating or ctx ends.
func (s *Supervisor) WaitWhileDeactivating(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for s.GetState() == StateDeactivating {
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for deactivation: %w", ctx.Err())
		case <-ticker.C:
		}
	}
	return nil
}

// Teardown executes two-phase shutdown: graceful RPC call followed by hard kill fallback.
func (s *Supervisor) Teardown(
	ctx context.Context,
	gracePeriod time.Duration,
	callShutdown func(ctx context.Context) error,
) error {
	s.mu.Lock()
	if s.Child.GetProcess() == nil {
		s.mu.Unlock()
		return s.teardownNilProcess(ctx, callShutdown)
	}
	if s.state == StateDeactivating && s.teardownBegun {
		exitCh := s.exitCh
		s.mu.Unlock()
		return waitUntilDeactivated(ctx, exitCh)
	}
	s.state = StateDeactivating
	s.teardownBegun = true
	closeActivationSignal(&s.activatingCh)
	proc := s.Child.GetProcess()
	exitCh := s.exitCh
	if callShutdown == nil {
		callShutdown = s.callShutdown
	}
	s.mu.Unlock()

	// A plugin that cannot be asked to stop is killed without waiting out the grace period.
	if callShutdown == nil || callShutdown(ctx) == nil {
		select {
		case <-exitCh:
			s.markDormant(proc)
			return nil
		case <-time.After(gracePeriod):
		}
	}
	if err := kill(proc, exitCh); err != nil {
		return err
	}
	s.markDormant(proc)
	return nil
}

func (s *Supervisor) teardownNilProcess(ctx context.Context, callShutdown func(ctx context.Context) error) error {
	var shutdownErr error
	defer func() {
		if shutdownErr != nil && s.handleError != nil {
			s.handleError(fmt.Errorf("request plugin shutdown: %w", shutdownErr))
		}
	}()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == StateDormant {
		return ErrProcessNotRunning
	}
	s.state = StateDormant
	s.inProcess = false
	s.teardownBegun = false
	s.leases.Clear()
	closeActivationSignal(&s.activatingCh)
	if callShutdown != nil {
		shutdownErr = callShutdown(ctx)
	}
	return nil
}

func waitUntilDeactivated(ctx context.Context, exitCh chan struct{}) error {
	if exitCh == nil {
		return nil
	}
	select {
	case <-exitCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func kill(proc *os.Process, exitCh chan struct{}) error {
	if proc != nil {
		if err := proc.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("kill plugin process %d: %w", proc.Pid, err)
		}
	}
	if exitCh != nil {
		<-exitCh
	}
	return nil
}

func (s *Supervisor) markDormant(proc *os.Process) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if proc == nil || s.Child.GetProcess() == proc {
		s.state = StateDormant
		s.Child.SetProcess(nil)
		s.inProcess = false
		s.teardownBegun = false
		s.leases.Clear()
		s.activationErr = nil
		closeActivationSignal(&s.activatingCh)
	}
}

// SetRPCShutdown sets the RPC callback invoked during graceful eviction.
func (s *Supervisor) SetRPCShutdown(fn func(ctx context.Context) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callShutdown = fn
}

// GetExitSignal returns the channel the supervisor closes once it has reaped its current process, run
// its exit callback and gone Dormant.
func (s *Supervisor) GetExitSignal() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exitCh
}
