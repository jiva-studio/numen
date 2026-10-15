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
	}
}

// GetState returns the current lifecycle state of the supervised plugin.
func (s *Supervisor) GetState() ProcessState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// SetPermissions stores the security permissions declared by the supervised plugin.
func (s *Supervisor) SetPermissions(permissions []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.permissions = slices.Clone(permissions)
}

// HasPermission reports whether the supervised plugin holds the given permission.
func (s *Supervisor) HasPermission(permission string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
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
		s.mu.Unlock()
		if rpcShutdown != nil {
			_ = rpcShutdown(ctx)
		}
		return nil
	}
	s.state = StateDeactivating
	proc := s.process
	s.mu.Unlock()

	exited := make(chan struct{})
	go s.waitForExit(proc, exited)

	_ = s.requestGracefulShutdown(ctx, rpcShutdown)

	select {
	case <-exited:
		s.markDormant()
		return nil
	case <-time.After(gracePeriod):
		return s.forceKill(proc, exited)
	}
}

func (s *Supervisor) requestGracefulShutdown(ctx context.Context, rpcShutdown func(ctx context.Context) error) error {
	if rpcShutdown != nil {
		return rpcShutdown(ctx)
	}
	return nil
}

func (s *Supervisor) waitForExit(proc *os.Process, exited chan struct{}) {
	if proc != nil {
		_, _ = proc.Wait()
	}
	close(exited)
}

func (s *Supervisor) forceKill(proc *os.Process, exited chan struct{}) error {
	if proc != nil {
		_ = proc.Kill()
	}
	<-exited
	s.markDormant()
	return nil
}

func (s *Supervisor) markDormant() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = StateDormant
	s.process = nil
	s.activeRefCount = 0
}
