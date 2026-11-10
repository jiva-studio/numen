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
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/note/domain"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/eventbus"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/process"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/security"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

var (
	errPluginNotFound = errors.New("plugin not found")
	ErrSafeModeActive = errors.New("plugin host is running in safe mode")
	errHostClosed     = errors.New("plugin host is shut down")
)

// Launcher is a function that spawns a plugin process and provides its stdout stream.
type Launcher func(ctx context.Context, p manifest.Discovered, port int, token string) (*os.Process, io.ReadCloser, error)

// PluginStatus describes the enablement and runtime status of a plugin.
type PluginStatus struct {
	ID            string `json:"id"`
	Dir           string `json:"dir,omitempty"`
	Enabled       bool   `json:"enabled"`
	IsActive      bool   `json:"isActive"`
	Failed        bool   `json:"failed"`
	FailureReason string `json:"failureReason,omitempty"`
}

const settingEnabled = "enabled"

// Host manages the discovery, lifecycle, activation, and communication endpoints of plugins.
type Host struct {
	mu                     sync.RWMutex
	target                 string
	dirs                   []string
	plugins                map[string]manifest.Discovered
	supervisors            map[string]*lifecycle.Supervisor
	endpoints              map[string]plugindomain.Endpoint
	launcher               Launcher
	safeMode               bool
	vaultPath              string
	bus                    *eventbus.Bus
	settings               map[string]map[string]any
	performance            string
	persistHandler         plugindomain.PersistHandler
	evictTicker            *time.Ticker
	evictDone              chan struct{}
	evictWg                sync.WaitGroup
	isEvictStopped         bool
	isClosed               bool
	stdinsMu               sync.Mutex
	stdins                 map[stdinKey]io.Closer
	processLimits          process.Limits
	activation             lifecycle.ActivationPolicy
	onPluginFailed         func(plugindomain.Failure)
	onFailureCleared       func(pluginID string)
	errorHandler           eventbus.ErrorHandler
	onDisk                 map[string]bool
	discoverMu             sync.RWMutex
	failedPlugins          []plugindomain.Failure
	problems               []manifestjson.DiscoveryProblem
	installed              []manifest.Discovered
	hasScanned             bool
	completer              hostendpoint.Completer
	vault                  *hostendpoint.VaultAccess
	texts                  hostendpoint.TextVersions
	window                 hostendpoint.WindowControl
	vaultList              hostendpoint.VaultList
	search                 *hostendpoint.SearchAccess
	secrets                *secretDelivery
	hostEndpoint           *hostendpoint.Endpoint
	hostAddress            string
	dependencies           pluginport.DependencyRegistry
	work                   hostendpoint.WorkQueue
	builtIns               map[string]builtInPlugin
	contentTypes           contentTypeTable
	typeCache              contentTypeCache
	clock                  machineport.Clock
	storage                storageRoots
	listenersMu            sync.Mutex
	listeners              map[string]map[string]listenerBinding
	listenerPorts          map[string]map[string]int
	reportedTargetFailures map[string]struct{}
}

// NewHost creates a new plugin host for the specified target application and search paths.
// The host reads the time from clock.
func NewHost(clock machineport.Clock, target string, dirs ...string) *Host {
	h := &Host{
		clock:                  clock,
		mu:                     sync.RWMutex{},
		target:                 target,
		dirs:                   dirs,
		plugins:                make(map[string]manifest.Discovered),
		supervisors:            make(map[string]*lifecycle.Supervisor),
		endpoints:              make(map[string]plugindomain.Endpoint),
		launcher:               nil,
		safeMode:               false,
		vaultPath:              "",
		bus:                    nil,
		settings:               make(map[string]map[string]any),
		persistHandler:         nil,
		evictTicker:            time.NewTicker(30 * time.Second),
		evictDone:              make(chan struct{}),
		isClosed:               false,
		stdins:                 make(map[stdinKey]io.Closer),
		processLimits:          process.DefaultLimits(),
		activation:             lifecycle.DefaultActivationPolicy(),
		onPluginFailed:         nil,
		onFailureCleared:       nil,
		errorHandler:           nil,
		onDisk:                 make(map[string]bool),
		discoverMu:             sync.RWMutex{},
		failedPlugins:          nil,
		problems:               nil,
		completer:              nil,
		hostEndpoint:           nil,
		hostAddress:            hostendpoint.Address,
		dependencies:           nil,
		work:                   nil,
		builtIns:               make(map[string]builtInPlugin),
		contentTypes:           nil,
		typeCache:              contentTypeCache{},
		listeners:              make(map[string]map[string]listenerBinding),
		listenerPorts:          make(map[string]map[string]int),
		reportedTargetFailures: make(map[string]struct{}),
	}
	h.secrets = newSecretDelivery(secretHost{
		getConfigSnapshot: h.getConfigSnapshot,
		listPluginIDs:     h.listPluginIDs,
		reportError:       h.reportError,
	})
	h.bus = eventbus.New(h, clock, 50*time.Millisecond, 250*time.Millisecond)
	h.startEvictLoop()
	return h
}

// SetProcessLimits sets execution limits for launched plugin processes.
func (h *Host) SetProcessLimits(limits process.Limits) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.processLimits = limits
}

// SetActivationPolicy sets the activation deadline of plugins.
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

// SetOnPluginFailed registers a callback invoked each time a start of a plugin fails.
func (h *Host) SetOnPluginFailed(fn func(plugindomain.Failure)) {
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
	h.syncAllListeners(context.Background())
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

func (h *Host) stopPlugin(ctx context.Context, pluginID string) error {
	h.mu.Lock()
	sup, ok := h.supervisors[pluginID]
	if !ok {
		h.mu.Unlock()
		return nil
	}
	target := pluginTeardownTarget{id: pluginID, pid: sup.GetProcessID(), sup: sup, endpoint: h.endpoints[pluginID]}
	h.forgetEndpointLocked(pluginID)
	h.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, stopPluginTimeout)
	defer cancel()
	if err := h.teardownTarget(ctx, target, stopPluginGracePeriod); err != nil && !errors.Is(err, lifecycle.ErrProcessNotRunning) {
		return fmt.Errorf("stop plugin %s: %w", pluginID, err)
	}
	return nil
}

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
func (h *Host) GetPlugins() []manifest.Discovered {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.safeMode {
		return nil
	}

	return h.getPluginsInOrderLocked()
}

func (h *Host) getPluginsInOrderLocked() []manifest.Discovered {
	result := make([]manifest.Discovered, 0, len(h.plugins))
	for _, p := range h.plugins {
		result = append(result, p)
	}
	slices.SortFunc(result, func(a, b manifest.Discovered) int { return strings.Compare(a.Manifest.ID, b.Manifest.ID) })
	return result
}

// GetPlugin retrieves a single plugin by its unique identifier.
func (h *Host) GetPlugin(id string) (manifest.Discovered, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.safeMode {
		return manifest.Discovered{}, false
	}

	p, ok := h.plugins[id]
	if !ok || !h.isPluginEnabledLocked(id) {
		return manifest.Discovered{}, false
	}
	return p, true
}

// ResolveCapability finds the enabled plugin with the lowest id among those declaring the given
// capability. Where none is enabled, the error is a *plugin.UnavailableError: Disabled when a
// plugin declares it and it is switched off or the host is in safe mode, NotInstalled when
// none declares it.
func (h *Host) ResolveCapability(capability string) (manifest.Discovered, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	reason := plugindomain.NotInstalled
	for _, p := range h.getPluginsInOrderLocked() {
		if !p.Manifest.HasCapability(capability) {
			continue
		}
		if h.safeMode || !h.isPluginEnabledLocked(p.Manifest.ID) {
			reason = plugindomain.Disabled
			continue
		}
		return p, nil
	}
	missing := &plugindomain.UnavailableError{Capability: capability, Reason: reason}
	if h.safeMode {
		missing.Err = ErrSafeModeActive
	}
	return manifest.Discovered{}, missing
}

// ResolveService finds the enabled plugin with the lowest id among those that serve the service to
// the windows.
func (h *Host) ResolveService(service string) (manifest.Discovered, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.safeMode {
		return manifest.Discovered{}, false
	}
	for _, p := range h.getPluginsInOrderLocked() {
		if p.Manifest.HasService(service) && h.isPluginEnabledLocked(p.Manifest.ID) {
			return p, true
		}
	}
	return manifest.Discovered{}, false
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

func getProfileOf(declared manifest.NoteType) domain.NoteProfile {
	return declared.GetProfile()
}

// FindPluginByCapability finds the plugin ResolveCapability finds, and whether there is one.
func (h *Host) FindPluginByCapability(capability string) (manifest.Discovered, bool) {
	p, err := h.ResolveCapability(capability)
	return p, err == nil
}

// TriggerEvent activates every enabled plugin whose manifest declares an activation event matching event, and delivers the event to it.
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
		if p.Manifest.IsActivatedBy(event) {
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

func (h *Host) prepareSupervisorLocked(pluginID string) (manifest.Discovered, *lifecycle.Supervisor, error) {
	if h.isClosed {
		return manifest.Discovered{}, nil, errHostClosed
	}
	if h.safeMode {
		return manifest.Discovered{}, nil, &plugindomain.UnavailableError{PluginID: pluginID, Reason: plugindomain.Disabled, Err: ErrSafeModeActive}
	}

	p, ok := h.plugins[pluginID]
	if !ok {
		return manifest.Discovered{}, nil, &plugindomain.UnavailableError{
			PluginID: pluginID, Reason: plugindomain.NotInstalled, Err: errPluginNotFound,
		}
	}

	h.registerSupervisorLocked(p)
	sup := h.supervisors[pluginID]

	if !h.isPluginEnabledLocked(pluginID) {
		return manifest.Discovered{}, nil, &plugindomain.UnavailableError{PluginID: pluginID, Reason: plugindomain.Disabled}
	}
	if _, isBuiltIn := h.getBuiltInLocked(pluginID); !isBuiltIn && !p.Manifest.HasBackend() {
		return manifest.Discovered{}, nil, &plugindomain.UnavailableError{PluginID: pluginID, Reason: plugindomain.WithoutBackend}
	}

	return p, sup, nil
}

func (h *Host) awaitActivation(ctx context.Context, pluginID string, sup *lifecycle.Supervisor) (plugindomain.Endpoint, bool, error) {
	waitCh := sup.ActivatingChan()
	select {
	case <-ctx.Done():
		return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, false, ctx.Err()
	case <-waitCh:
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if endpoint, ok := h.acquireRunningLocked(pluginID, sup); ok {
		return endpoint, true, nil
	}
	if err := sup.ActivationError(); err != nil {
		return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, false, err
	}
	return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, false, nil
}

func (h *Host) dropEndpointOfStoppedPlugin(pluginID string) {
	h.mu.Lock()
	h.forgetEndpointLocked(pluginID)
	h.mu.Unlock()
	h.clearPluginWork(pluginID)
	h.clearPluginTasks(pluginID)
}

func (h *Host) forgetEndpointLocked(pluginID string) {
	delete(h.endpoints, pluginID)
	delete(h.listenerPorts, pluginID)
}

func (h *Host) acquireRunningLocked(pluginID string, sup *lifecycle.Supervisor) (plugindomain.Endpoint, bool) {
	endpoint := h.endpoints[pluginID]
	if isEmptyEndpoint(endpoint) || endpoint.Token == "" || !sup.Acquire(h.clock.Now()) {
		return emptyEndpoint(), false
	}
	return endpoint, true
}

func (h *Host) failActivation(ctx context.Context, pluginID string, sup *lifecycle.Supervisor, cause error) {
	h.mu.Lock()
	if sup.GetState() != lifecycle.StateActivating {
		h.mu.Unlock()
		return
	}
	sup.SetDormant(cause)
	if ctx.Err() == nil {
		sup.RecordActivationFailure(h.clock.Now(), cause)
		h.failedPlugins = append(h.failedPlugins, h.newPluginFailureLocked(pluginID, sup, cause))
	}
	h.mu.Unlock()
}

func (h *Host) newPluginFailureLocked(pluginID string, sup *lifecycle.Supervisor, cause error) plugindomain.Failure {
	return plugindomain.Failure{
		PluginID: pluginID,
		Name:     h.plugins[pluginID].Manifest.Name,
		Reason:   lifecycle.DescribeFailure(cause),
		Detail:   cause.Error(),
		At:       sup.GetFailedAt(),
	}
}

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

func (h *Host) settlePluginInState(
	ctx context.Context,
	pluginID string,
	sup *lifecycle.Supervisor,
	state lifecycle.ProcessState,
) (plugindomain.Endpoint, bool, error) {
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
func (h *Host) EnsureActive(ctx context.Context, pluginID string) (plugindomain.Endpoint, error) {
	endpoint, err := h.ensureActiveDuringDiscovery(ctx, pluginID)
	h.deliverPluginFailures()
	return endpoint, err
}

func (h *Host) ensureActiveDuringDiscovery(ctx context.Context, pluginID string) (plugindomain.Endpoint, error) {
	h.discoverMu.RLock()
	defer h.discoverMu.RUnlock()

	for {
		h.mu.Lock()
		p, sup, err := h.prepareSupervisorLocked(pluginID)
		if err != nil {
			h.mu.Unlock()
			return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, err
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

		h.beginActivationLocked(sup)
		h.mu.Unlock()

		return h.runActivation(ctx, p, sup)
	}
}

func (h *Host) beginActivationLocked(sup *lifecycle.Supervisor) {
	sup.SetActivating()
}

func (h *Host) runActivation(ctx context.Context, p manifest.Discovered, sup *lifecycle.Supervisor) (plugindomain.Endpoint, error) {
	endpoint, err := h.activatePlugin(ctx, p, sup)
	if err != nil {
		h.failActivation(ctx, p.Manifest.ID, sup, err)
		return endpoint, err
	}
	h.secrets.markAwaitingFirstConfigure(p.Manifest.ID)
	h.clearFailure(p.Manifest.ID, sup)

	// No refresh runs between the Configure of the activation and the end of the wait for it.
	// A failed push leaves the plugin running, and it gets its settings again on the next change.
	if lockErr := h.secrets.lockSecrets(ctx); lockErr != nil {
		h.secrets.clearAwaitingFirstConfigure(p.Manifest.ID)
		return endpoint, lockErr
	}
	configureErr := h.secrets.configurePlugin(ctx, p.Manifest.ID)
	h.secrets.clearAwaitingFirstConfigure(p.Manifest.ID)
	h.secrets.unlockSecrets()
	if configureErr != nil {
		h.reportError(fmt.Errorf("push settings to plugin %s: %w", p.Manifest.ID, configureErr))
	}
	return endpoint, nil
}

func (h *Host) cleanupAbortedProcess(proc *os.Process, stdout io.Closer, pluginID string) {
	h.killProcess(proc, pluginID)
	h.closeResource("stdout of plugin "+pluginID, stdout)
	h.closeStdin(pluginID, proc.Pid)
}

func (h *Host) killProcess(proc *os.Process, pluginID string) {
	if err := proc.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		h.reportError(fmt.Errorf("kill plugin %s: %w", pluginID, err))
	}
	if _, err := proc.Wait(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		h.reportError(fmt.Errorf("wait for plugin %s: %w", pluginID, err))
	}
}

func (h *Host) closeResource(name string, c io.Closer) {
	if err := c.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		h.reportError(fmt.Errorf("close %s: %w", name, err))
	}
}

func (h *Host) closeStdin(pluginID string, pid int) {
	key := stdinKey{pluginID: pluginID, pid: pid}
	h.stdinsMu.Lock()
	c, ok := h.stdins[key]
	delete(h.stdins, key)
	h.stdinsMu.Unlock()
	if ok {
		h.closeResource("stdin of plugin "+pluginID, c)
	}
}

func (h *Host) waitForReady(ctx context.Context, stdout io.Reader) (transport.Handshake, error) {
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

func (h *Host) startPluginProcess(ctx context.Context, p manifest.Discovered, sup *lifecycle.Supervisor) (plugindomain.Endpoint, error) {
	token, err := transport.GenerateAuthToken()
	if err != nil {
		return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("generate token: %w", err)
	}
	if token == "" {
		return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, errors.New("empty auth token")
	}

	h.mu.RLock()
	launcher := h.launcher
	h.mu.RUnlock()
	if launcher == nil {
		launcher = h.defaultLauncher
	}

	proc, stdout, err := launcher(ctx, p, 0, token)
	if err != nil {
		return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("launch plugin %s: %w", p.Manifest.ID, err)
	}

	handshake, err := h.waitForReady(ctx, stdout)
	if err == nil {
		err = checkReportedListeners(p.Manifest, handshake.Listeners)
	}
	if err != nil {
		h.cleanupAbortedProcess(proc, stdout, p.Manifest.ID)
		return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("handshake plugin %s: %w", p.Manifest.ID, err)
	}

	go func() {
		if _, err := io.Copy(io.Discard, stdout); err != nil && !errors.Is(err, os.ErrClosed) {
			h.reportError(fmt.Errorf("read output of plugin %s: %w", p.Manifest.ID, err))
		}
		h.closeResource("stdout of plugin "+p.Manifest.ID, stdout)
	}()

	endpoint := plugindomain.Endpoint{
		Host:      "127.0.0.1",
		Port:      handshake.Port,
		Token:     token,
		Transport: transport.NewNetworkTransport(),
	}
	if err := transport.CheckPluginProtocol(ctx, p.Manifest, endpoint); err != nil {
		h.cleanupAbortedProcess(proc, stdout, p.Manifest.ID)
		return emptyEndpoint(), err
	}

	now := h.clock.Now()
	h.mu.Lock()
	if sup.GetState() != lifecycle.StateActivating {
		h.mu.Unlock()
		h.cleanupAbortedProcess(proc, stdout, p.Manifest.ID)
		return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, lifecycle.ErrActivationAborted
	}

	if err := sup.Attach(ctx, proc, now); err != nil {
		h.mu.Unlock()
		h.cleanupAbortedProcess(proc, stdout, p.Manifest.ID)
		return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("attach plugin %s: %w", p.Manifest.ID, err)
	}
	sup.SetOnExit(func() {
		h.dropEndpointOfStoppedPlugin(p.Manifest.ID)

		h.closeStdin(p.Manifest.ID, proc.Pid)
	})
	sup.Acquire(now)

	h.endpoints[p.Manifest.ID] = endpoint
	h.listenerPorts[p.Manifest.ID] = handshake.Listeners
	sup.SetRPCShutdown(h.createRPCShutdown(endpoint, 2*time.Second))
	sup.FinishActivating()
	h.mu.Unlock()

	return endpoint, nil
}

func resolvePluginBinary(p manifest.Discovered) (string, error) {
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
	p manifest.Discovered,
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
		return nil, nil, errors.Join(fmt.Errorf("pipe stdout: %w", err), stdin.Close())
	}

	if err := process.HideInheritedFiles(); err != nil {
		return nil, nil, errors.Join(fmt.Errorf("hide inherited files: %w", err), stdin.Close(), stdout.Close())
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, errors.Join(fmt.Errorf("start %s: %w", binPath, err), stdin.Close(), stdout.Close())
	}

	if _, err := io.WriteString(stdin, token+"\n"); err != nil {
		h.killProcess(cmd.Process, p.Manifest.ID)
		return nil, nil, errors.Join(fmt.Errorf("write token: %w", err), stdin.Close(), stdout.Close())
	}

	if err := process.ApplyLimits(cmd.Process.Pid, limits); err != nil {
		h.killProcess(cmd.Process, p.Manifest.ID)
		return nil, nil, errors.Join(fmt.Errorf("apply process limits: %w", err), stdin.Close(), stdout.Close())
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
		sup.Acquire(h.clock.Now())
	}
}

// Release decrements the active consumer reference count for a plugin.
func (h *Host) Release(pluginID string) {
	h.mu.RLock()
	sup, ok := h.supervisors[pluginID]
	h.mu.RUnlock()
	if ok {
		sup.Release(h.clock.Now())
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
	endpoint plugindomain.Endpoint
}

// idleThreshold is how long a plugin without a lease may stay idle before the host stops it.
const idleThreshold = 180 * time.Second

// EvictIdle stops the plugins that hold no lease and have been idle for longer than threshold.
func (h *Host) EvictIdle(ctx context.Context, threshold, gracePeriod time.Duration) []string {
	h.mu.Lock()
	now := h.clock.Now()
	var targets []pluginTeardownTarget

	for id, sup := range h.supervisors {
		pid := sup.GetProcessID()
		if sup.BeginDeactivation(threshold, now) {
			targets = append(targets, pluginTeardownTarget{
				id:       id,
				pid:      pid,
				sup:      sup,
				endpoint: h.endpoints[id],
			})
			h.forgetEndpointLocked(id)
		}
	}
	h.mu.Unlock()

	return h.evictTargets(ctx, targets, gracePeriod)
}

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
	h.evictWg.Go(func() {
		for {
			select {
			case <-h.evictDone:
				return
			case <-h.evictTicker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_ = h.EvictIdle(ctx, idleThreshold, 2*time.Second)
				cancel()
			}
		}
	})
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

	targets := h.takeTeardownTargets()
	h.closeAllListeners()

	for _, target := range targets {
		if err := h.teardownTarget(ctx, target, gracePeriod); err != nil && firstErr == nil &&
			!errors.Is(err, lifecycle.ErrProcessNotRunning) {
			firstErr = err
		}
	}
	if err := h.closeAllStdins(); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := h.closeHostEndpoint(ctx); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func (h *Host) takeTeardownTargets() []pluginTeardownTarget {
	h.mu.Lock()
	defer h.mu.Unlock()
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
			h.forgetEndpointLocked(id)
		}
	}
	return targets
}

func (h *Host) closeAllStdins() error {
	h.stdinsMu.Lock()
	stdins := h.stdins
	h.stdins = nil
	h.stdinsMu.Unlock()
	var failed []error
	for key, c := range stdins {
		if err := c.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			failed = append(failed, fmt.Errorf("close stdin of plugin %s: %w", key.pluginID, err))
		}
	}
	return errors.Join(failed...)
}

func (h *Host) createRPCShutdown(endpoint plugindomain.Endpoint, gracePeriod time.Duration) func(context.Context) error {
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
			transport.GetBaseURL(endpoint),
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
func (h *Host) SetPersistHandler(handler plugindomain.PersistHandler) {
	h.mu.Lock()
	h.persistHandler = handler
	h.mu.Unlock()
	h.fillGeneratedSettings()
}

// SetSettings loads persisted plugin configurations into host memory.
func (h *Host) SetSettings(all map[string]map[string]any) {
	h.mu.Lock()
	for pluginID, cfg := range all {
		if _, ok := h.settings[pluginID]; !ok {
			h.settings[pluginID] = make(map[string]any)
		}
		maps.Copy(h.settings[pluginID], cfg)
	}
	h.mu.Unlock()
	h.fillGeneratedSettings()
	h.syncAllListeners(context.Background())
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
	settings := maps.Clone(h.settings[pluginID])
	if p, ok := h.plugins[pluginID]; ok {
		settings = p.Manifest.ResolveSettings(settings)
	}
	if val, ok := settings[settingEnabled].(bool); ok {
		return val
	}
	return true
}

// SetPluginEnabled sets whether the plugin is enabled, persists the choice and stops the plugin when it is disabled.
func (h *Host) SetPluginEnabled(ctx context.Context, pluginID string, enabled bool) error {
	return h.SetPluginSetting(ctx, pluginID, settingEnabled, enabled)
}

func (h *Host) applyEnabledChange(ctx context.Context, pluginID string, value any) error {
	enabled, ok := value.(bool)
	if !ok {
		return nil
	}
	var stopErr error
	if !enabled {
		stopErr = h.stopPlugin(ctx, pluginID)
	} else {
		h.fillGeneratedSettings()
	}
	h.mu.RLock()
	sup := h.supervisors[pluginID]
	h.mu.RUnlock()
	if sup != nil {
		h.clearFailure(pluginID, sup)
	}
	h.syncListenersAndReport(ctx, pluginID)
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
		if cause := sup.GetActivationFailure(); cause != nil {
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

func (h *Host) getPluginSettingsLocked(pluginID string) map[string]any {
	rawStored := h.settings[pluginID]
	result := maps.Clone(rawStored)
	if p, ok := h.plugins[pluginID]; ok {
		result = p.Manifest.ResolveSettings(result)
	}
	if result == nil {
		result = map[string]any{}
	}
	if p, ok := h.plugins[pluginID]; ok {
		for _, name := range p.Manifest.GetSecretNames() {
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

// SetPluginSetting persists one setting, then writes it to memory and notifies the plugin. A listener
// port the host cannot bind is rejected before it is persisted.
func (h *Host) SetPluginSetting(ctx context.Context, pluginID, key string, value any) error {
	if err := h.checkListenerPort(ctx, pluginID, key, value); err != nil {
		return err
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
// A value the listeners of the plugin cannot be synced with is replaced by the
// previous one, and the error is returned.
func (h *Host) UpdatePluginSetting(ctx context.Context, pluginID, key string, value any) error {
	h.mu.Lock()
	before := h.getRequirementIDsLocked(pluginID)
	if _, ok := h.settings[pluginID]; !ok {
		h.settings[pluginID] = make(map[string]any)
	}
	previous, hasPrevious := h.settings[pluginID][key]
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
		return h.NotifyPluginConfig(ctx, pluginID)
	}
	if err := h.syncListeners(ctx, pluginID); err != nil {
		h.mu.Lock()
		if values, ok := h.settings[pluginID]; ok && hasPrevious {
			values[key] = previous
		} else if ok {
			delete(values, key)
		}
		h.mu.Unlock()
		h.syncListenersAndReport(ctx, pluginID)
		return err
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
	curVal := false
	if value, isCurrentBool := current[key].(bool); isCurrentBool {
		curVal = value
	}
	newVal := !curVal
	h.mu.RUnlock()

	err := h.SetPluginSetting(ctx, pluginID, key, newVal)
	return newVal, err
}

// NotifyPluginConfig sends the resolved configuration to a running plugin process.
func (h *Host) NotifyPluginConfig(ctx context.Context, pluginID string) error {
	if err := h.secrets.lockSecrets(ctx); err != nil {
		return err
	}
	defer h.secrets.unlockSecrets()
	return h.secrets.configurePlugin(ctx, pluginID)
}

// configSnapshot is what a Configure call to one plugin is made of, read while mu is held.
type configSnapshot struct {
	pluginID   string
	endpoint   plugindomain.Endpoint
	configJSON string
	values     map[string]string
	declared   []manifest.SecretDeclaration
	requested  []string
	store      pluginport.KeyStore
	source     plugindomain.SecretSource
}

func (h *Host) getConfigSnapshot(pluginID string) (configSnapshot, bool, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	endpoint, hasEndpoint := h.endpoints[pluginID]
	sup, hasSup := h.supervisors[pluginID]
	if !hasEndpoint || !hasSup {
		return configSnapshot{}, false, nil
	}
	if state := sup.GetState(); state != lifecycle.StateActive && state != lifecycle.StateIdle {
		return configSnapshot{}, false, nil
	}
	cfgJSON, err := h.getPluginConfigJSONLocked(pluginID)
	if err != nil {
		return configSnapshot{}, true, err
	}
	rawSettings := h.getPluginSettingsLocked(pluginID)
	values := formatPluginSettings(rawSettings)
	if h.performance != "" {
		values[plugindomain.ConfigPerformanceProfile] = h.performance
	}
	return configSnapshot{
		pluginID:   pluginID,
		endpoint:   endpoint,
		configJSON: cfgJSON,
		values:     values,
		declared:   h.listActiveDeclarationsLocked(pluginID, rawSettings),
		requested:  plugindomain.ListAskedSecretGroups(h.listDeclarationsLocked()),
	}, true, nil
}

func (h *Host) listActiveDeclarationsLocked(pluginID string, values map[string]any) []manifest.SecretDeclaration {
	p, ok := h.plugins[pluginID]
	if !ok {
		return nil
	}
	var active []manifest.SecretDeclaration
	for _, declaration := range p.Manifest.Secrets {
		if p.Manifest.IsSecretActive(declaration, values) {
			active = append(active, declaration)
		}
	}
	return active
}

func (h *Host) toPluginSettings(manifestSettings []manifest.Setting) []plugindomain.Setting {
	settings := make([]plugindomain.Setting, 0, len(manifestSettings))
	for _, s := range manifestSettings {
		defVal, options := s.Default, s.Options
		defJSON := ""
		if defVal != nil {
			if b, err := json.Marshal(defVal); err == nil {
				defJSON = string(b)
			}
		}
		settings = append(settings, plugindomain.NewSetting(s, defJSON, options))
	}
	return settings
}

func resolveSettings(declared manifest.Manifest, settings []plugindomain.Setting, values map[string]any) {
	for index := range settings {
		one, found := declared.GetSetting(settings[index].Key)
		if !found {
			continue
		}
		settings[index].ActiveOptions = declared.GetSettingOptions(one, values)
		if len(settings[index].ActiveOptions) == 0 {
			settings[index].ActiveOptions = settings[index].Options
		}
		settings[index].IsShown = declared.IsSettingShown(one, values)
		if value, ok := values[one.Key]; ok {
			if encoded, err := json.Marshal(value); err == nil {
				settings[index].ValueJSON = string(encoded)
			}
		}
	}
}

// GetPluginInfos returns specifications of all registered plugins and their settings schemas, in
// the order of their ids.
func (h *Host) GetPluginInfos() []plugindomain.Info {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]plugindomain.Info, 0, len(h.plugins))
	for _, p := range h.getPluginsInOrderLocked() {
		isActive := h.isPluginActiveLocked(p.Manifest.ID)
		settings := h.toPluginSettings(p.Manifest.Settings)
		resolveSettings(p.Manifest, settings, h.getPluginSettingsLocked(p.Manifest.ID))

		result = append(result, plugindomain.Info{
			ID:            p.Manifest.ID,
			Name:          p.Manifest.Name,
			Version:       p.Manifest.Version,
			Description:   p.Manifest.Description,
			Capabilities:  p.Manifest.Capabilities,
			Settings:      settings,
			IsActive:      isActive,
			Permissions:   p.Manifest.Permissions,
			Secrets:       p.Manifest.Secrets,
			Failure:       h.getFailureLocked(p.Manifest.ID),
			HasFrontend:   p.Manifest.HasFrontend(),
			HasBackend:    p.Manifest.HasBackend(),
			Readers:       h.listReaderInfosLocked(p.Manifest),
			SettingsViews: slices.Clone(p.Manifest.SettingsViews),
		})
	}
	return result
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

func emptyEndpoint() plugindomain.Endpoint {
	return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}
}

func isEmptyEndpoint(e plugindomain.Endpoint) bool {
	return e.Host == "" && e.Port == 0 && e.Transport == nil
}

// IsPluginSuspended reports whether the last start of the plugin failed, which holds it suspended until
// its failure is cleared.
func (h *Host) IsPluginSuspended(pluginID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	sup, ok := h.supervisors[pluginID]
	return ok && sup.GetActivationFailure() != nil
}

func (h *Host) getFailureLocked(pluginID string) *plugindomain.Failure {
	sup, ok := h.supervisors[pluginID]
	if !ok {
		return nil
	}
	cause := sup.GetActivationFailure()
	if cause == nil {
		return nil
	}
	failure := h.newPluginFailureLocked(pluginID, sup, cause)
	return &failure
}

func (h *Host) clearFailure(pluginID string, sup *lifecycle.Supervisor) {
	h.mu.Lock()
	delete(h.reportedTargetFailures, pluginID)
	h.mu.Unlock()
	if !sup.ClearActivationFailure() {
		return
	}
	h.mu.RLock()
	notify := h.onFailureCleared
	h.mu.RUnlock()
	if notify != nil {
		notify(pluginID)
	}
}

// RetryPlugin starts a plugin once, as any use of it does, and answers why the start failed where it did.
func (h *Host) RetryPlugin(ctx context.Context, pluginID string) (*plugindomain.Failure, error) {
	h.mu.RLock()
	_, recognized := h.plugins[pluginID]
	h.mu.RUnlock()
	if !recognized {
		return nil, &plugindomain.UnavailableError{PluginID: pluginID, Reason: plugindomain.NotInstalled, Err: errPluginNotFound}
	}
	h.mu.Lock()
	delete(h.reportedTargetFailures, pluginID)
	h.mu.Unlock()
	defer h.syncListenersAndReport(ctx, pluginID)

	_, err := h.EnsureActive(ctx, pluginID)
	if err == nil {
		h.Release(pluginID)
		return nil, nil
	}
	var unavailable *plugindomain.UnavailableError
	if errors.As(err, &unavailable) || ctx.Err() != nil {
		return nil, err
	}
	h.mu.RLock()
	failure := h.getFailureLocked(pluginID)
	h.mu.RUnlock()
	return failure, nil
}

func (h *Host) getRequirements(p manifest.Discovered) []manifest.DependencyRequirement {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return p.Manifest.GetDependencyRequirements(p.Dir, h.getPluginSettingsLocked(p.Manifest.ID))
}

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

// SetPerformanceProfile records the profile of the performance setting and pushes it to every
// running plugin when it changed. A plugin that is not running receives it with its first
// Configure call.
func (h *Host) SetPerformanceProfile(ctx context.Context, profile string) {
	h.mu.Lock()
	if h.performance == profile {
		h.mu.Unlock()
		return
	}
	h.performance = profile
	ids := make([]string, 0, len(h.endpoints))
	for id := range h.endpoints {
		ids = append(ids, id)
	}
	h.mu.Unlock()

	for _, id := range ids {
		if err := h.NotifyPluginConfig(ctx, id); err != nil {
			h.reportError(fmt.Errorf("push the performance profile to plugin %s: %w", id, err))
		}
	}
}
