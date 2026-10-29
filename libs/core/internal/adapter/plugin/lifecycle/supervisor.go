package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/manifestfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/process"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/security"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

var (
	ErrProcessNotRunning = errors.New("plugin process is not running")
	errNilProcess        = errors.New("process handle is nil")
	// ErrMemoryExceeded indicates the plugin process exceeded its memory limit.
	ErrMemoryExceeded = errors.New("plugin exceeded memory limit")
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
	mu               sync.Mutex
	state            ProcessState
	process          *os.Process
	inProcess        bool
	teardownBegun    bool
	failures         int
	failure          error
	failed           bool
	failedAt         time.Time
	retryAt          time.Time
	activationPolicy ActivationPolicy
	lastActiveTime   time.Time
	activeRefCount   int
	permissions      []string
	validator        security.VaultPathValidator
	exitCh           chan struct{}
	onExit           func()
	activatingCh     chan struct{}
	activationErr    error
	cleanupErr       error

	memoryLimitBytes  uint64
	monitorInterval   time.Duration
	memoryGracePeriod time.Duration
	rpcShutdown       func(ctx context.Context) error
	onMemoryExceeded  func()
	monitorStopCh     chan struct{}
	rssReader         func(pid int) (uint64, error)
}

// New creates an idle supervisor for a plugin.
func New() *Supervisor {
	return &Supervisor{
		mu:                sync.Mutex{},
		state:             StateDormant,
		process:           nil,
		inProcess:         false,
		teardownBegun:     false,
		failures:          0,
		failure:           nil,
		failed:            false,
		failedAt:          time.Time{},
		retryAt:           time.Time{},
		activationPolicy:  DefaultActivationPolicy(),
		lastActiveTime:    time.Time{},
		activeRefCount:    0,
		permissions:       nil,
		validator:         security.NewVaultPathValidator(),
		exitCh:            nil,
		onExit:            nil,
		activatingCh:      nil,
		activationErr:     nil,
		memoryLimitBytes:  0,
		monitorInterval:   500 * time.Millisecond,
		memoryGracePeriod: 2 * time.Second,
		rpcShutdown:       nil,
		onMemoryExceeded:  nil,
		monitorStopCh:     nil,
		rssReader:         process.ReadRSS,
	}
}

// SetActivationPolicy sets the retry and backoff policy applied when activation fails or memory is exceeded.
func (s *Supervisor) SetActivationPolicy(policy ActivationPolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activationPolicy = policy
}

// GetActivationPolicy returns the configured activation policy.
func (s *Supervisor) GetActivationPolicy() ActivationPolicy {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activationPolicy
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

// CleanupError returns any error from the most recent cgroup cleanup.
func (s *Supervisor) CleanupError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cleanupErr
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
	s.stopMemoryMonitorLocked()
	s.state = StateDormant
	s.process = nil
	s.inProcess = false
	s.teardownBegun = false
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
		norm[i] = pluginport.NormalizePermission(p)
	}
	s.permissions = norm
}

// HasPermission reports whether the supervised plugin holds the given permission.
func (s *Supervisor) HasPermission(permission string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Contains(s.permissions, pluginport.NormalizePermission(permission))
}

// CheckPermission returns an error if the supervised plugin lacks the requested permission.
func (s *Supervisor) CheckPermission(permission string) error {
	if !s.HasPermission(permission) {
		return pluginport.ErrPermissionDenied
	}
	return nil
}

// ValidatePath checks that targetPath is within vaultRoot and the plugin has vault:read permission.
func (s *Supervisor) ValidatePath(vaultRoot, targetPath string) (string, error) {
	if err := s.CheckPermission(manifestfile.PermissionVaultRead); err != nil {
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
	s.teardownBegun = false
	s.state = StateActive
	s.lastActiveTime = now
	s.activationErr = nil
	if s.activatingCh != nil {
		close(s.activatingCh)
		s.activatingCh = nil
	}
	exitCh := make(chan struct{})
	s.exitCh = exitCh

	s.startMemoryMonitorLocked(p)

	go func() {
		_, _ = p.Wait()
		close(exitCh)
		cleanupErr := process.CleanupCgroup(p.Pid)
		s.mu.Lock()
		s.cleanupErr = cleanupErr
		s.stopMemoryMonitorLocked()
		if s.process == p {
			s.process = nil
			s.inProcess = false
			s.teardownBegun = false
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

// ActivateInProcess marks a plugin served inside the host as running and held by one consumer.
func (s *Supervisor) ActivateInProcess(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inProcess = true
	s.state = StateActive
	s.activeRefCount = 1
	s.lastActiveTime = now
	s.activationErr = nil
	if s.activatingCh != nil {
		close(s.activatingCh)
		s.activatingCh = nil
	}
}

func (s *Supervisor) isRunningLocked() bool {
	if s.state != StateActive && s.state != StateIdle {
		return false
	}
	return s.process != nil || s.inProcess
}

// Acquire increments the active consumer count and reports whether a running plugin was held.
func (s *Supervisor) Acquire(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.isRunningLocked() {
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
	if !s.isRunningLocked() {
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

// IsIdle reports whether the process has been idle longer than the specified threshold.
func (s *Supervisor) IsIdle(threshold time.Duration, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != StateIdle || s.activeRefCount > 0 {
		return false
	}
	return now.Sub(s.lastActiveTime) >= threshold
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
	s.stopMemoryMonitorLocked()
	if s.process == nil {
		s.mu.Unlock()
		return s.teardownNilProcess(ctx, rpcShutdown)
	}
	if s.state == StateDeactivating && s.teardownBegun {
		exitCh := s.exitCh
		s.mu.Unlock()
		return s.waitDeactivating(ctx, exitCh)
	}
	s.state = StateDeactivating
	s.teardownBegun = true
	if s.activatingCh != nil {
		close(s.activatingCh)
		s.activatingCh = nil
	}
	proc := s.process
	exitCh := s.exitCh
	if rpcShutdown == nil {
		rpcShutdown = s.rpcShutdown
	}
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

func (s *Supervisor) teardownNilProcess(ctx context.Context, rpcShutdown func(ctx context.Context) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == StateDormant {
		return ErrProcessNotRunning
	}
	s.state = StateDormant
	s.inProcess = false
	s.teardownBegun = false
	s.activeRefCount = 0
	if s.activatingCh != nil {
		close(s.activatingCh)
		s.activatingCh = nil
	}
	if rpcShutdown != nil {
		_ = rpcShutdown(ctx)
	}
	return nil
}

func (s *Supervisor) waitDeactivating(ctx context.Context, exitCh chan struct{}) error {
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
		s.stopMemoryMonitorLocked()
		s.state = StateDormant
		s.process = nil
		s.inProcess = false
		s.teardownBegun = false
		s.activeRefCount = 0
		s.activationErr = nil
		if s.activatingCh != nil {
			close(s.activatingCh)
			s.activatingCh = nil
		}
	}
}

func (s *Supervisor) stopMemoryMonitorLocked() {
	if s.monitorStopCh != nil {
		close(s.monitorStopCh)
		s.monitorStopCh = nil
	}
}

func (s *Supervisor) startMemoryMonitorLocked(p *os.Process) {
	if s.memoryLimitBytes == 0 || p == nil {
		return
	}
	s.stopMemoryMonitorLocked()
	stopCh := make(chan struct{})
	s.monitorStopCh = stopCh
	interval := s.monitorInterval
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	go s.runMemoryMonitor(p, stopCh, interval)
}

func (s *Supervisor) inspectMemoryExceeded(proc *os.Process) (func(context.Context) error, time.Duration, func(), bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.process != proc || (s.state != StateActive && s.state != StateIdle) || s.memoryLimitBytes == 0 {
		return nil, 0, nil, false
	}
	reader := s.rssReader
	if reader == nil {
		reader = process.ReadRSS
	}
	rss, err := reader(proc.Pid)
	if err != nil || rss <= s.memoryLimitBytes {
		return nil, 0, nil, false
	}
	return s.rpcShutdown, s.memoryGracePeriod, s.onMemoryExceeded, true
}

func (s *Supervisor) evictForMemory(rpc func(context.Context) error, gracePeriod time.Duration, onExceeded func()) {
	s.RecordActivationFailure(time.Now(), s.activationPolicy, fmt.Errorf("%w: %d bytes", ErrMemoryExceeded, s.memoryLimitBytes))
	if onExceeded != nil {
		onExceeded()
	}
	ctx, cancel := context.WithTimeout(context.Background(), gracePeriod+3*time.Second)
	defer cancel()
	_ = s.Teardown(ctx, gracePeriod, rpc)
}

func (s *Supervisor) runMemoryMonitor(proc *os.Process, stopCh <-chan struct{}, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			rpc, gracePeriod, onExceeded, shouldEvict := s.inspectMemoryExceeded(proc)
			if shouldEvict {
				s.evictForMemory(rpc, gracePeriod, onExceeded)
				return
			}
		}
	}
}

// SetMemoryLimit configures the resident memory threshold in bytes for eviction.
func (s *Supervisor) SetMemoryLimit(limit uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.memoryLimitBytes = limit
}

// GetMemoryLimit returns the configured resident memory threshold in bytes.
func (s *Supervisor) GetMemoryLimit() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.memoryLimitBytes
}

// SetMemoryMonitorInterval configures the polling interval for memory monitoring.
func (s *Supervisor) SetMemoryMonitorInterval(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.monitorInterval = interval
}

// SetMemoryGracePeriod configures the teardown grace period when memory is exceeded.
func (s *Supervisor) SetMemoryGracePeriod(period time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.memoryGracePeriod = period
}

// SetRPCShutdown sets the RPC callback invoked during graceful eviction.
func (s *Supervisor) SetRPCShutdown(fn func(ctx context.Context) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rpcShutdown = fn
}

// SetOnMemoryExceeded registers a callback invoked when resident memory exceeds limit.
func (s *Supervisor) SetOnMemoryExceeded(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onMemoryExceeded = fn
}

// SetRSSReader overrides the function that reads process resident memory.
func (s *Supervisor) SetRSSReader(fn func(pid int) (uint64, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rssReader = fn
}

// GetResidentMemory returns the resident set size in bytes of the supervised process.
func (s *Supervisor) GetResidentMemory() (uint64, error) {
	s.mu.Lock()
	proc := s.process
	reader := s.rssReader
	s.mu.Unlock()

	if proc == nil {
		return 0, ErrProcessNotRunning
	}
	if reader == nil {
		reader = process.ReadRSS
	}
	return reader(proc.Pid)
}

// IsMemoryExceeded reports whether resident memory exceeds the configured limit.
func (s *Supervisor) IsMemoryExceeded() bool {
	s.mu.Lock()
	limit := s.memoryLimitBytes
	proc := s.process
	reader := s.rssReader
	s.mu.Unlock()

	if limit == 0 || proc == nil {
		return false
	}
	if reader == nil {
		reader = process.ReadRSS
	}
	rss, err := reader(proc.Pid)
	if err != nil {
		return false
	}
	return rss > limit
}

// GetExitSignal returns the channel the supervisor closes once it has reaped its current process.
func (s *Supervisor) GetExitSignal() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exitCh
}

// BeginTeardown marks the supervisor as deactivating with its teardown under way.
func (s *Supervisor) BeginTeardown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = StateDeactivating
	s.teardownBegun = true
}

// GetOnMemoryExceeded returns what the supervisor calls once its process exceeds its memory limit.
func (s *Supervisor) GetOnMemoryExceeded() func() {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.onMemoryExceeded
}
