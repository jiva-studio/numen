package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/eventbus"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/manifestfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/process"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/security"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	completionport "github.com/jiva-studio/numen/libs/core/usecase/completion/completioncontract"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

var (
	errPluginNotFound = errors.New("plugin not found")
	ErrSafeModeActive = errors.New("plugin host is running in safe mode")
	errHostClosed     = errors.New("plugin host is shut down")
)

// Launcher is a function that spawns a plugin process and provides its stdout stream.
type Launcher func(ctx context.Context, p manifestfile.DiscoveredPlugin, port int, token string) (*os.Process, io.ReadCloser, error)

// PersistHandler persists a changed plugin setting to host configuration.
type PersistHandler = pluginport.PluginPersistHandler

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
	mu               sync.RWMutex
	target           string
	dirs             []string
	plugins          map[string]manifestfile.DiscoveredPlugin
	supervisors      map[string]*lifecycle.Supervisor
	endpoints        map[string]transport.Endpoint
	launcher         Launcher
	safeMode         bool
	vaultPath        string
	bus              *eventbus.Bus
	settings         map[string]map[string]any
	persistHandler   PersistHandler
	evictTicker      *time.Ticker
	evictDone        chan struct{}
	evictWg          sync.WaitGroup
	isEvictStopped   bool
	isClosed         bool
	stdinsMu         sync.Mutex
	stdins           map[stdinKey]io.Closer
	processLimits    process.Limits
	activation       lifecycle.ActivationPolicy
	onPluginFailed   func(pluginport.PluginFailure)
	onFailureCleared func(pluginID string)
	errorHandler     eventbus.ErrorHandler
	onDisk           map[string]bool
	discoverMu       sync.RWMutex
	failedPlugins    []pluginport.PluginFailure
	problems         []manifestfile.DiscoveryProblem
	completer        completionport.Completer
	vault            pluginport.PluginVault
	texts            pluginport.PluginTexts
	// consentAsks holds the plugins that requested a completion while the consent
	// prompt was unanswered, since the host started.
	consentAsks   map[string]bool
	secretLocksMu sync.Mutex
	secretLocks   map[string]*sync.Mutex
	keyStore      pluginport.KeyStore
	// noKeyStoreOnce reports once that a recorded secret cannot be read on a
	// machine without a key store.
	noKeyStoreOnce sync.Once
	hostEndpoint   *hostendpoint.Endpoint
	hostAddress    string
	dependencies   pluginport.DependencyCatalog
	work           pluginport.PluginWork
	builtIns       map[string]builtInPlugin
	contentTypes   contentTypeTable
	typeCache      contentTypeCache
}

// NewHost creates a new plugin host for the specified target application and search paths.
func NewHost(target string, dirs ...string) *Host {
	h := &Host{
		mu:               sync.RWMutex{},
		target:           target,
		dirs:             dirs,
		plugins:          make(map[string]manifestfile.DiscoveredPlugin),
		supervisors:      make(map[string]*lifecycle.Supervisor),
		endpoints:        make(map[string]transport.Endpoint),
		launcher:         nil,
		safeMode:         false,
		vaultPath:        "",
		bus:              nil,
		settings:         make(map[string]map[string]any),
		persistHandler:   nil,
		evictTicker:      time.NewTicker(30 * time.Second),
		evictDone:        make(chan struct{}),
		isClosed:         false,
		stdins:           make(map[stdinKey]io.Closer),
		processLimits:    process.DefaultLimits(),
		activation:       lifecycle.DefaultActivationPolicy(),
		onPluginFailed:   nil,
		onFailureCleared: nil,
		errorHandler:     nil,
		onDisk:           make(map[string]bool),
		discoverMu:       sync.RWMutex{},
		failedPlugins:    nil,
		problems:         nil,
		completer:        nil,
		consentAsks:      make(map[string]bool),
		keyStore:         nil,
		hostEndpoint:     nil,
		hostAddress:      hostendpoint.Address,
		dependencies:     nil,
		work:             nil,
		builtIns:         make(map[string]builtInPlugin),
		contentTypes:     nil,
		typeCache:        contentTypeCache{},
	}
	h.bus = eventbus.New(h, 50*time.Millisecond, 250*time.Millisecond)
	h.startEvictLoop()
	return h
}

// SetProcessLimits sets execution limits for launched plugin processes.
func (h *Host) SetProcessLimits(limits process.Limits) {
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
func (h *Host) SetActivationPolicy(policy lifecycle.ActivationPolicy) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.activation = policy
	for _, sup := range h.supervisors {
		sup.SetActivationPolicy(policy)
	}
}

// SetErrorHandler registers a callback for problems the host meets outside a call, such as a failed shutdown.
func (h *Host) SetErrorHandler(handler eventbus.ErrorHandler) {
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
func (h *Host) SetOnPluginFailed(fn func(pluginport.PluginFailure)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onPluginFailed = fn
}

// SetOnFailureCleared registers a callback invoked with the id of a plugin whose failure the host forgets, because the plugin started, was enabled or was disabled.
func (h *Host) SetOnFailureCleared(fn func(pluginID string)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onFailureCleared = fn
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
		if sup.GetState() != lifecycle.StateDormant {
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
	if err := h.teardownTarget(ctx, target, stopPluginGracePeriod); err != nil && !errors.Is(err, lifecycle.ErrProcessNotRunning) {
		return fmt.Errorf("stop plugin %s: %w", pluginID, err)
	}
	return nil
}

// teardownTarget stops the process of a target and closes its stdin pipe.
func (h *Host) teardownTarget(ctx context.Context, target pluginTeardownTarget, gracePeriod time.Duration) error {
	rpcShutdown := h.createRPCShutdown(target.endpoint, gracePeriod)
	err := target.sup.Teardown(ctx, gracePeriod, rpcShutdown)
	h.closeStdin(target.id, target.pid)
	h.clearPluginWork(target.id)
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
func (h *Host) GetEventBus() *eventbus.Bus {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.bus
}

// GetPlugins returns all discovered plugins registered with the host, in the order of their ids.
func (h *Host) GetPlugins() []manifestfile.DiscoveredPlugin {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.safeMode {
		return nil
	}

	return h.getPluginsInOrderLocked()
}

// getPluginsInOrderLocked returns the registered plugins in the order of their
// ids. Every lookup and listing of the host uses this order, so the first match
// is the plugin with the lowest id.
func (h *Host) getPluginsInOrderLocked() []manifestfile.DiscoveredPlugin {
	result := make([]manifestfile.DiscoveredPlugin, 0, len(h.plugins))
	for _, p := range h.plugins {
		result = append(result, p)
	}
	slices.SortFunc(result, func(a, b manifestfile.DiscoveredPlugin) int { return strings.Compare(a.Manifest.ID, b.Manifest.ID) })
	return result
}

// GetPlugin retrieves a single plugin by its unique identifier.
func (h *Host) GetPlugin(id string) (manifestfile.DiscoveredPlugin, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.safeMode {
		return manifestfile.EmptyDiscoveredPlugin(), false
	}

	p, ok := h.plugins[id]
	if !ok || !h.isPluginEnabledLocked(id) {
		return manifestfile.EmptyDiscoveredPlugin(), false
	}
	return p, true
}

// ResolveCapability finds the enabled plugin with the lowest id among those declaring the given
// capability. Where none is enabled, the error is a *port.UnavailableError: PluginDisabled when a
// plugin declares it and it is switched off or the host is in safe mode, PluginNotInstalled when
// none declares it.
func (h *Host) ResolveCapability(capability string) (manifestfile.DiscoveredPlugin, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	reason := pluginport.PluginNotInstalled
	for _, p := range h.getPluginsInOrderLocked() {
		if !p.Manifest.HasCapability(capability) {
			continue
		}
		if h.safeMode || !h.isPluginEnabledLocked(p.Manifest.ID) {
			reason = pluginport.PluginDisabled
			continue
		}
		return p, nil
	}
	missing := &pluginport.UnavailableError{Capability: capability, Reason: reason}
	if h.safeMode {
		missing.Err = ErrSafeModeActive
	}
	return manifestfile.EmptyDiscoveredPlugin(), missing
}

// ResolveService finds the enabled plugin with the lowest id among those that serve the service to
// the windows.
func (h *Host) ResolveService(service string) (manifestfile.DiscoveredPlugin, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.safeMode {
		return manifestfile.EmptyDiscoveredPlugin(), false
	}
	for _, p := range h.getPluginsInOrderLocked() {
		if p.Manifest.HasService(service) && h.isPluginEnabledLocked(p.Manifest.ID) {
			return p, true
		}
	}
	return manifestfile.EmptyDiscoveredPlugin(), false
}

// ResolveNoteProfile returns how the index stores the notes of a type, as
// defined by the enabled plugin with the lowest id among those declaring the
// type.
func (h *Host) ResolveNoteProfile(noteType domain.NoteType) (domain.NoteProfile, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.safeMode {
		return domain.NoteProfile{}, false
	}
	for _, p := range h.getPluginsInOrderLocked() {
		if declared, found := p.Manifest.GetNoteType(string(noteType)); found && h.isPluginEnabledLocked(p.Manifest.ID) {
			return getProfileOf(declared), true
		}
	}
	return domain.NoteProfile{}, false
}

// getProfileOf is how the index keeps the notes of a declared type.
func getProfileOf(declared pluginport.PluginNoteType) domain.NoteProfile {
	return declared.GetProfile()
}

// FindPluginByCapability finds the plugin ResolveCapability finds, and whether there is one.
func (h *Host) FindPluginByCapability(capability string) (manifestfile.DiscoveredPlugin, bool) {
	p, err := h.ResolveCapability(capability)
	return p, err == nil
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
		if p.Manifest.CanActivateOn(event) || h.isOpeningFileLocked(p, event) {
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
		deliverErr := transport.DeliverPluginEvent(ctx, endpoint, event)
		h.Release(id)
		activated = append(activated, id)
		if deliverErr != nil {
			errs = append(errs, fmt.Errorf("deliver %s to plugin %s: %w", event, id, deliverErr))
		}
	}
	return activated, errors.Join(errs...)
}

func (h *Host) prepareSupervisorLocked(pluginID string) (manifestfile.DiscoveredPlugin, *lifecycle.Supervisor, error) {
	if h.isClosed {
		return manifestfile.DiscoveredPlugin{}, nil, errHostClosed
	}
	if h.safeMode {
		return manifestfile.DiscoveredPlugin{}, nil, &pluginport.UnavailableError{PluginID: pluginID, Reason: pluginport.PluginDisabled, Err: ErrSafeModeActive}
	}

	p, ok := h.plugins[pluginID]
	if !ok {
		return manifestfile.DiscoveredPlugin{}, nil, &pluginport.UnavailableError{
			PluginID: pluginID, Reason: pluginport.PluginNotInstalled, Err: errPluginNotFound,
		}
	}

	h.registerSupervisorLocked(p)
	sup := h.supervisors[pluginID]

	if !h.isPluginEnabledLocked(pluginID) {
		return manifestfile.DiscoveredPlugin{}, nil, &pluginport.UnavailableError{PluginID: pluginID, Reason: pluginport.PluginDisabled}
	}
	if _, isBuiltIn := h.getBuiltInLocked(pluginID); !isBuiltIn && !p.Manifest.HasBackend() {
		return manifestfile.DiscoveredPlugin{}, nil, &pluginport.UnavailableError{PluginID: pluginID, Reason: pluginport.PluginWithoutBackend}
	}

	return p, sup, nil
}

func (h *Host) awaitActivation(ctx context.Context, pluginID string, sup *lifecycle.Supervisor) (transport.Endpoint, bool, error) {
	waitCh := sup.ActivatingChan()
	select {
	case <-ctx.Done():
		return transport.Endpoint{Host: "", Port: 0, Token: ""}, false, ctx.Err()
	case <-waitCh:
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if endpoint, ok := h.acquireRunningLocked(pluginID, sup); ok {
		return endpoint, true, nil
	}
	if err := sup.ActivationError(); err != nil {
		return transport.Endpoint{Host: "", Port: 0, Token: ""}, false, err
	}
	return transport.Endpoint{Host: "", Port: 0, Token: ""}, false, nil
}

// dropEndpointOfStoppedPlugin forgets the endpoint of an exited process unless a replacement is already running.
func (h *Host) dropEndpointOfStoppedPlugin(pluginID string, sup *lifecycle.Supervisor) {
	h.mu.Lock()
	state := sup.GetState()
	isStopped := state == lifecycle.StateDormant || state == lifecycle.StateActivating
	if isStopped {
		delete(h.endpoints, pluginID)
	}
	h.mu.Unlock()
	if isStopped {
		h.clearPluginWork(pluginID)
	}
}

// acquireRunningLocked holds the plugin for a consumer when it is running with a usable endpoint.
func (h *Host) acquireRunningLocked(pluginID string, sup *lifecycle.Supervisor) (transport.Endpoint, bool) {
	endpoint := h.endpoints[pluginID]
	if isEmptyEndpoint(endpoint) || endpoint.Token == "" || !sup.Acquire(time.Now()) {
		return emptyEndpoint(), false
	}
	return endpoint, true
}

// failActivation returns a plugin whose activation failed to Dormant and counts the failure in the same step,
// so no other caller can start it again before the backoff applies.
func (h *Host) failActivation(ctx context.Context, pluginID string, sup *lifecycle.Supervisor, cause error) {
	h.mu.Lock()
	if sup.GetState() != lifecycle.StateActivating {
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

func (h *Host) newPluginFailureLocked(pluginID string, sup *lifecycle.Supervisor, cause error) pluginport.PluginFailure {
	return pluginport.PluginFailure{
		PluginID: pluginID,
		Name:     h.plugins[pluginID].Manifest.Name,
		Reason:   lifecycle.DescribeFailure(cause),
		Detail:   cause.Error(),
		At:       sup.GetFailedAt(),
	}
}

// deliverPluginFailures passes the plugins that failed permanently since the
// previous call to the failure callback, outside every lock.
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
	sup *lifecycle.Supervisor,
	state lifecycle.ProcessState,
) (transport.Endpoint, bool, error) {
	switch state {
	case lifecycle.StateDeactivating:
		err := sup.WaitWhileDeactivating(ctx)
		return emptyEndpoint(), err != nil, err
	case lifecycle.StateActivating:
		endpoint, ready, err := h.awaitActivation(ctx, pluginID, sup)
		return endpoint, ready || err != nil, err
	default:
		if err := sup.Teardown(ctx, 0, nil); err != nil && !errors.Is(err, lifecycle.ErrProcessNotRunning) {
			return emptyEndpoint(), true, fmt.Errorf("retire unreachable plugin %s: %w", pluginID, err)
		}
		return emptyEndpoint(), false, nil
	}
}

// EnsureActive ensures the plugin process is running and returns its endpoint.
func (h *Host) EnsureActive(ctx context.Context, pluginID string) (transport.Endpoint, error) {
	endpoint, err := h.ensureActiveDuringDiscovery(ctx, pluginID)
	h.deliverPluginFailures()
	return endpoint, err
}

// ensureActiveDuringDiscovery activates a plugin while holding off a discovery that could retire it.
func (h *Host) ensureActiveDuringDiscovery(ctx context.Context, pluginID string) (transport.Endpoint, error) {
	h.discoverMu.RLock()
	defer h.discoverMu.RUnlock()

	for {
		h.mu.Lock()
		p, sup, err := h.prepareSupervisorLocked(pluginID)
		if err != nil {
			h.mu.Unlock()
			return transport.Endpoint{Host: "", Port: 0, Token: ""}, err
		}

		if endpoint, ok := h.acquireRunningLocked(pluginID, sup); ok {
			h.mu.Unlock()
			return endpoint, nil
		}

		if state := sup.GetState(); state != lifecycle.StateDormant {
			h.mu.Unlock()
			endpoint, isDone, err := h.settlePluginInState(ctx, pluginID, sup, state)
			if isDone {
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
func (h *Host) beginActivationLocked(pluginID string, sup *lifecycle.Supervisor) error {
	if err := sup.CheckActivationAllowed(time.Now()); err != nil {
		return fmt.Errorf("activate plugin %s: %w", pluginID, err)
	}
	sup.SetActivating()
	return nil
}

func (h *Host) runActivation(ctx context.Context, p manifestfile.DiscoveredPlugin, sup *lifecycle.Supervisor) (transport.Endpoint, error) {
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
	return transport.WaitForHandshake(ctx, stdout)
}

//nolint:contextcheck // background supervisor outlives request
func (h *Host) startPluginProcess(ctx context.Context, p manifestfile.DiscoveredPlugin, sup *lifecycle.Supervisor) (transport.Endpoint, error) {
	token, err := transport.GenerateAuthToken()
	if err != nil {
		return transport.Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("generate token: %w", err)
	}
	if token == "" {
		return transport.Endpoint{Host: "", Port: 0, Token: ""}, errors.New("empty auth token")
	}

	h.mu.RLock()
	launcher := h.launcher
	h.mu.RUnlock()
	if launcher == nil {
		launcher = h.defaultLauncher
	}

	proc, stdout, err := launcher(ctx, p, 0, token)
	if err != nil {
		return transport.Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("launch plugin %s: %w", p.Manifest.ID, err)
	}

	actualPort, err := h.waitForReady(ctx, stdout)
	if err != nil {
		h.cleanupAbortedProcess(proc, stdout, p.Manifest.ID)
		return transport.Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("handshake plugin %s: %w", p.Manifest.ID, err)
	}

	go func() {
		defer stdout.Close()
		_, _ = io.Copy(io.Discard, stdout)
	}()

	endpoint := transport.Endpoint{
		Host:  "127.0.0.1",
		Port:  actualPort,
		Token: token,
	}
	if err := transport.CheckPluginProtocol(ctx, p.Manifest, endpoint); err != nil {
		h.cleanupAbortedProcess(proc, stdout, p.Manifest.ID)
		return emptyEndpoint(), err
	}

	now := time.Now()
	h.mu.Lock()
	if sup.GetState() != lifecycle.StateActivating {
		h.mu.Unlock()
		h.cleanupAbortedProcess(proc, stdout, p.Manifest.ID)
		return transport.Endpoint{Host: "", Port: 0, Token: ""}, lifecycle.ErrActivationAborted
	}

	if err := sup.Attach(proc, now); err != nil {
		h.mu.Unlock()
		h.cleanupAbortedProcess(proc, stdout, p.Manifest.ID)
		return transport.Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("attach plugin %s: %w", p.Manifest.ID, err)
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

func resolvePluginBinary(p manifestfile.DiscoveredPlugin) (string, error) {
	if p.Manifest.Backend.Entrypoint == "" {
		return "", security.ErrEmptyEntrypoint
	}

	entrypoint := p.Manifest.Backend.Entrypoint
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(entrypoint), ".exe") {
		entrypoint += ".exe"
	}

	binPath, err := security.ValidateExecutablePath(p.Dir, entrypoint)
	if err != nil {
		return "", fmt.Errorf("validate plugin binary: %w", err)
	}
	return binPath, nil
}

func (h *Host) defaultLauncher(
	ctx context.Context,
	p manifestfile.DiscoveredPlugin,
	port int,
	token string,
) (*os.Process, io.ReadCloser, error) {
	binPath, err := resolvePluginBinary(p)
	if err != nil {
		return nil, nil, err
	}

	h.mu.RLock()
	vaultPath := h.vaultPath
	limits := h.processLimits
	h.mu.RUnlock()

	if p.Manifest.Lifecycle.MemoryLimitBytes > 0 {
		limits.MemoryLimitBytes = p.Manifest.Lifecycle.MemoryLimitBytes
	}

	workDir, err := process.ValidateWorkingDirectory(p.Dir, p.Dir, vaultPath)
	if err != nil {
		return nil, nil, fmt.Errorf("validate plugin working directory: %w", err)
	}

	cmd := exec.CommandContext(context.WithoutCancel(ctx), binPath, process.GetPluginArgs(port)...)
	process.ConfigureSysProcAttr(cmd)
	cmd.Dir = workDir

	pluginVars, err := h.getLaunchVars(ctx, p.Manifest.ID, h.getRequirements(p), port, token)
	if err != nil {
		return nil, nil, err
	}
	cmd.Env = process.SanitizeEnvironment(os.Environ(), pluginVars, p.Manifest.GetRuntimeIDs())

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

	if _, err := io.WriteString(stdin, token+"\n"); err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, nil, fmt.Errorf("write token: %w", err)
	}

	if err := process.ApplyLimits(cmd.Process.Pid, limits); err != nil {
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
	sup      *lifecycle.Supervisor
	endpoint transport.Endpoint
}

// EvictIdle stops idle plugins whose idle duration exceeds their configured timeout.
func (h *Host) EvictIdle(ctx context.Context, defaultThreshold, gracePeriod time.Duration) []string {
	h.mu.Lock()
	now := time.Now()
	var targets []pluginTeardownTarget

	for id, sup := range h.supervisors {
		threshold := defaultThreshold
		if p, ok := h.plugins[id]; ok {
			if p.Manifest.Lifecycle.IdleTimeoutSeconds == 0 {
				continue
			}
			if p.Manifest.Lifecycle.IdleTimeoutSeconds > 0 {
				threshold = time.Duration(p.Manifest.Lifecycle.IdleTimeoutSeconds) * time.Second
			}
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

	evicted := h.evictTargets(ctx, targets, gracePeriod)
	for _, target := range targets {
		if slices.Contains(evicted, target.id) {
			target.sup.RecordActivationFailure(time.Now(), h.activation, fmt.Errorf("%w: %d bytes", lifecycle.ErrMemoryExceeded, target.sup.GetMemoryLimit()))
			h.reportMemoryExceeded(target.id, target.sup)
		}
	}
	return evicted
}

func (h *Host) reportMemoryExceeded(pluginID string, sup *lifecycle.Supervisor) {
	memLimit := sup.GetMemoryLimit()
	err := fmt.Errorf("plugin %s was stopped: it used more than %d bytes of memory", pluginID, memLimit)
	h.reportError(err)
	h.mu.Lock()
	if failed, cause := sup.GetActivationFailure(); failed {
		h.failedPlugins = append(h.failedPlugins, h.newPluginFailureLocked(pluginID, sup, cause))
	}
	h.mu.Unlock()
	h.deliverPluginFailures()
}

// evictTargets stops the targets and returns the ids that stopped; a failed stop is reported to the error handler.
func (h *Host) evictTargets(ctx context.Context, targets []pluginTeardownTarget, gracePeriod time.Duration) []string {
	evicted := make([]string, 0, len(targets))
	for _, target := range targets {
		if err := h.teardownTarget(ctx, target, gracePeriod); err != nil && !errors.Is(err, lifecycle.ErrProcessNotRunning) {
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
		if sup.GetState() != lifecycle.StateDormant {
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

func (h *Host) createRPCShutdown(endpoint transport.Endpoint, gracePeriod time.Duration) func(context.Context) error {
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
			transport.EndpointClient(endpoint),
			endpoint.BaseURL(),
			connect.WithInterceptors(transport.AuthInterceptor(endpoint.Token)),
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

// applyEnabledChange stops a plugin that was disabled and clears the failures of one that was enabled or disabled.
func (h *Host) applyEnabledChange(ctx context.Context, pluginID string, value any) error {
	enabled, ok := value.(bool)
	if !ok {
		return nil
	}
	var stopErr error
	if !enabled {
		stopErr = h.stopPlugin(ctx, pluginID)
	}
	h.mu.RLock()
	sup := h.supervisors[pluginID]
	h.mu.RUnlock()
	if sup != nil {
		h.clearFailure(pluginID, sup)
	}
	return stopErr
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
		return sup.GetState() == lifecycle.StateActive || sup.GetState() == lifecycle.StateIdle
	}
	return false
}

// GetPluginSettings returns resolved settings for a plugin with defaults overlaid by stored values.
func (h *Host) GetPluginSettings(pluginID string) map[string]any {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.getPluginSettingsLocked(pluginID)
}

// getPluginSettingsLocked returns the values in force of the settings of the plugin, the settings whose
// condition fails left out.
func (h *Host) getPluginSettingsLocked(pluginID string) map[string]any {
	rawStored := h.settings[pluginID]
	result := maps.Clone(rawStored)
	if p, ok := h.plugins[pluginID]; ok {
		result = p.Manifest.ResolveSettings(result)
		if _, hasStored := rawStored[keyExecutablePath]; !hasStored && h.hasExecutablePathSetting(p.Manifest) {
			if paths := resolveDiscoveredACPPaths(); paths != "" {
				result[keyExecutablePath] = paths
			}
		}
	}
	if result == nil {
		result = map[string]any{}
	}
	delete(result, pluginport.SettingStoredSecrets)
	delete(result, pluginport.SettingAIConsent)
	if p, ok := h.plugins[pluginID]; ok {
		for _, name := range p.Manifest.GetSecretNames() {
			delete(result, name)
		}
	}
	return result
}

func (h *Host) hasExecutablePathSetting(manifest pluginport.PluginManifest) bool {
	for _, s := range manifest.Settings {
		if s.Key == keyExecutablePath {
			return true
		}
	}
	return false
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
func (h *Host) SetPluginSetting(ctx context.Context, pluginID, key string, value any) error {
	if key == pluginport.SettingStoredSecrets {
		return fmt.Errorf("plugin %s: %w: %s", pluginID, pluginport.ErrReservedSettingKey, key)
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
	if key == pluginport.SettingStoredSecrets {
		return nil
	}
	h.mu.Lock()
	before := h.getRequirementIDsLocked(pluginID)
	if _, ok := h.settings[pluginID]; !ok {
		h.settings[pluginID] = make(map[string]any)
	}
	h.settings[pluginID][key] = value
	isRequirementChanged := !slices.Equal(before, h.getRequirementIDsLocked(pluginID))
	h.mu.Unlock()

	if isRequirementChanged && key != settingEnabled {
		if err := h.stopPlugin(ctx, pluginID); err != nil {
			return err
		}
	}

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
	if state != lifecycle.StateActive && state != lifecycle.StateIdle {
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
	secretNames := h.getActiveSecretNamesLocked(pluginID)
	stored := h.getStoredSecretsLocked(pluginID)
	store := h.keyStore
	h.mu.RUnlock()

	secrets, failures := readPluginSecrets(ctx, store, pluginID, secretNames, stored)
	secretsErr := h.getSecretsError(failures)
	for name, key := range secrets {
		values[name] = key
	}

	httpClient := transport.EndpointClient(endpoint)
	client := numenv1connect.NewPluginLifecycleServiceClient(
		httpClient,
		endpoint.BaseURL(),
		connect.WithInterceptors(transport.AuthInterceptor(endpoint.Token)),
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

func toPluginSettings(manifestSettings []pluginport.PluginManifestSetting) []pluginport.PluginSetting {
	settings := make([]pluginport.PluginSetting, 0, len(manifestSettings))
	for _, s := range manifestSettings {
		defVal := resolveSettingDefault(s)
		defJSON := ""
		if defVal != nil {
			if b, err := json.Marshal(defVal); err == nil {
				defJSON = string(b)
			}
		}
		var minPtr, maxPtr, stepPtr *float64
		if s.Min != 0 || s.Max != 0 {
			minVal := s.Min
			maxVal := s.Max
			minPtr = &minVal
			maxPtr = &maxVal
		}
		if s.Step != 0 {
			stepVal := s.Step
			stepPtr = &stepVal
		}
		options := s.Options
		if s.Key == keyExecutablePath && len(options) == 0 {
			options = resolveDiscoveredACPOptions()
		}
		settings = append(settings, pluginport.PluginSetting{
			Key:         s.Key,
			Type:        s.Type,
			Title:       s.Title,
			Description: s.Description,
			DefaultJSON: defJSON,
			Palette:     s.Palette,
			Icon:        s.Icon,
			Options:     options,
			Min:         minPtr,
			Max:         maxPtr,
			Step:        stepPtr,
			ModelsOf:    s.ModelsOf,
			Replaces:    s.Replaces,
			OptionsBy:   s.OptionsBy,
			When:        s.When,
			Accept:      s.Accept,
		})
	}
	return settings
}

// GetPluginInfos returns specifications of all registered plugins and their settings schemas, in
// the order of their ids.
func (h *Host) GetPluginInfos() []pluginport.PluginInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]pluginport.PluginInfo, 0, len(h.plugins))
	for _, p := range h.getPluginsInOrderLocked() {
		isActive := h.isPluginActiveLocked(p.Manifest.ID)
		settings := toPluginSettings(p.Manifest.Settings)

		result = append(result, pluginport.PluginInfo{
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
			HasFrontend:   p.Manifest.HasFrontend(),
			Readers:       h.listReaderInfosLocked(p.Manifest),
		})
	}
	return result
}

func resolveSettingDefault(s pluginport.PluginManifestSetting) any {
	if s.Default == "" {
		switch s.Key {
		case "yt_dlp":
			if path, err := exec.LookPath("yt-dlp"); err == nil {
				return path
			}
		case "ffmpeg":
			if path, err := exec.LookPath("ffmpeg"); err == nil {
				return path
			}
		case keyExecutablePath:
			if paths := resolveDiscoveredACPPaths(); paths != "" {
				return paths
			}
		}
	}
	return s.Default
}

type acpExecutableCandidate struct {
	title  string
	binary string
	arg    string
}

const (
	acpSubcommand     = "acp"
	acpFlag           = "--acp"
	keyExecutablePath = "executable_path"
)

var acpExecutableCandidates = []acpExecutableCandidate{
	{title: "Claude Code (ACP)", binary: "claude-code-acp"},
	{title: "Codex (ACP)", binary: "codex-acp"},
	{title: "Cursor Agent", binary: "cursor-agent", arg: acpSubcommand},
	{title: "Gemini CLI", binary: "gemini", arg: acpFlag},
	{title: "Goose", binary: "goose", arg: acpSubcommand},
	{title: "Kimi CLI", binary: "kimi", arg: acpSubcommand},
	{title: "OpenCode", binary: "opencode", arg: acpSubcommand},
	{title: "Qwen Code", binary: "qwen", arg: acpFlag},
}

func resolveDiscoveredACPPaths() string {
	options := resolveDiscoveredACPOptions()
	paths := make([]string, 0, len(options))
	for _, option := range options {
		paths = append(paths, option.Value)
	}
	return strings.Join(paths, "\n")
}

func resolveDiscoveredACPOptions() []pluginport.PluginSettingOption {
	var options []pluginport.PluginSettingOption
	for _, candidate := range acpExecutableCandidates {
		binaryPath, err := exec.LookPath(candidate.binary)
		if err != nil || binaryPath == "" {
			continue
		}
		detail := binaryPath
		if candidate.arg != "" {
			detail = binaryPath + " " + candidate.arg
		}
		options = append(options, pluginport.PluginSettingOption{
			Value:  binaryPath,
			Title:  candidate.title,
			Detail: detail,
		})
	}
	return options
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

func emptyEndpoint() transport.Endpoint {
	return transport.Endpoint{Host: "", Port: 0, Token: ""}
}

func isEmptyEndpoint(e transport.Endpoint) bool {
	return e.Host == "" && e.Port == 0 && e.Transport == nil
}

// getFailureLocked is why the host suspends a plugin, and nil while it does not.
func (h *Host) getFailureLocked(pluginID string) *pluginport.PluginFailure {
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

// clearFailure removes the recorded activation failures of a plugin and passes
// the cleared callback the place where the plugin had been suspended.
func (h *Host) clearFailure(pluginID string, sup *lifecycle.Supervisor) {
	wasFailed, _ := sup.GetActivationFailure()
	sup.ResetActivationFailures()
	if !wasFailed {
		return
	}
	h.mu.RLock()
	notify := h.onFailureCleared
	h.mu.RUnlock()
	if notify != nil {
		notify(pluginID)
	}
}

// RetryPlugin forgets the failures of a plugin and starts it once. The failure is set where the plugin fails
// again, and the host suspends it at once.
func (h *Host) RetryPlugin(ctx context.Context, pluginID string) (*pluginport.PluginFailure, error) {
	h.mu.RLock()
	_, recognized := h.plugins[pluginID]
	sup := h.supervisors[pluginID]
	h.mu.RUnlock()
	if !recognized {
		return nil, &pluginport.UnavailableError{PluginID: pluginID, Reason: pluginport.PluginNotInstalled, Err: errPluginNotFound}
	}
	if sup != nil {
		h.clearFailure(pluginID, sup)
	}

	_, err := h.EnsureActive(ctx, pluginID)
	if err == nil {
		h.Release(pluginID)
		return nil, nil
	}
	var unavailable *pluginport.UnavailableError
	if errors.As(err, &unavailable) || ctx.Err() != nil {
		return nil, err
	}
	failure := h.recordActivationFailure(pluginID, err)
	h.deliverPluginFailures()
	return failure, nil
}

// recordActivationFailure records a failed activation as the last one the plugin is allowed, and returns the failure.
func (h *Host) recordActivationFailure(pluginID string, cause error) *pluginport.PluginFailure {
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

// getRequirements returns the dependencies the plugin is launched with under the settings in force.
func (h *Host) getRequirements(p manifestfile.DiscoveredPlugin) []pluginport.DependencyRequirement {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return p.Manifest.GetDependencyRequirements(p.Dir, h.getPluginSettingsLocked(p.Manifest.ID))
}

// getActiveSecretNamesLocked returns the names of the secrets of the plugin whose condition holds
// under the settings in force.
func (h *Host) getActiveSecretNamesLocked(pluginID string) []string {
	p, ok := h.plugins[pluginID]
	if !ok {
		return nil
	}
	values := h.getPluginSettingsLocked(pluginID)
	var names []string
	for _, secret := range p.Manifest.Secrets {
		if p.Manifest.IsSecretActive(secret, values) {
			names = append(names, secret.Name)
		}
	}
	return names
}

// getRequirementIDsLocked returns the ids of the dependencies the plugin is launched with under the
// settings in force. A plugin launched with others is stopped, and the next call launches it with them.
func (h *Host) getRequirementIDsLocked(pluginID string) []string {
	p, ok := h.plugins[pluginID]
	if !ok {
		return nil
	}
	var ids []string
	for _, req := range p.Manifest.GetDependencyRequirements(p.Dir, h.getPluginSettingsLocked(pluginID)) {
		ids = append(ids, req.ID)
	}
	return ids
}
