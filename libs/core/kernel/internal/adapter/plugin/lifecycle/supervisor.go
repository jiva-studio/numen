package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

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

// Supervisor monitors and controls the execution lifecycle of a single plugin process. Failure keeps why
// the last start failed, and Permissions what the manifest of the plugin lets it do.
type Supervisor struct {
	Failure     process.StartFailure
	Permissions security.Permissions

	mu             sync.Mutex
	state          ProcessState
	process        *os.Process
	inProcess      bool
	teardownBegun  bool
	lastActiveTime time.Time
	activeRefCount int
	exitCh         chan struct{}
	onExit         func()
	activatingCh   chan struct{}
	activationErr  error
	rpcShutdown    func(ctx context.Context) error
	onError        func(error)
}

// New creates an idle supervisor for a plugin. onError receives the errors the supervisor meets outside
// a call, such as a failed wait for the process; it may be nil.
func New(onError func(error)) *Supervisor {
	return &Supervisor{Permissions: security.NewPermissions(), state: StateDormant, onError: onError}
}

// Usage is what a running plugin holds: when a lease was last taken or returned, the leases, whether
// it is served inside the host, and whether it is idle.
type Usage struct {
	LastActiveTime time.Time
	LeaseCount     int
	IsInProcess    bool
	IsIdle         bool
}

// GetUsage returns what the plugin holds, and false where it is not running.
func (s *Supervisor) GetUsage() (Usage, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !isRunning(s) {
		return Usage{}, false
	}
	return Usage{
		LastActiveTime: s.lastActiveTime,
		LeaseCount:     s.activeRefCount,
		IsInProcess:    s.inProcess,
		IsIdle:         s.state == StateIdle && s.activeRefCount == 0,
	}, true
}

// SetOnExit sets a callback invoked when the plugin process terminates, before the supervisor goes Dormant.
func (s *Supervisor) SetOnExit(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onExit = fn
}

// GetState returns the current lifecycle state of the supervised plugin.
func (s *Supervisor) GetState() ProcessState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// SetActivating transitions the supervisor state to StateActivating.
func (s *Supervisor) SetActivating() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = StateActivating
	s.activationErr = nil
	s.activatingCh = make(chan struct{})
	return s.activatingCh
}

// ActivatingChan returns a channel closed when the pending activation completes.
func (s *Supervisor) ActivatingChan() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activatingCh == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return s.activatingCh
}

// ActivationError returns the error from the most recent failed activation attempt.
func (s *Supervisor) ActivationError() error {
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
	s.process = nil
	s.inProcess = false
	s.teardownBegun = false
	s.activeRefCount = 0
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
	s.process = p
	s.teardownBegun = false
	s.state = StateActive
	s.lastActiveTime = now
	s.activationErr = nil
	closeActivationSignal(&s.activatingCh)
	exitCh := make(chan struct{})
	s.exitCh = exitCh

	go func() {
		_, waitErr := p.Wait()
		if waitErr != nil && !errors.Is(waitErr, os.ErrProcessDone) && s.onError != nil {
			s.onError(fmt.Errorf("wait for plugin process %d: %w", p.Pid, waitErr))
		}
		s.mu.Lock()
		var onExit func()
		if s.process == p {
			onExit = s.onExit
		}
		s.mu.Unlock()
		// The supervisor holds p until onExit returns, so no replacement starts before the host has
		// forgotten the state of the ended process.
		if onExit != nil {
			onExit()
		}
		s.mu.Lock()
		if s.process == p {
			s.process = nil
			s.inProcess = false
			s.teardownBegun = false
			s.state = StateDormant
			s.activeRefCount = 0
		}
		s.mu.Unlock()
		close(exitCh)
	}()
	return nil
}

// ActivateInProcess marks a plugin served inside the host as running and held by one consumer.
func (s *Supervisor) ActivateInProcess(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inProcess = true
	s.state = StateActive
	s.activeRefCount = 1
	s.lastActiveTime = now
	s.activationErr = nil
	closeActivationSignal(&s.activatingCh)
}

func isRunning(s *Supervisor) bool {
	if s.state != StateActive && s.state != StateIdle {
		return false
	}
	return s.process != nil || s.inProcess
}

// Acquire increments the active consumer count and reports whether a running plugin was held.
func (s *Supervisor) Acquire(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !isRunning(s) {
		return false
	}
	s.activeRefCount++
	s.lastActiveTime = now
	s.state = StateActive
	return true
}

// Release decrements active consumers and marks a running plugin idle when the count is zero.
func (s *Supervisor) Release(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !isRunning(s) {
		return
	}
	if s.activeRefCount > 0 {
		s.activeRefCount--
	}
	s.lastActiveTime = now
	if s.activeRefCount == 0 {
		s.state = StateIdle
	}
}

// BeginDeactivation moves a plugin that has been idle for the threshold to StateDeactivating and reports whether it did.
func (s *Supervisor) BeginDeactivation(threshold time.Duration, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != StateIdle || s.activeRefCount > 0 || now.Sub(s.lastActiveTime) < threshold {
		return false
	}
	s.state = StateDeactivating
	return true
}

// GetProcessID returns the pid of the attached process, or zero when none is attached.
func (s *Supervisor) GetProcessID() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.process == nil {
		return 0
	}
	return s.process.Pid
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
	rpcShutdown func(ctx context.Context) error,
) error {
	s.mu.Lock()
	if s.process == nil {
		s.mu.Unlock()
		return s.teardownNilProcess(ctx, rpcShutdown)
	}
	if s.state == StateDeactivating && s.teardownBegun {
		exitCh := s.exitCh
		s.mu.Unlock()
		return waitUntilDeactivated(ctx, exitCh)
	}
	s.state = StateDeactivating
	s.teardownBegun = true
	closeActivationSignal(&s.activatingCh)
	proc := s.process
	exitCh := s.exitCh
	if rpcShutdown == nil {
		rpcShutdown = s.rpcShutdown
	}
	s.mu.Unlock()

	// A plugin that cannot be asked to stop is killed without waiting out the grace period.
	if rpcShutdown == nil || rpcShutdown(ctx) == nil {
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

func (s *Supervisor) teardownNilProcess(ctx context.Context, rpcShutdown func(ctx context.Context) error) error {
	var shutdownErr error
	defer func() {
		if shutdownErr != nil && s.onError != nil {
			s.onError(fmt.Errorf("request plugin shutdown: %w", shutdownErr))
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
	s.activeRefCount = 0
	closeActivationSignal(&s.activatingCh)
	if rpcShutdown != nil {
		shutdownErr = rpcShutdown(ctx)
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
	if proc == nil || s.process == proc {
		s.state = StateDormant
		s.process = nil
		s.inProcess = false
		s.teardownBegun = false
		s.activeRefCount = 0
		s.activationErr = nil
		closeActivationSignal(&s.activatingCh)
	}
}

// SetRPCShutdown sets the RPC callback invoked during graceful eviction.
func (s *Supervisor) SetRPCShutdown(fn func(ctx context.Context) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rpcShutdown = fn
}

// GetExitSignal returns the channel the supervisor closes once it has reaped its current process, run
// its exit callback and gone Dormant.
func (s *Supervisor) GetExitSignal() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exitCh
}
