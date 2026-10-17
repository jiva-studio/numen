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
	"strconv"
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
)

// Launcher is a function that spawns a plugin process and provides its stdout stream.
type Launcher func(ctx context.Context, p DiscoveredPlugin, port int, token string) (*os.Process, io.ReadCloser, error)

// PersistHandler persists a changed plugin setting to host configuration.
type PersistHandler = port.PluginPersistHandler

// PluginStatus describes the enablement and runtime status of a plugin.
type PluginStatus struct {
	ID       string `json:"id"`
	Enabled  bool   `json:"enabled"`
	IsActive bool   `json:"isActive"`
}

// Host manages the discovery, lifecycle, activation, and communication endpoints of plugins.
type Host struct {
	mu             sync.RWMutex
	target         string
	dirs           []string
	plugins        map[string]DiscoveredPlugin
	supervisors    map[string]*Supervisor
	endpoints      map[string]Endpoint
	launcher       Launcher
	safeMode       bool
	vaultPath      string
	bus            *EventBus
	settings       map[string]map[string]any
	persistHandler PersistHandler
	evictTicker    *time.Ticker
	evictDone      chan struct{}
	evictWg        sync.WaitGroup
	isEvictStopped bool
	enabled        map[string]bool
	stdinsMu       sync.Mutex
	stdins         map[string]io.Closer
	processLimits  ProcessLimits
}

// NewHost creates a new plugin host for the specified target application and search paths.
func NewHost(target string, dirs ...string) *Host {
	h := &Host{
		mu:             sync.RWMutex{},
		target:         target,
		dirs:           dirs,
		plugins:        make(map[string]DiscoveredPlugin),
		supervisors:    make(map[string]*Supervisor),
		endpoints:      make(map[string]Endpoint),
		launcher:       nil,
		safeMode:       false,
		vaultPath:      "",
		bus:            nil,
		settings:       make(map[string]map[string]any),
		persistHandler: nil,
		evictTicker:    time.NewTicker(30 * time.Second),
		evictDone:      make(chan struct{}),
		enabled:        make(map[string]bool),
		stdins:         make(map[string]io.Closer),
		processLimits:  DefaultProcessLimits(),
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

// SetLauncher overrides the process launcher for testing.
func (h *Host) SetLauncher(l Launcher) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.launcher = l
}

// SetSafeMode enables or disables safe recovery mode.
func (h *Host) SetSafeMode(enabled bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.safeMode = enabled
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

// Discover scans configured directories and registers eligible plugins.
func (h *Host) Discover() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.safeMode {
		return nil
	}

	all, err := DiscoverPlugins(h.dirs)
	if err != nil {
		return fmt.Errorf("discover plugins: %w", err)
	}

	discovered := FilterPluginsByTarget(all, h.target)

	for _, p := range discovered {
		h.plugins[p.Manifest.ID] = p
		sup, exists := h.supervisors[p.Manifest.ID]
		if !exists {
			sup = NewSupervisor()
			id := p.Manifest.ID
			sup.SetOnExit(func() {
				h.mu.Lock()
				delete(h.endpoints, id)
				h.mu.Unlock()
			})
			h.supervisors[p.Manifest.ID] = sup
		}
		sup.SetPermissions(p.Manifest.Permissions)
		memLimit := h.processLimits.MemoryLimitBytes
		if p.Manifest.Lifecycle.MemoryLimitBytes > 0 {
			memLimit = p.Manifest.Lifecycle.MemoryLimitBytes
		}
		sup.SetMemoryLimit(memLimit)
	}
	return nil
}

// GetPlugins returns all discovered plugins registered with the host.
func (h *Host) GetPlugins() []DiscoveredPlugin {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.safeMode {
		return nil
	}

	result := make([]DiscoveredPlugin, 0, len(h.plugins))
	for _, p := range h.plugins {
		result = append(result, p)
	}
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
	return p, ok
}

// FindPluginByCapability finds the first registered plugin that declares the given capability.
func (h *Host) FindPluginByCapability(capability string) (DiscoveredPlugin, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.safeMode {
		return emptyDiscoveredPlugin(), false
	}

	for _, p := range h.plugins {
		if !h.isPluginEnabledLocked(p.Manifest.ID) {
			continue
		}
		if p.Manifest.HasCapability(capability) {
			return p, true
		}
	}
	return emptyDiscoveredPlugin(), false
}

// FindPluginByFileExtension finds the first registered plugin that declares support for the given file extension.
func (h *Host) FindPluginByFileExtension(ext string) (DiscoveredPlugin, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, p := range h.plugins {
		if !h.isPluginEnabledLocked(p.Manifest.ID) {
			continue
		}
		if p.Manifest.HasFileExtension(ext) {
			return p, true
		}
	}
	return emptyDiscoveredPlugin(), false
}

// TriggerEvent activates all dormant or idle plugins registered for the given event.
func (h *Host) TriggerEvent(ctx context.Context, event string) ([]string, error) {
	h.mu.RLock()
	if h.safeMode {
		h.mu.RUnlock()
		return nil, nil
	}
	candidates := make([]string, 0)
	for id, p := range h.plugins {
		if !h.isPluginEnabledLocked(id) {
			continue
		}
		if p.Manifest.CanActivateOn(event) {
			candidates = append(candidates, id)
		}
	}
	h.mu.RUnlock()

	activated := make([]string, 0, len(candidates))
	for _, id := range candidates {
		if _, err := h.EnsureActive(ctx, id); err != nil {
			return activated, fmt.Errorf("activate plugin %s on %s: %w", id, event, err)
		}
		h.Release(id)
		activated = append(activated, id)
	}
	return activated, nil
}

func (h *Host) prepareSupervisorLocked(pluginID string) (DiscoveredPlugin, *Supervisor, error) {
	if h.safeMode {
		return DiscoveredPlugin{}, nil, errSafeModeActive
	}

	p, ok := h.plugins[pluginID]
	if !ok {
		p, ok = h.resolveMonolithPlugin(pluginID)
		if !ok {
			return DiscoveredPlugin{}, nil, errPluginNotFound
		}
		h.plugins[pluginID] = p
	}

	sup, hasSup := h.supervisors[pluginID]
	if !hasSup {
		sup = NewSupervisor()
		sup.SetPermissions(p.Manifest.Permissions)
		id := pluginID
		sup.SetOnExit(func() {
			h.mu.Lock()
			delete(h.endpoints, id)
			h.mu.Unlock()
		})
		h.supervisors[pluginID] = sup
	}
	memLimit := h.processLimits.MemoryLimitBytes
	if p.Manifest.Lifecycle.MemoryLimitBytes > 0 {
		memLimit = p.Manifest.Lifecycle.MemoryLimitBytes
	}
	sup.SetMemoryLimit(memLimit)

	if !h.isPluginEnabledLocked(pluginID) {
		return DiscoveredPlugin{}, nil, fmt.Errorf("plugin %s is disabled", pluginID)
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

// acquireRunningLocked holds the plugin for a consumer when it is running with a usable endpoint.
func (h *Host) acquireRunningLocked(pluginID string, sup *Supervisor) (Endpoint, bool) {
	endpoint := h.endpoints[pluginID]
	if isEmptyEndpoint(endpoint) || !sup.Acquire(time.Now()) {
		return emptyEndpoint(), false
	}
	return endpoint, true
}

func (h *Host) failActivation(sup *Supervisor, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if sup.GetState() == StateActivating {
		sup.SetDormant(err)
	}
}

// EnsureActive ensures the plugin process is running and returns its endpoint.
func (h *Host) EnsureActive(ctx context.Context, pluginID string) (Endpoint, error) {
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

		if state := sup.GetState(); state == StateActivating {
			h.mu.Unlock()
			endpoint, ready, err := h.awaitActivation(ctx, pluginID, sup)
			if ready || err != nil {
				return endpoint, err
			}
			continue
		}

		sup.SetActivating()
		h.mu.Unlock()

		endpoint, err := h.activatePlugin(ctx, p, sup)
		if err != nil {
			h.failActivation(sup, err)
			return endpoint, err
		}

		_ = h.NotifyPluginConfig(ctx, p.Manifest.ID)
		return endpoint, nil
	}
}

func (h *Host) cleanupAbortedProcess(proc *os.Process, stdout io.Closer, pluginID string) {
	_ = proc.Kill()
	_, _ = proc.Wait()
	_ = stdout.Close()
	h.stdinsMu.Lock()
	if c, ok := h.stdins[pluginID]; ok {
		_ = c.Close()
		delete(h.stdins, pluginID)
	}
	h.stdinsMu.Unlock()
}

//nolint:contextcheck // background supervisor outlives request
func (h *Host) startPluginProcess(ctx context.Context, p DiscoveredPlugin, sup *Supervisor) (Endpoint, error) {
	token, err := GenerateAuthToken()
	if err != nil {
		h.mu.Lock()
		sup.SetDormant(err)
		h.mu.Unlock()
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
		h.mu.Lock()
		sup.SetDormant(err)
		h.mu.Unlock()
		return Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("launch plugin %s: %w", p.Manifest.ID, err)
	}

	actualPort, err := WaitForHandshake(ctx, stdout)
	if err != nil {
		h.cleanupAbortedProcess(proc, stdout, p.Manifest.ID)
		h.mu.Lock()
		sup.SetDormant(err)
		h.mu.Unlock()
		return Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("handshake plugin %s: %w", p.Manifest.ID, err)
	}

	go func() {
		defer stdout.Close()
		_, _ = io.Copy(io.Discard, stdout)
	}()

	now := time.Now()
	h.mu.Lock()
	if sup.GetState() != StateActivating {
		h.mu.Unlock()
		h.cleanupAbortedProcess(proc, stdout, p.Manifest.ID)
		return Endpoint{Host: "", Port: 0, Token: ""}, errors.New("plugin activation aborted")
	}

	if err := sup.Attach(proc, now); err != nil {
		sup.SetDormant(err)
		h.mu.Unlock()
		h.cleanupAbortedProcess(proc, stdout, p.Manifest.ID)
		return Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("attach plugin %s: %w", p.Manifest.ID, err)
	}
	sup.SetOnExit(func() {
		h.mu.Lock()
		delete(h.endpoints, p.Manifest.ID)
		h.mu.Unlock()

		h.stdinsMu.Lock()
		if c, ok := h.stdins[p.Manifest.ID]; ok {
			_ = c.Close()
			delete(h.stdins, p.Manifest.ID)
		}
		h.stdinsMu.Unlock()
	})
	sup.Acquire(now)

	endpoint := Endpoint{
		Host:  "127.0.0.1",
		Port:  actualPort,
		Token: token,
	}
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
	configJSON, _ := h.getPluginConfigJSONLocked(p.Manifest.ID)
	limits := h.processLimits
	h.mu.RUnlock()

	if p.Manifest.Lifecycle.MemoryLimitBytes > 0 {
		limits.MemoryLimitBytes = p.Manifest.Lifecycle.MemoryLimitBytes
	}

	workDir, err := ValidateWorkingDirectory(p.Dir, p.Dir, vaultPath)
	if err != nil {
		return nil, nil, fmt.Errorf("validate plugin working directory: %w", err)
	}

	//nolint:contextcheck // the plugin process outlives the request that launched it
	cmd := exec.CommandContext(context.Background(), binPath, "--port="+strconv.Itoa(port), "--token="+token)
	configureSysProcAttr(cmd)
	cmd.Dir = workDir

	pluginVars := []string{
		"NUMEN_VAULT_PATH=" + vaultPath,
		"NUMEN_PLUGIN_PORT=" + strconv.Itoa(port),
		"NUMEN_PLUGIN_TOKEN=" + token,
		"NUMEN_PLUGIN_ID=" + p.Manifest.ID,
		"NUMEN_PLUGIN_CONFIG=" + configJSON,
	}
	cmd.Env = sanitizeEnvironment(os.Environ(), pluginVars)

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
		h.stdins = make(map[string]io.Closer)
	}
	h.stdins[p.Manifest.ID] = stdin
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

type pluginTeardownTarget struct {
	id       string
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

		if sup.IsIdle(threshold, now) {
			targets = append(targets, pluginTeardownTarget{
				id:       id,
				sup:      sup,
				endpoint: h.endpoints[id],
			})
			delete(h.endpoints, id)
		}
	}
	h.mu.Unlock()

	evicted := make([]string, 0, len(targets))
	for _, target := range targets {
		rpcShutdown := h.createRPCShutdown(target.endpoint, gracePeriod)
		_ = target.sup.Teardown(ctx, gracePeriod, rpcShutdown)
		h.stdinsMu.Lock()
		if c, ok := h.stdins[target.id]; ok {
			_ = c.Close()
			delete(h.stdins, target.id)
		}
		h.stdinsMu.Unlock()
		evicted = append(evicted, target.id)
	}
	return evicted
}

// EvictMemoryExceeded stops plugins whose resident memory exceeds their configured limit.
func (h *Host) EvictMemoryExceeded(ctx context.Context, gracePeriod time.Duration) []string {
	h.mu.Lock()
	var targets []pluginTeardownTarget

	for id, sup := range h.supervisors {
		if sup.IsMemoryExceeded() {
			targets = append(targets, pluginTeardownTarget{
				id:       id,
				sup:      sup,
				endpoint: h.endpoints[id],
			})
			delete(h.endpoints, id)
		}
	}
	h.mu.Unlock()

	evicted := make([]string, 0, len(targets))
	for _, target := range targets {
		rpcShutdown := h.createRPCShutdown(target.endpoint, gracePeriod)
		_ = target.sup.Teardown(ctx, gracePeriod, rpcShutdown)
		h.stdinsMu.Lock()
		if c, ok := h.stdins[target.id]; ok {
			_ = c.Close()
			delete(h.stdins, target.id)
		}
		h.stdinsMu.Unlock()
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
	var targets []pluginTeardownTarget
	for id, sup := range h.supervisors {
		if sup.GetState() != StateDormant {
			targets = append(targets, pluginTeardownTarget{
				id:       id,
				sup:      sup,
				endpoint: h.endpoints[id],
			})
			delete(h.endpoints, id)
		}
	}
	h.mu.Unlock()

	for _, target := range targets {
		rpcShutdown := h.createRPCShutdown(target.endpoint, gracePeriod)
		if err := target.sup.Teardown(ctx, gracePeriod, rpcShutdown); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	h.stdinsMu.Lock()
	for id, c := range h.stdins {
		_ = c.Close()
		delete(h.stdins, id)
	}
	h.stdinsMu.Unlock()
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
		if val, ok := cfg["enabled"].(bool); ok {
			h.enabled[pluginID] = val
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
	if val, ok := h.enabled[pluginID]; ok {
		return val
	}
	settings := h.getPluginSettingsLocked(pluginID)
	if val, ok := settings["enabled"].(bool); ok {
		return val
	}
	return true
}

// SetPluginEnabled sets whether the plugin is enabled and updates its configuration.
func (h *Host) SetPluginEnabled(pluginID string, enabled bool) {
	h.mu.Lock()
	if h.enabled == nil {
		h.enabled = make(map[string]bool)
	}
	h.enabled[pluginID] = enabled
	if _, ok := h.settings[pluginID]; !ok {
		h.settings[pluginID] = make(map[string]any)
	}
	h.settings[pluginID]["enabled"] = enabled
	persist := h.persistHandler
	h.mu.Unlock()

	if persist != nil {
		_ = persist(pluginID, "enabled", enabled)
	}
}

// GetPluginStatus returns the status of the specified plugin.
func (h *Host) GetPluginStatus(pluginID string) (PluginStatus, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	p, ok := h.plugins[pluginID]
	if !ok {
		return PluginStatus{}, false
	}
	return PluginStatus{
		ID:       p.Manifest.ID,
		Enabled:  h.isPluginEnabledLocked(p.Manifest.ID),
		IsActive: h.isPluginActiveLocked(p.Manifest.ID),
	}, true
}

// GetPluginStatuses returns statuses for all registered plugins.
func (h *Host) GetPluginStatuses() []PluginStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	statuses := make([]PluginStatus, 0, len(h.plugins))
	for _, p := range h.plugins {
		statuses = append(statuses, PluginStatus{
			ID:       p.Manifest.ID,
			Enabled:  h.isPluginEnabledLocked(p.Manifest.ID),
			IsActive: h.isPluginActiveLocked(p.Manifest.ID),
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

// SetPluginSetting writes one setting value to memory, persists it, and notifies the plugin.
func (h *Host) SetPluginSetting(ctx context.Context, pluginID, key string, value any) error {
	h.mu.Lock()
	if _, ok := h.settings[pluginID]; !ok {
		h.settings[pluginID] = make(map[string]any)
	}
	h.settings[pluginID][key] = value
	persist := h.persistHandler
	h.mu.Unlock()

	if persist != nil {
		if err := persist(pluginID, key, value); err != nil {
			return fmt.Errorf("persist plugin setting: %w", err)
		}
	}
	return h.NotifyPluginConfig(ctx, pluginID)
}

// UpdatePluginSetting updates the setting in memory and notifies the plugin without re-persisting.
func (h *Host) UpdatePluginSetting(ctx context.Context, pluginID, key string, value any) error {
	h.mu.Lock()
	if _, ok := h.settings[pluginID]; !ok {
		h.settings[pluginID] = make(map[string]any)
	}
	h.settings[pluginID][key] = value
	h.mu.Unlock()

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
	for _, p := range h.plugins {
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
	h.mu.RUnlock()

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
		if errors.As(err, &connectErr) && connectErr.Code() == connect.CodeUnimplemented {
			return nil
		}
		return fmt.Errorf("configure plugin %s: %w", pluginID, err)
	}
	return nil
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

// GetPluginInfos returns specifications of all registered plugins and their settings schemas.
func (h *Host) GetPluginInfos() []port.PluginInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]port.PluginInfo, 0, len(h.plugins))
	for _, p := range h.plugins {
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
			ID:           p.Manifest.ID,
			Name:         p.Manifest.Name,
			Version:      p.Manifest.Version,
			Description:  p.Manifest.Description,
			Capabilities: p.Manifest.Capabilities,
			Settings:     settings,
			IsActive:     isActive,
		})
	}
	return result
}

// GetWirePluginInfos returns wire specifications of all registered plugins and their settings schemas.
func (h *Host) GetWirePluginInfos() []*v1.PluginInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]*v1.PluginInfo, 0, len(h.plugins))
	for _, p := range h.plugins {
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
