package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var (
	errPluginNotFound  = errors.New("plugin not found")
	errEmptyEntrypoint = errors.New("plugin entrypoint is empty")
	errSafeModeActive  = errors.New("plugin host is running in safe mode")
	errHostClosed      = errors.New("plugin host is shut down")
)

// Launcher is a function that spawns a plugin process and provides its stdout stream.
type Launcher func(ctx context.Context, p DiscoveredPlugin, port int, token string) (*os.Process, io.ReadCloser, error)

// PersistHandler persists a changed plugin setting to host configuration.
type PersistHandler = port.PluginPersistHandler

// PluginStatus describes the enablement and runtime status of a plugin.
type PluginStatus struct {
	ID string `json:"id"`
	// Dir is the directory of a plugin that failed discovery; such an entry has no ID.
	Dir      string `json:"dir,omitempty"`
	Enabled  bool   `json:"enabled"`
	IsActive bool   `json:"isActive"`
	// Failed reports that the plugin failed to activate repeatedly and is not started again.
	Failed bool `json:"failed"`
	// FailureReason is the last activation error of a failed plugin.
	FailureReason string `json:"failureReason,omitempty"`
}

const settingEnabled = "enabled"

// Host manages the discovery, lifecycle, activation, and communication endpoints of plugins.
type Host struct {
	mu                sync.RWMutex
	target            string
	dirs              []string
	plugins           map[string]DiscoveredPlugin
	supervisors       map[string]*Supervisor
	endpoints         map[string]Endpoint
	launcher          Launcher
	safeMode          bool
	vaultPath         string
	bus               *EventBus
	settings          map[string]map[string]any
	persistHandler    PersistHandler
	evictTicker       *time.Ticker
	evictDone         chan struct{}
	evictWg           sync.WaitGroup
	isEvictStopped    bool
	isClosed          bool
	stdinsMu          sync.Mutex
	stdins            map[stdinKey]io.Closer
	processLimits     ProcessLimits
	activation        ActivationPolicy
	onPluginFailed    func(port.PluginFailure)
	onPluginRecovered func(pluginID string)
	errorHandler      ErrorHandler
	onDisk            map[string]bool
	discoverMu        sync.RWMutex
	failedPlugins     []port.PluginFailure
	problems          []DiscoveryProblem
	completer         port.Completer
	// consentAsks holds the plugins that asked for a completion the person has
	// not answered for, since the host started.
	consentAsks   map[string]bool
	secretLocksMu sync.Mutex
	secretLocks   map[string]*sync.Mutex
	keyStore      port.KeyStore
	// noKeyStoreOnce reports once that a recorded secret cannot be read on a
	// machine without a key store.
	noKeyStoreOnce sync.Once
	hostEndpoint   *HostEndpoint
	hostAddress    string
	onnxRuntime    RuntimeProvisioner
	builtIns       map[string]builtInPlugin
}

// NewHost creates a new plugin host for the specified target application and search paths.
func NewHost(target string, dirs ...string) *Host {
	h := &Host{
		mu:                sync.RWMutex{},
		target:            target,
		dirs:              dirs,
		plugins:           make(map[string]DiscoveredPlugin),
		supervisors:       make(map[string]*Supervisor),
		endpoints:         make(map[string]Endpoint),
		launcher:          nil,
		safeMode:          false,
		vaultPath:         "",
		bus:               nil,
		settings:          make(map[string]map[string]any),
		persistHandler:    nil,
		evictTicker:       time.NewTicker(30 * time.Second),
		evictDone:         make(chan struct{}),
		isClosed:          false,
		stdins:            make(map[stdinKey]io.Closer),
		processLimits:     DefaultProcessLimits(),
		activation:        DefaultActivationPolicy(),
		onPluginFailed:    nil,
		onPluginRecovered: nil,
		errorHandler:      nil,
		onDisk:            make(map[string]bool),
		discoverMu:        sync.RWMutex{},
		failedPlugins:     nil,
		problems:          nil,
		completer:         nil,
		consentAsks:       make(map[string]bool),
		keyStore:          nil,
		hostEndpoint:      nil,
		hostAddress:       hostEndpointAddress,
		onnxRuntime:       nil,
		builtIns:          make(map[string]builtInPlugin),
	}
	h.bus = NewEventBus(h, 50*time.Millisecond, 250*time.Millisecond)
	h.startEvictLoop()
	return h
}

// SetProcessLimits sets execution limits for launched plugin processes.
func (h *Host) SetProcessLimits(limits ProcessLimits) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.processLimits = limits
	for id, sup := range h.supervisors {
		if p, ok := h.plugins[id]; ok && p.Manifest.Lifecycle.MemoryLimitBytes > 0 {
			sup.SetMemoryLimit(p.Manifest.Lifecycle.MemoryLimitBytes)
		} else {
			sup.SetMemoryLimit(limits.MemoryLimitBytes)
		}
	}
}

// SetActivationPolicy sets the activation deadline and the restart backoff of plugins.
func (h *Host) SetActivationPolicy(policy ActivationPolicy) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.activation = policy
}

// SetErrorHandler registers a callback for problems the host meets outside a call, such as a failed shutdown.
func (h *Host) SetErrorHandler(handler ErrorHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errorHandler = handler
}

func (h *Host) reportError(err error) {
	h.mu.RLock()
	handler := h.errorHandler
	h.mu.RUnlock()
	if handler != nil {
		handler(err)
	}
}

// SetOnPluginFailed registers a callback invoked once when a plugin has failed to activate repeatedly,
// and again when a retry of it fails.
func (h *Host) SetOnPluginFailed(fn func(port.PluginFailure)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onPluginFailed = fn
}

// SetOnPluginRecovered registers a callback invoked with the id of a plugin the host stops holding off.
func (h *Host) SetOnPluginRecovered(fn func(pluginID string)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onPluginRecovered = fn
}

// SetLauncher overrides the process launcher for testing.
func (h *Host) SetLauncher(l Launcher) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.launcher = l
}

// SetSafeMode enables or disables safe recovery mode.
func (h *Host) SetSafeMode(enabled bool) {
	h.mu.Lock()
	h.safeMode = enabled
	var running []string
	if enabled {
		running = h.getRunningPluginIDsLocked()
	}
	h.mu.Unlock()

	for _, id := range running {
		if err := h.stopPlugin(context.Background(), id); err != nil {
			h.reportError(err)
		}
	}
}

func (h *Host) getRunningPluginIDsLocked() []string {
	ids := make([]string, 0, len(h.supervisors))
	for id, sup := range h.supervisors {
		if sup.GetState() != StateDormant {
			ids = append(ids, id)
		}
	}
	return ids
}

// stopPlugin shuts one plugin down, whatever state it is in.
func (h *Host) stopPlugin(ctx context.Context, pluginID string) error {
	h.mu.Lock()
	sup, ok := h.supervisors[pluginID]
	if !ok {
		h.mu.Unlock()
		return nil
	}
	target := pluginTeardownTarget{id: pluginID, pid: sup.GetProcessID(), sup: sup, endpoint: h.endpoints[pluginID]}
	delete(h.endpoints, pluginID)
	h.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, stopPluginTimeout)
	defer cancel()
	if err := h.teardownTarget(ctx, target, stopPluginGracePeriod); err != nil && !errors.Is(err, errProcessNotRunning) {
		return fmt.Errorf("stop plugin %s: %w", pluginID, err)
	}
	return nil
}

// teardownTarget stops the process of a target and closes its stdin pipe.
func (h *Host) teardownTarget(ctx context.Context, target pluginTeardownTarget, gracePeriod time.Duration) error {
	rpcShutdown := h.createRPCShutdown(target.endpoint, gracePeriod)
	err := target.sup.Teardown(ctx, gracePeriod, rpcShutdown)
	h.closeStdin(target.id, target.pid)
	return err
}

// IsSafeMode reports whether the host operates in safe recovery mode.
func (h *Host) IsSafeMode() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.safeMode
}

// SetVaultPath sets the active vault root path for path traversal checks.
func (h *Host) SetVaultPath(path string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.vaultPath = path
}

// GetVaultPath returns the currently configured active vault path.
func (h *Host) GetVaultPath() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.vaultPath
}

// CheckPluginPermission verifies that the specified plugin holds the required permission.
func (h *Host) CheckPluginPermission(pluginID, permission string) error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	sup, ok := h.supervisors[pluginID]
	if !ok {
		return errPluginNotFound
	}
	return sup.CheckPermission(permission)
}

// ValidatePluginPath verifies that targetPath is within the active vault and permitted for the plugin.
func (h *Host) ValidatePluginPath(pluginID, targetPath string) (string, error) {
	h.mu.RLock()
	sup, ok := h.supervisors[pluginID]
	vaultPath := h.vaultPath
	h.mu.RUnlock()

	if !ok {
		return "", errPluginNotFound
	}
	return sup.ValidatePath(vaultPath, targetPath)
}

// EmitEvent queues an event into the debounced and throttled event bus.
func (h *Host) EmitEvent(event string) {
	h.mu.RLock()
	bus := h.bus
	isSafe := h.safeMode
	h.mu.RUnlock()

	if isSafe || bus == nil {
		return
	}
	bus.Publish(event)
}

// GetEventBus returns the event bus attached to this host.
func (h *Host) GetEventBus() *EventBus {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.bus
}

// GetPlugins returns all discovered plugins registered with the host, in the order of their ids.
func (h *Host) GetPlugins() []DiscoveredPlugin {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.safeMode {
		return nil
	}

	return h.getPluginsInOrderLocked()
}

// getPluginsInOrderLocked returns the registered plugins in the order of their ids. Every lookup
// and listing of the host walks the plugins in this order, so the first plugin to match is the one
// with the lowest id.
func (h *Host) getPluginsInOrderLocked() []DiscoveredPlugin {
	result := make([]DiscoveredPlugin, 0, len(h.plugins))
	for _, p := range h.plugins {
		result = append(result, p)
	}
	slices.SortFunc(result, func(a, b DiscoveredPlugin) int { return strings.Compare(a.Manifest.ID, b.Manifest.ID) })
	return result
}

// GetPlugin retrieves a single plugin by its unique identifier.
func (h *Host) GetPlugin(id string) (DiscoveredPlugin, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.safeMode {
		return emptyDiscoveredPlugin(), false
	}

	p, ok := h.plugins[id]
	if !ok || !h.isPluginEnabledLocked(id) {
		return emptyDiscoveredPlugin(), false
	}
	return p, true
}

// ResolveCapability finds the enabled plugin with the lowest id among those declaring the given
// capability. Where none is enabled, the error is a *port.UnavailableError: PluginDisabled when a
// plugin declares it and it is switched off or the host is in safe mode, PluginNotInstalled when
// none declares it.
func (h *Host) ResolveCapability(capability string) (DiscoveredPlugin, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	reason := port.PluginNotInstalled
	for _, p := range h.getPluginsInOrderLocked() {
		if !p.Manifest.HasCapability(capability) {
			continue
		}
		if h.safeMode || !h.isPluginEnabledLocked(p.Manifest.ID) {
			reason = port.PluginDisabled
			continue
		}
		return p, nil
	}
	missing := &port.UnavailableError{Capability: capability, Reason: reason}
	if h.safeMode {
		missing.Err = errSafeModeActive
	}
	return emptyDiscoveredPlugin(), missing
}

// FindPluginByCapability finds the plugin ResolveCapability finds, and whether there is one.
func (h *Host) FindPluginByCapability(capability string) (DiscoveredPlugin, bool) {
	p, err := h.ResolveCapability(capability)
	return p, err == nil
}

// FindPluginByFileExtension finds the enabled plugin with the lowest id among those declaring
// support for the given file extension.
func (h *Host) FindPluginByFileExtension(ext string) (DiscoveredPlugin, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.safeMode {
		return emptyDiscoveredPlugin(), false
	}

	for _, p := range h.getPluginsInOrderLocked() {
		if !h.isPluginEnabledLocked(p.Manifest.ID) {
			continue
		}
		if p.Manifest.HasFileExtension(ext) {
			return p, true
		}
	}
	return emptyDiscoveredPlugin(), false
}

// TriggerEvent activates every enabled plugin whose activationEvents match the event and delivers the event to it.
func (h *Host) TriggerEvent(ctx context.Context, event string) ([]string, error) {
	h.mu.RLock()
	if h.safeMode {
		h.mu.RUnlock()
		return nil, nil
	}
	candidates := make([]string, 0)
	for _, p := range h.getPluginsInOrderLocked() {
		id := p.Manifest.ID
		if !h.isPluginEnabledLocked(id) {
			continue
		}
		if p.Manifest.CanActivateOn(event) {
			candidates = append(candidates, id)
		}
	}
	h.mu.RUnlock()

	activated := make([]string, 0, len(candidates))
	var errs []error
	for _, id := range candidates {
		endpoint, err := h.EnsureActive(ctx, id)
		if err != nil {
			errs = append(errs, fmt.Errorf("activate plugin %s on %s: %w", id, event, err))
			continue
		}
		deliverErr := deliverPluginEvent(ctx, endpoint, event)
		h.Release(id)
		activated = append(activated, id)
		if deliverErr != nil {
			errs = append(errs, fmt.Errorf("deliver %s to plugin %s: %w", event, id, deliverErr))
		}
	}
	return activated, errors.Join(errs...)
}

func (h *Host) prepareSupervisorLocked(pluginID string) (DiscoveredPlugin, *Supervisor, error) {
	if h.isClosed {
		return DiscoveredPlugin{}, nil, errHostClosed
	}
	if h.safeMode {
		return DiscoveredPlugin{}, nil, &port.UnavailableError{PluginID: pluginID, Reason: port.PluginDisabled, Err: errSafeModeActive}
	}

	p, ok := h.plugins[pluginID]
	if !ok {
		return DiscoveredPlugin{}, nil, &port.UnavailableError{
			PluginID: pluginID, Reason: port.PluginNotInstalled, Err: errPluginNotFound,
		}
	}

	sup, hasSup := h.supervisors[pluginID]
	if !hasSup {
		sup = NewSupervisor()
		sup.SetPermissions(p.Manifest.Permissions)
		id := pluginID
		sup.SetOnExit(func() { h.dropEndpointOfStoppedPlugin(id, sup) })
		h.supervisors[pluginID] = sup
	}
	memLimit := h.processLimits.MemoryLimitBytes
	if p.Manifest.Lifecycle.MemoryLimitBytes > 0 {
		memLimit = p.Manifest.Lifecycle.MemoryLimitBytes
	}
	sup.SetMemoryLimit(memLimit)

	if !h.isPluginEnabledLocked(pluginID) {
		return DiscoveredPlugin{}, nil, &port.UnavailableError{PluginID: pluginID, Reason: port.PluginDisabled}
	}

	return p, sup, nil
}

func (h *Host) awaitActivation(ctx context.Context, pluginID string, sup *Supervisor) (Endpoint, bool, error) {
	waitCh := sup.ActivatingChan()
	select {
	case <-ctx.Done():
		return Endpoint{Host: "", Port: 0, Token: ""}, false, ctx.Err()
	case <-waitCh:
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if endpoint, ok := h.acquireRunningLocked(pluginID, sup); ok {
		return endpoint, true, nil
	}
	if err := sup.ActivationError(); err != nil {
		return Endpoint{Host: "", Port: 0, Token: ""}, false, err
	}
	return Endpoint{Host: "", Port: 0, Token: ""}, false, nil
}

// dropEndpointOfStoppedPlugin forgets the endpoint of an exited process unless a replacement is already running.
func (h *Host) dropEndpointOfStoppedPlugin(pluginID string, sup *Supervisor) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if state := sup.GetState(); state == StateDormant || state == StateActivating {
		delete(h.endpoints, pluginID)
	}
}

// acquireRunningLocked holds the plugin for a consumer when it is running with a usable endpoint.
func (h *Host) acquireRunningLocked(pluginID string, sup *Supervisor) (Endpoint, bool) {
	endpoint := h.endpoints[pluginID]
	if isEmptyEndpoint(endpoint) || !sup.Acquire(time.Now()) {
		return emptyEndpoint(), false
	}
	return endpoint, true
}

// failActivation returns a plugin whose activation failed to Dormant and counts the failure in the same step,
// so no other caller can start it again before the backoff applies.
func (h *Host) failActivation(ctx context.Context, pluginID string, sup *Supervisor, cause error) {
	h.mu.Lock()
	if sup.GetState() != StateActivating {
		h.mu.Unlock()
		return
	}
	sup.SetDormant(cause)
	failedNow := false
	if ctx.Err() == nil {
		wasFailed, _ := sup.GetActivationFailure()
		failedNow = sup.RecordActivationFailure(time.Now(), h.activation, cause) && !wasFailed
	}
	if failedNow {
		h.failedPlugins = append(h.failedPlugins, h.newPluginFailureLocked(pluginID, sup, cause))
	}
	h.mu.Unlock()
}

// newPluginFailureLocked is the failure a person is told of for a plugin the host has given up.
func (h *Host) newPluginFailureLocked(pluginID string, sup *Supervisor, cause error) port.PluginFailure {
	return port.PluginFailure{
		PluginID: pluginID,
		Name:     h.plugins[pluginID].Manifest.Name,
		Reason:   describeFailure(cause),
		At:       sup.GetFailedAt(),
	}
}

// deliverPluginFailures hands the plugins that just failed for good to the failure callback, outside every lock.
func (h *Host) deliverPluginFailures() {
	h.mu.Lock()
	failures := h.failedPlugins
	h.failedPlugins = nil
	notify := h.onPluginFailed
	h.mu.Unlock()

	if notify == nil {
		return
	}
	for _, failure := range failures {
		notify(failure)
	}
}

// settlePluginInState waits out or clears a state other than Dormant and reports whether EnsureActive is finished.
func (h *Host) settlePluginInState(
	ctx context.Context,
	pluginID string,
	sup *Supervisor,
	state ProcessState,
) (Endpoint, bool, error) {
	switch state {
	case StateDeactivating:
		err := sup.WaitWhileDeactivating(ctx)
		return emptyEndpoint(), err != nil, err
	case StateActivating:
		endpoint, ready, err := h.awaitActivation(ctx, pluginID, sup)
		return endpoint, ready || err != nil, err
	default:
		if err := sup.Teardown(ctx, 0, nil); err != nil && !errors.Is(err, errProcessNotRunning) {
			return emptyEndpoint(), true, fmt.Errorf("retire unreachable plugin %s: %w", pluginID, err)
		}
		return emptyEndpoint(), false, nil
	}
}

// EnsureActive ensures the plugin process is running and returns its endpoint.
func (h *Host) EnsureActive(ctx context.Context, pluginID string) (Endpoint, error) {
	endpoint, err := h.ensureActiveDuringDiscovery(ctx, pluginID)
	h.deliverPluginFailures()
	return endpoint, err
}

// ensureActiveDuringDiscovery activates a plugin while holding off a discovery that could retire it.
func (h *Host) ensureActiveDuringDiscovery(ctx context.Context, pluginID string) (Endpoint, error) {
	h.discoverMu.RLock()
	defer h.discoverMu.RUnlock()

	for {
		h.mu.Lock()
		p, sup, err := h.prepareSupervisorLocked(pluginID)
		if err != nil {
			h.mu.Unlock()
			return Endpoint{Host: "", Port: 0, Token: ""}, err
		}

		if endpoint, ok := h.acquireRunningLocked(pluginID, sup); ok {
			h.mu.Unlock()
			return endpoint, nil
		}

		if state := sup.GetState(); state != StateDormant {
			h.mu.Unlock()
			endpoint, done, err := h.settlePluginInState(ctx, pluginID, sup, state)
			if done {
				return endpoint, err
			}
			continue
		}

		if err := h.beginActivationLocked(pluginID, sup); err != nil {
			h.mu.Unlock()
			return emptyEndpoint(), err
		}
		h.mu.Unlock()

		return h.runActivation(ctx, p, sup)
	}
}

// beginActivationLocked moves a dormant plugin to Activating unless it is backing off or has failed.
func (h *Host) beginActivationLocked(pluginID string, sup *Supervisor) error {
	if err := sup.CheckActivationAllowed(time.Now()); err != nil {
		return fmt.Errorf("activate plugin %s: %w", pluginID, err)
	}
	sup.SetActivating()
	return nil
}

func (h *Host) runActivation(ctx context.Context, p DiscoveredPlugin, sup *Supervisor) (Endpoint, error) {
	endpoint, err := h.activatePlugin(ctx, p, sup)
	if err != nil {
		h.failActivation(ctx, p.Manifest.ID, sup, err)
		return endpoint, err
	}
	sup.RecordActivationSuccess()

	// The plugin also receives its settings on the next change, so a failed push leaves it running.
	_ = h.NotifyPluginConfig(ctx, p.Manifest.ID)
	return endpoint, nil
}

func (h *Host) cleanupAbortedProcess(proc *os.Process, stdout io.Closer, pluginID string) {
	_ = proc.Kill()
	_, _ = proc.Wait()
	_ = stdout.Close()
	h.closeStdin(pluginID, proc.Pid)
}

// closeStdin closes and forgets the stdin pipe of one process generation of a plugin.
func (h *Host) closeStdin(pluginID string, pid int) {
	key := stdinKey{pluginID: pluginID, pid: pid}
	h.stdinsMu.Lock()
	defer h.stdinsMu.Unlock()
	if c, ok := h.stdins[key]; ok {
		_ = c.Close()
		delete(h.stdins, key)
	}
}

// waitForReady reads the handshake of a started process within the activation deadline.
func (h *Host) waitForReady(ctx context.Context, stdout io.Reader) (int, error) {
	h.mu.RLock()
	timeout := h.activation.Timeout
	h.mu.RUnlock()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return WaitForHandshake(ctx, stdout)
}

//nolint:contextcheck // background supervisor outlives request
func (h *Host) startPluginProcess(ctx context.Context, p DiscoveredPlugin, sup *Supervisor) (Endpoint, error) {
	token, err := GenerateAuthToken()
	if err != nil {
		return Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("generate token: %w", err)
	}

	h.mu.RLock()
	launcher := h.launcher
	h.mu.RUnlock()
	if launcher == nil {
		launcher = h.defaultLauncher
	}

	proc, stdout, err := launcher(ctx, p, 0, token)
	if err != nil {
		return Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("launch plugin %s: %w", p.Manifest.ID, err)
	}

	actualPort, err := h.waitForReady(ctx, stdout)
	if err != nil {
		h.cleanupAbortedProcess(proc, stdout, p.Manifest.ID)
		return Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("handshake plugin %s: %w", p.Manifest.ID, err)
	}

	go func() {
		defer stdout.Close()
		_, _ = io.Copy(io.Discard, stdout)
	}()

	endpoint := Endpoint{
		Host:  "127.0.0.1",
		Port:  actualPort,
		Token: token,
	}
	if err := checkPluginProtocol(ctx, p.Manifest, endpoint); err != nil {
		h.cleanupAbortedProcess(proc, stdout, p.Manifest.ID)
		return emptyEndpoint(), err
	}

	now := time.Now()
	h.mu.Lock()
	if sup.GetState() != StateActivating {
		h.mu.Unlock()
		h.cleanupAbortedProcess(proc, stdout, p.Manifest.ID)
		return Endpoint{Host: "", Port: 0, Token: ""}, errActivationAborted
	}

	if err := sup.Attach(proc, now); err != nil {
		h.mu.Unlock()
		h.cleanupAbortedProcess(proc, stdout, p.Manifest.ID)
		return Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("attach plugin %s: %w", p.Manifest.ID, err)
	}
	sup.SetOnExit(func() {
		h.dropEndpointOfStoppedPlugin(p.Manifest.ID, sup)

		h.closeStdin(p.Manifest.ID, proc.Pid)
	})
	sup.Acquire(now)

	h.endpoints[p.Manifest.ID] = endpoint
	sup.SetRPCShutdown(h.createRPCShutdown(endpoint, 2*time.Second))
	sup.FinishActivating()
	h.mu.Unlock()

	return endpoint, nil
}

func (h *Host) defaultLauncher(
	ctx context.Context,
	p DiscoveredPlugin,
	port int,
	token string,
) (*os.Process, io.ReadCloser, error) {
	if p.Manifest.Backend.Entrypoint == "" {
		return nil, nil, errEmptyEntrypoint
	}

	entrypoint := p.Manifest.Backend.Entrypoint
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(entrypoint), ".exe") {
		entrypoint += ".exe"
	}

	binPath, err := validateExecutablePath(p.Dir, entrypoint)
	if err != nil {
		return nil, nil, fmt.Errorf("validate plugin binary: %w", err)
	}

	h.mu.RLock()
	vaultPath := h.vaultPath
	limits := h.processLimits
	h.mu.RUnlock()

	if p.Manifest.Lifecycle.MemoryLimitBytes > 0 {
		limits.MemoryLimitBytes = p.Manifest.Lifecycle.MemoryLimitBytes
	}

	workDir, err := ValidateWorkingDirectory(p.Dir, p.Dir, vaultPath)
	if err != nil {
		return nil, nil, fmt.Errorf("validate plugin working directory: %w", err)
	}

	cmd := exec.CommandContext(context.WithoutCancel(ctx), binPath, getPluginArgs(port, token)...)
	configureSysProcAttr(cmd)
	cmd.Dir = workDir

	pluginVars, err := h.getLaunchVars(ctx, p.Manifest.ID, p.Dir, p.Manifest.Runtimes, port, token)
	if err != nil {
		return nil, nil, err
	}
	cmd.Env = sanitizeEnvironment(os.Environ(), pluginVars, p.Manifest.Runtimes)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("pipe stdin: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, nil, fmt.Errorf("pipe stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, nil, fmt.Errorf("start %s: %w", binPath, err)
	}

	if err := applyProcessLimits(cmd.Process.Pid, limits); err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, nil, fmt.Errorf("apply process limits: %w", err)
	}

	h.stdinsMu.Lock()
	if h.stdins == nil {
		h.stdins = make(map[stdinKey]io.Closer)
	}
	h.stdins[stdinKey{pluginID: p.Manifest.ID, pid: cmd.Process.Pid}] = stdin
	h.stdinsMu.Unlock()

	return cmd.Process, stdout, nil
}

// Acquire increments the active consumer reference count for a plugin.
func (h *Host) Acquire(pluginID string) {
	h.mu.RLock()
	sup, ok := h.supervisors[pluginID]
	h.mu.RUnlock()
	if ok {
		sup.Acquire(time.Now())
	}
}

// Release decrements the active consumer reference count for a plugin.
func (h *Host) Release(pluginID string) {
	h.mu.RLock()
	sup, ok := h.supervisors[pluginID]
	h.mu.RUnlock()
	if ok {
		sup.Release(time.Now())
	}
}

type stdinKey struct {
	pluginID string
	pid      int
}

const (
	stopPluginGracePeriod = 2 * time.Second
	stopPluginTimeout     = 5 * time.Second
)

type pluginTeardownTarget struct {
	id       string
	pid      int
	sup      *Supervisor
	endpoint Endpoint
}

// EvictIdle stops idle plugins whose idle duration exceeds their configured timeout.
func (h *Host) EvictIdle(ctx context.Context, defaultThreshold, gracePeriod time.Duration) []string {
	h.mu.Lock()
	now := time.Now()
	var targets []pluginTeardownTarget

	for id, sup := range h.supervisors {
		threshold := defaultThreshold
		if p, ok := h.plugins[id]; ok && p.Manifest.Lifecycle.IdleTimeoutSeconds > 0 {
			threshold = time.Duration(p.Manifest.Lifecycle.IdleTimeoutSeconds) * time.Second
		}

		pid := sup.GetProcessID()
		if sup.BeginDeactivation(threshold, now) {
			targets = append(targets, pluginTeardownTarget{
				id:       id,
				pid:      pid,
				sup:      sup,
				endpoint: h.endpoints[id],
			})
			delete(h.endpoints, id)
		}
	}
	h.mu.Unlock()

	return h.evictTargets(ctx, targets, gracePeriod)
}

// EvictMemoryExceeded stops plugins whose resident memory exceeds their configured limit.
func (h *Host) EvictMemoryExceeded(ctx context.Context, gracePeriod time.Duration) []string {
	h.mu.Lock()
	var targets []pluginTeardownTarget

	for id, sup := range h.supervisors {
		if sup.IsMemoryExceeded() {
			targets = append(targets, pluginTeardownTarget{
				id:       id,
				pid:      sup.GetProcessID(),
				sup:      sup,
				endpoint: h.endpoints[id],
			})
			delete(h.endpoints, id)
		}
	}
	h.mu.Unlock()

	return h.evictTargets(ctx, targets, gracePeriod)
}

// evictTargets stops the targets and returns the ids that stopped; a failed stop is reported to the error handler.
func (h *Host) evictTargets(ctx context.Context, targets []pluginTeardownTarget, gracePeriod time.Duration) []string {
	evicted := make([]string, 0, len(targets))
	for _, target := range targets {
		if err := h.teardownTarget(ctx, target, gracePeriod); err != nil && !errors.Is(err, errProcessNotRunning) {
			h.reportError(fmt.Errorf("evict plugin %s: %w", target.id, err))
			continue
		}
		evicted = append(evicted, target.id)
	}
	return evicted
}

func (h *Host) startEvictLoop() {
	h.evictWg.Add(1)
	go func() {
		defer h.evictWg.Done()
		for {
			select {
			case <-h.evictDone:
				return
			case <-h.evictTicker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_ = h.EvictIdle(ctx, 3*time.Minute, 2*time.Second)
				_ = h.EvictMemoryExceeded(ctx, 2*time.Second)
				cancel()
			}
		}
	}()
}

func (h *Host) stopEvictLoop() {
	h.mu.Lock()
	if !h.isEvictStopped {
		h.isEvictStopped = true
		if h.evictTicker != nil {
			h.evictTicker.Stop()
		}
		close(h.evictDone)
	}
	h.mu.Unlock()
	h.evictWg.Wait()
}

// TeardownAll shuts down all active plugin processes.
func (h *Host) TeardownAll(ctx context.Context, gracePeriod time.Duration) error {
	h.stopEvictLoop()

	var firstErr error
	if h.bus != nil {
		if err := h.bus.Close(ctx); err != nil {
			firstErr = err
		}
	}

	h.mu.Lock()
	h.isClosed = true
	var targets []pluginTeardownTarget
	for id, sup := range h.supervisors {
		if sup.GetState() != StateDormant {
			targets = append(targets, pluginTeardownTarget{
				id:       id,
				pid:      sup.GetProcessID(),
				sup:      sup,
				endpoint: h.endpoints[id],
			})
			delete(h.endpoints, id)
		}
	}
	h.mu.Unlock()

	for _, target := range targets {
		if err := h.teardownTarget(ctx, target, gracePeriod); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	h.stdinsMu.Lock()
	for key, c := range h.stdins {
		_ = c.Close()
		delete(h.stdins, key)
	}
	h.stdinsMu.Unlock()
	if err := h.closeHostEndpoint(ctx); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func (h *Host) createRPCShutdown(endpoint Endpoint, gracePeriod time.Duration) func(context.Context) error {
	if endpoint.Host == "" && endpoint.Port == 0 && endpoint.Transport == nil {
		return nil
	}
	return func(ctx context.Context) error {
		timeout := gracePeriod
		if timeout <= 0 {
			timeout = 2 * time.Second
		}
		rpcCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		client := numenv1connect.NewPluginLifecycleServiceClient(
			EndpointClient(endpoint),
			endpoint.BaseURL(),
			connect.WithInterceptors(authInterceptor(endpoint.Token)),
		)
		req := connect.NewRequest(&v1.ShutdownRequest{
			TimeoutSeconds: int32(gracePeriod.Seconds()),
		})
		_, err := client.Shutdown(rpcCtx, req)
		return err
	}
}

// SetPersistHandler registers a callback that writes plugin settings changes into core configuration.
func (h *Host) SetPersistHandler(handler PersistHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.persistHandler = handler
}

// SetSettings loads persisted plugin configurations into host memory.
func (h *Host) SetSettings(all map[string]map[string]any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for pluginID, cfg := range all {
		if _, ok := h.settings[pluginID]; !ok {
			h.settings[pluginID] = make(map[string]any)
		}
		for k, v := range cfg {
			h.settings[pluginID][k] = v
		}
	}
}

// IsPluginEnabled reports whether the specified plugin is enabled.
func (h *Host) IsPluginEnabled(pluginID string) bool {
	return h.isPluginEnabled(pluginID)
}

func (h *Host) isPluginEnabled(pluginID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.isPluginEnabledLocked(pluginID)
}

func (h *Host) isPluginEnabledLocked(pluginID string) bool {
	settings := h.getPluginSettingsLocked(pluginID)
	if val, ok := settings["enabled"].(bool); ok {
		return val
	}
	return true
}

// SetPluginEnabled sets whether the plugin is enabled, persists the choice and stops the plugin when it is disabled.
func (h *Host) SetPluginEnabled(ctx context.Context, pluginID string, enabled bool) error {
	return h.SetPluginSetting(ctx, pluginID, settingEnabled, enabled)
}

// applyEnabledChange stops a plugin that was disabled and clears the failures of one that was enabled.
func (h *Host) applyEnabledChange(ctx context.Context, pluginID string, value any) error {
	enabled, ok := value.(bool)
	if !ok {
		return nil
	}
	if !enabled {
		return h.stopPlugin(ctx, pluginID)
	}
	h.mu.RLock()
	sup := h.supervisors[pluginID]
	h.mu.RUnlock()
	if sup != nil {
		h.clearFailure(pluginID, sup)
	}
	return nil
}

// GetPluginStatus returns the status of the specified plugin.
func (h *Host) GetPluginStatus(pluginID string) (PluginStatus, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	p, ok := h.plugins[pluginID]
	if !ok {
		return PluginStatus{}, false
	}
	return h.getPluginStatusLocked(p.Manifest.ID), true
}

func (h *Host) getPluginStatusLocked(pluginID string) PluginStatus {
	status := PluginStatus{
		ID:            pluginID,
		Enabled:       h.isPluginEnabledLocked(pluginID),
		IsActive:      h.isPluginActiveLocked(pluginID),
		Failed:        false,
		FailureReason: "",
		Dir:           "",
	}
	if sup, ok := h.supervisors[pluginID]; ok {
		if failed, cause := sup.GetActivationFailure(); failed {
			status.Failed = true
			status.FailureReason = cause.Error()
		}
	}
	return status
}

// GetPluginStatuses returns statuses for all registered plugins, in the order of their ids.
func (h *Host) GetPluginStatuses() []PluginStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	statuses := make([]PluginStatus, 0, len(h.plugins)+len(h.problems))
	for _, p := range h.getPluginsInOrderLocked() {
		statuses = append(statuses, h.getPluginStatusLocked(p.Manifest.ID))
	}
	for _, problem := range h.problems {
		statuses = append(statuses, PluginStatus{
			ID:            "",
			Dir:           problem.Dir,
			Enabled:       false,
			IsActive:      false,
			Failed:        true,
			FailureReason: problem.Err.Error(),
		})
	}
	return statuses
}

func (h *Host) isPluginActiveLocked(pluginID string) bool {
	if sup, ok := h.supervisors[pluginID]; ok {
		return sup.GetState() == StateActive || sup.GetState() == StateIdle
	}
	return false
}

// GetPluginSettings returns resolved settings for a plugin with defaults overlaid by stored values.
func (h *Host) GetPluginSettings(pluginID string) map[string]any {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.getPluginSettingsLocked(pluginID)
}

func (h *Host) getPluginSettingsLocked(pluginID string) map[string]any {
	result := make(map[string]any)
	if p, ok := h.plugins[pluginID]; ok {
		for _, s := range p.Manifest.Settings {
			if s.Default != nil {
				result[s.Key] = s.Default
			}
		}
	}
	if stored, ok := h.settings[pluginID]; ok {
		for k, v := range stored {
			result[k] = v
		}
	}
	delete(result, port.SettingStoredSecrets)
	delete(result, port.SettingAIConsent)
	if p, ok := h.plugins[pluginID]; ok {
		for _, name := range p.Manifest.Secrets {
			delete(result, name)
		}
	}
	return result
}

// GetPluginConfigJSON returns serialized JSON configuration for the specified plugin.
func (h *Host) GetPluginConfigJSON(pluginID string) (string, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.getPluginConfigJSONLocked(pluginID)
}

func (h *Host) getPluginConfigJSONLocked(pluginID string) (string, error) {
	cfg := h.getPluginSettingsLocked(pluginID)
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal plugin %s settings: %w", pluginID, err)
	}
	return string(data), nil
}

// SetPluginSetting persists one setting, then writes it to memory and notifies the plugin.
// The record of the stored secrets is the host's own and is refused.
func (h *Host) SetPluginSetting(ctx context.Context, pluginID, key string, value any) error {
	if key == port.SettingStoredSecrets {
		return fmt.Errorf("plugin %s: %w: %s", pluginID, port.ErrReservedSettingKey, key)
	}
	h.mu.RLock()
	persist := h.persistHandler
	h.mu.RUnlock()
	if persist != nil {
		if err := persist(pluginID, key, value); err != nil {
			return fmt.Errorf("persist plugin setting: %w", err)
		}
	}
	return h.UpdatePluginSetting(ctx, pluginID, key, value)
}

// UpdatePluginSetting updates the setting in memory and notifies the plugin without re-persisting.
// The record of the stored secrets is written by the host alone, so a write of it
// here changes nothing.
func (h *Host) UpdatePluginSetting(ctx context.Context, pluginID, key string, value any) error {
	if key == port.SettingStoredSecrets {
		return nil
	}
	h.mu.Lock()
	if _, ok := h.settings[pluginID]; !ok {
		h.settings[pluginID] = make(map[string]any)
	}
	h.settings[pluginID][key] = value
	h.mu.Unlock()

	if key == settingEnabled {
		if err := h.applyEnabledChange(ctx, pluginID, value); err != nil {
			return err
		}
	}
	return h.NotifyPluginConfig(ctx, pluginID)
}

// PaletteCommand declares a command palette toggle or option action for a plugin setting.
type PaletteCommand struct {
	ID       string `json:"id"`
	PluginID string `json:"pluginId"`
	Key      string `json:"key"`
	Title    string `json:"title"`
	Type     string `json:"type"`
}

// GetPaletteCommands returns command definitions for all registered plugin settings declaring palette: true.
func (h *Host) GetPaletteCommands() []PaletteCommand {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var commands []PaletteCommand
	for _, p := range h.getPluginsInOrderLocked() {
		for _, s := range p.Manifest.Settings {
			if s.Palette {
				commands = append(commands, PaletteCommand{
					ID:       fmt.Sprintf("plugin-setting:%s:%s", p.Manifest.ID, s.Key),
					PluginID: p.Manifest.ID,
					Key:      s.Key,
					Title:    fmt.Sprintf("%s: %s", p.Manifest.Name, s.Title),
					Type:     s.Type,
				})
			}
		}
	}
	return commands
}

// TogglePluginSetting inverts the boolean value of a setting with palette flag.
func (h *Host) TogglePluginSetting(ctx context.Context, pluginID, key string) (bool, error) {
	h.mu.RLock()
	p, ok := h.plugins[pluginID]
	if !ok {
		h.mu.RUnlock()
		return false, errPluginNotFound
	}
	var isBool bool
	for _, s := range p.Manifest.Settings {
		if s.Key == key && s.Type == "boolean" {
			isBool = true
			break
		}
	}
	if !isBool {
		h.mu.RUnlock()
		return false, errors.New("setting is not a boolean")
	}
	current := h.getPluginSettingsLocked(pluginID)
	curVal, _ := current[key].(bool)
	newVal := !curVal
	h.mu.RUnlock()

	err := h.SetPluginSetting(ctx, pluginID, key, newVal)
	return newVal, err
}

// NotifyPluginConfig sends the resolved configuration to a running plugin process.
func (h *Host) NotifyPluginConfig(ctx context.Context, pluginID string) error {
	defer h.lockSecrets(pluginID)()
	return h.notifyPluginConfigLocked(ctx, pluginID)
}

// notifyPluginConfigLocked is NotifyPluginConfig for a caller holding lockSecrets(pluginID).
func (h *Host) notifyPluginConfigLocked(ctx context.Context, pluginID string) error {
	h.mu.RLock()
	endpoint, hasEndpoint := h.endpoints[pluginID]
	sup, hasSup := h.supervisors[pluginID]
	if !hasEndpoint || !hasSup {
		h.mu.RUnlock()
		return nil
	}
	state := sup.GetState()
	if state != StateActive && state != StateIdle {
		h.mu.RUnlock()
		return nil
	}
	cfgJSON, err := h.getPluginConfigJSONLocked(pluginID)
	if err != nil {
		h.mu.RUnlock()
		return err
	}
	rawSettings := h.getPluginSettingsLocked(pluginID)
	values := formatPluginSettings(rawSettings)
	secretNames := slices.Clone(h.plugins[pluginID].Manifest.Secrets)
	stored := h.getStoredSecretsLocked(pluginID)
	store := h.keyStore
	h.mu.RUnlock()

	secrets, failures := readPluginSecrets(ctx, store, pluginID, secretNames, stored)
	secretsErr := h.getSecretsError(failures)
	for name, key := range secrets {
		values[name] = key
	}

	httpClient := EndpointClient(endpoint)
	client := numenv1connect.NewPluginLifecycleServiceClient(
		httpClient,
		endpoint.BaseURL(),
		connect.WithInterceptors(authInterceptor(endpoint.Token)),
	)
	req := connect.NewRequest(&v1.ConfigureRequest{
		PluginId:          pluginID,
		ConfigurationJson: cfgJSON,
		Values:            values,
	})
	configCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err = client.Configure(configCtx, req)
	if err != nil {
		var connectErr *connect.Error
		if errors.As(err, &connectErr) && connectErr.Code() == connect.CodeUnimplemented && len(secretNames) == 0 {
			return secretsErr
		}
		return errors.Join(secretsErr, fmt.Errorf("configure plugin %s: %w", pluginID, err))
	}
	return secretsErr
}

// NotifyAllPluginConfigs sends resolved configurations to all running plugins.
func (h *Host) NotifyAllPluginConfigs(ctx context.Context) error {
	h.mu.RLock()
	seen := make(map[string]struct{}, len(h.plugins))
	for id := range h.plugins {
		seen[id] = struct{}{}
	}
	for id := range h.supervisors {
		seen[id] = struct{}{}
	}
	h.mu.RUnlock()

	var firstErr error
	for id := range seen {
		if err := h.NotifyPluginConfig(ctx, id); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// GetPluginInfos returns specifications of all registered plugins and their settings schemas, in
// the order of their ids.
func (h *Host) GetPluginInfos() []port.PluginInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]port.PluginInfo, 0, len(h.plugins))
	for _, p := range h.getPluginsInOrderLocked() {
		isActive := h.isPluginActiveLocked(p.Manifest.ID)

		settings := make([]port.PluginSetting, 0, len(p.Manifest.Settings))
		for _, s := range p.Manifest.Settings {
			defJSON := ""
			if s.Default != nil {
				if b, err := json.Marshal(s.Default); err == nil {
					defJSON = string(b)
				}
			}
			minVal := s.Min
			maxVal := s.Max
			stepVal := s.Step
			settings = append(settings, port.PluginSetting{
				Key:         s.Key,
				Type:        s.Type,
				Title:       s.Title,
				Description: s.Description,
				DefaultJSON: defJSON,
				Palette:     s.Palette,
				Options:     s.Options,
				Min:         &minVal,
				Max:         &maxVal,
				Step:        &stepVal,
			})
		}

		result = append(result, port.PluginInfo{
			ID:            p.Manifest.ID,
			Name:          p.Manifest.Name,
			Version:       p.Manifest.Version,
			Description:   p.Manifest.Description,
			Capabilities:  p.Manifest.Capabilities,
			Settings:      settings,
			IsActive:      isActive,
			Permissions:   p.Manifest.Permissions,
			Secrets:       p.Manifest.Secrets,
			StoredSecrets: h.getStoredSecretsLocked(p.Manifest.ID),
			Consent:       h.getCompletionConsentLocked(p.Manifest),
			Failure:       h.getFailureLocked(p.Manifest.ID),
		})
	}
	return result
}

// GetWirePluginInfos returns wire specifications of all registered plugins and their settings
// schemas, in the order of their ids.
func (h *Host) GetWirePluginInfos() []*v1.PluginInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]*v1.PluginInfo, 0, len(h.plugins))
	for _, p := range h.getPluginsInOrderLocked() {
		isActive := h.isPluginActiveLocked(p.Manifest.ID)

		settings := make([]*v1.PluginSetting, 0, len(p.Manifest.Settings))
		for _, s := range p.Manifest.Settings {
			defJSON := ""
			if s.Default != nil {
				if b, err := json.Marshal(s.Default); err == nil {
					defJSON = string(b)
				}
			}
			minVal := s.Min
			maxVal := s.Max
			stepVal := s.Step
			settings = append(settings, &v1.PluginSetting{
				Key:         s.Key,
				Type:        s.Type,
				Title:       s.Title,
				Description: s.Description,
				DefaultJson: defJSON,
				Palette:     s.Palette,
				Options:     s.Options,
				Min:         &minVal,
				Max:         &maxVal,
				Step:        &stepVal,
			})
		}

		result = append(result, &v1.PluginInfo{
			Id:           p.Manifest.ID,
			Name:         p.Manifest.Name,
			Version:      p.Manifest.Version,
			Description:  p.Manifest.Description,
			Capabilities: p.Manifest.Capabilities,
			Settings:     settings,
			IsActive:     isActive,
			Permissions:  p.Manifest.Permissions,
			Secrets:      p.Manifest.Secrets,
		})
	}
	return result
}

// PluginServiceServer implements numenv1connect.PluginServiceHandler.
type PluginServiceServer struct {
	host *Host
}

// NewPluginServiceServer creates a server exposing plugin schemas over Connect-RPC.
func NewPluginServiceServer(host *Host) *PluginServiceServer {
	return &PluginServiceServer{host: host}
}

// GetPlugins returns discovered plugins and declarative settings schemas.
func (s *PluginServiceServer) GetPlugins(
	_ context.Context,
	_ *connect.Request[v1.GetPluginsRequest],
) (*connect.Response[v1.GetPluginsResponse], error) {
	return connect.NewResponse(&v1.GetPluginsResponse{
		Plugins: s.host.GetWirePluginInfos(),
	}), nil
}

func formatPluginSettings(rawSettings map[string]any) map[string]string {
	values := make(map[string]string, len(rawSettings))
	for k, v := range rawSettings {
		if val, ok := v.(string); ok {
			values[k] = val
			continue
		}
		if b, err := json.Marshal(v); err == nil {
			values[k] = string(b)
		} else {
			values[k] = fmt.Sprint(v)
		}
	}
	return values
}

func emptyEndpoint() Endpoint {
	return Endpoint{Host: "", Port: 0, Token: ""}
}

func isEmptyEndpoint(e Endpoint) bool {
	return e.Host == "" && e.Port == 0 && e.Transport == nil
}

// getFailureLocked is why the host holds a plugin off, and nil while it does not.
func (h *Host) getFailureLocked(pluginID string) *port.PluginFailure {
	sup, ok := h.supervisors[pluginID]
	if !ok {
		return nil
	}
	failed, cause := sup.GetActivationFailure()
	if !failed {
		return nil
	}
	failure := h.newPluginFailureLocked(pluginID, sup, cause)
	return &failure
}

// clearFailure forgets the failed activations of a plugin and tells the recovery callback where the plugin
// had been held off.
func (h *Host) clearFailure(pluginID string, sup *Supervisor) {
	wasFailed, _ := sup.GetActivationFailure()
	sup.ResetActivationFailures()
	if !wasFailed {
		return
	}
	h.mu.RLock()
	notify := h.onPluginRecovered
	h.mu.RUnlock()
	if notify != nil {
		notify(pluginID)
	}
}

// RetryPlugin forgets the failures of a plugin and starts it once. The failure is set where the plugin fails
// again, and the host holds it off at once.
func (h *Host) RetryPlugin(ctx context.Context, pluginID string) (*port.PluginFailure, error) {
	h.mu.RLock()
	_, known := h.plugins[pluginID]
	sup := h.supervisors[pluginID]
	h.mu.RUnlock()
	if !known {
		return nil, &port.UnavailableError{PluginID: pluginID, Reason: port.PluginNotInstalled, Err: errPluginNotFound}
	}
	if sup != nil {
		h.clearFailure(pluginID, sup)
	}

	_, err := h.EnsureActive(ctx, pluginID)
	if err == nil {
		h.Release(pluginID)
		return nil, nil
	}
	var unavailable *port.UnavailableError
	if errors.As(err, &unavailable) || ctx.Err() != nil {
		return nil, err
	}
	failure := h.holdOff(pluginID, err)
	h.deliverPluginFailures()
	return failure, nil
}

// holdOff records a failed activation as the last one the plugin is allowed, and returns the failure.
func (h *Host) holdOff(pluginID string, cause error) *port.PluginFailure {
	h.mu.Lock()
	defer h.mu.Unlock()
	sup, ok := h.supervisors[pluginID]
	if !ok {
		return nil
	}
	policy := h.activation
	policy.MaxFailures = 1
	if !sup.RecordActivationFailure(time.Now(), policy, cause) {
		return nil
	}
	failure := h.newPluginFailureLocked(pluginID, sup, cause)
	h.failedPlugins = append(h.failedPlugins, failure)
	return &failure
}
