package plugin

import (
	"context"
	"errors"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var (
	errProcessNotRunning = errors.New("plugin process is not running")
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

// Supervisor monitors and controls the execution lifecycle of a single plugin process.
type Supervisor struct {
	mu             sync.Mutex
	state          ProcessState
	process        *os.Process
	lastActiveTime time.Time
	activeRefCount int
	permissions    []string
	validator      VaultPathValidator
	exitCh         chan struct{}
	onExit         func()
	activatingCh   chan struct{}
	activationErr  error
}

// NewSupervisor creates an idle supervisor for a plugin.
func NewSupervisor() *Supervisor {
	return &Supervisor{
		mu:             sync.Mutex{},
		state:          StateDormant,
		process:        nil,
		lastActiveTime: time.Time{},
		activeRefCount: 0,
		permissions:    nil,
		validator:      NewVaultPathValidator(),
		exitCh:         nil,
		onExit:         nil,
		activatingCh:   nil,
		activationErr:  nil,
	}
}

// SetOnExit sets a callback invoked when the plugin process terminates.
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
	if s.activatingCh != nil {
		close(s.activatingCh)
		s.activatingCh = nil
	}
}

// SetDormant transitions the supervisor state to StateDormant and clears activation state.
func (s *Supervisor) SetDormant(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = StateDormant
	s.process = nil
	s.activeRefCount = 0
	s.activationErr = err
	if s.activatingCh != nil {
		close(s.activatingCh)
		s.activatingCh = nil
	}
}

// SetPermissions stores the security permissions declared by the supervised plugin.
func (s *Supervisor) SetPermissions(permissions []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	norm := make([]string, len(permissions))
	for i, p := range permissions {
		if p == "network" {
			norm[i] = PermissionNetworkOutbound
		} else {
			norm[i] = p
		}
	}
	s.permissions = norm
}

// HasPermission reports whether the supervised plugin holds the given permission.
func (s *Supervisor) HasPermission(permission string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if permission == "network" {
		permission = PermissionNetworkOutbound
	}
	return slices.Contains(s.permissions, permission)
}

// CheckPermission returns an error if the supervised plugin lacks the requested permission.
func (s *Supervisor) CheckPermission(permission string) error {
	if !s.HasPermission(permission) {
		return port.ErrPermissionDenied
	}
	return nil
}

// ValidatePath checks that targetPath is within vaultRoot and the plugin has vault:read permission.
func (s *Supervisor) ValidatePath(vaultRoot, targetPath string) (string, error) {
	if err := s.CheckPermission(PermissionVaultRead); err != nil {
		return "", err
	}
	return s.validator.ValidatePath(vaultRoot, targetPath)
}

// Attach registers a newly started OS process under this supervisor.
func (s *Supervisor) Attach(p *os.Process, now time.Time) error {
	if p == nil {
		return errNilProcess
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.process = p
	s.state = StateActive
	s.lastActiveTime = now
	s.activationErr = nil
	if s.activatingCh != nil {
		close(s.activatingCh)
		s.activatingCh = nil
	}
	exitCh := make(chan struct{})
	s.exitCh = exitCh
	go func() {
		_, _ = p.Wait()
		close(exitCh)
		s.mu.Lock()
		if s.process == p {
			s.process = nil
			s.state = StateDormant
			s.activeRefCount = 0
			onExit := s.onExit
			s.mu.Unlock()
			if onExit != nil {
				onExit()
			}
			return
		}
		s.mu.Unlock()
	}()
	return nil
}

// Acquire increments the active consumer count, preventing idle eviction.
func (s *Supervisor) Acquire(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeRefCount++
	s.lastActiveTime = now
	s.state = StateActive
}

// Release decrements active consumers and marks the plugin idle when count reaches zero.
func (s *Supervisor) Release(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeRefCount > 0 {
		s.activeRefCount--
	}
	s.lastActiveTime = now
	if s.activeRefCount == 0 {
		s.state = StateIdle
	}
}

// IsIdle reports whether the process has been idle longer than the specified threshold.
func (s *Supervisor) IsIdle(threshold time.Duration, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != StateIdle || s.activeRefCount > 0 {
		return false
	}
	return now.Sub(s.lastActiveTime) >= threshold
}

// Teardown executes two-phase shutdown: graceful RPC call followed by hard kill fallback.
func (s *Supervisor) Teardown(
	ctx context.Context,
	gracePeriod time.Duration,
	rpcShutdown func(ctx context.Context) error,
) error {
	s.mu.Lock()
	if s.process == nil {
		if s.state == StateDormant {
			s.mu.Unlock()
			return errProcessNotRunning
		}
		s.state = StateDormant
		s.activeRefCount = 0
		if s.activatingCh != nil {
			close(s.activatingCh)
			s.activatingCh = nil
		}
		s.mu.Unlock()
		if rpcShutdown != nil {
			_ = rpcShutdown(ctx)
		}
		return nil
	}
	s.state = StateDeactivating
	if s.activatingCh != nil {
		close(s.activatingCh)
		s.activatingCh = nil
	}
	proc := s.process
	exitCh := s.exitCh
	s.mu.Unlock()

	_ = s.requestGracefulShutdown(ctx, rpcShutdown)

	select {
	case <-exitCh:
		s.markDormant(proc)
		return nil
	case <-time.After(gracePeriod):
		return s.forceKill(proc, exitCh)
	}
}

func (s *Supervisor) requestGracefulShutdown(ctx context.Context, rpcShutdown func(ctx context.Context) error) error {
	if rpcShutdown != nil {
		return rpcShutdown(ctx)
	}
	return nil
}

func (s *Supervisor) forceKill(proc *os.Process, exitCh chan struct{}) error {
	if proc != nil {
		_ = proc.Kill()
	}
	if exitCh != nil {
		<-exitCh
	}
	s.markDormant(proc)
	return nil
}

func (s *Supervisor) markDormant(proc *os.Process) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if proc == nil || s.process == proc {
		s.state = StateDormant
		s.process = nil
		s.activeRefCount = 0
		s.activationErr = nil
		if s.activatingCh != nil {
			close(s.activatingCh)
			s.activatingCh = nil
		}
	}
}

