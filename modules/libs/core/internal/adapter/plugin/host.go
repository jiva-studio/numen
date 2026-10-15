package plugin

import (
	"context"
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
)

var (
	errPluginNotFound  = errors.New("plugin not found")
	errEmptyEntrypoint = errors.New("plugin entrypoint is empty")
	errSafeModeActive  = errors.New("plugin host is running in safe mode")
)

// Launcher is a function that spawns a plugin process and provides its stdout stream.
type Launcher func(ctx context.Context, p DiscoveredPlugin, port int, token string) (*os.Process, io.ReadCloser, error)

// Host manages the discovery, lifecycle, activation, and communication endpoints of plugins.
type Host struct {
	mu          sync.RWMutex
	target      string
	dirs        []string
	plugins     map[string]DiscoveredPlugin
	supervisors map[string]*Supervisor
	endpoints   map[string]Endpoint
	launcher    Launcher
	safeMode    bool
	vaultPath   string
	bus         *EventBus
}

// NewHost creates a new plugin host for the specified target application and search paths.
func NewHost(target string, dirs ...string) *Host {
	h := &Host{
		mu:          sync.RWMutex{},
		target:      target,
		dirs:        dirs,
		plugins:     make(map[string]DiscoveredPlugin),
		supervisors: make(map[string]*Supervisor),
		endpoints:   make(map[string]Endpoint),
		launcher:    defaultLauncher,
		safeMode:    false,
		vaultPath:   "",
		bus:         nil,
	}
	h.bus = NewEventBus(h, 50*time.Millisecond, 250*time.Millisecond)
	return h
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
			h.supervisors[p.Manifest.ID] = sup
		}
		sup.SetPermissions(p.Manifest.Permissions)
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
		activated = append(activated, id)
	}
	return activated, nil
}

// EnsureActive ensures the plugin process is running and returns its endpoint.
func (h *Host) EnsureActive(ctx context.Context, pluginID string) (Endpoint, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.safeMode {
		return Endpoint{Host: "", Port: 0, Token: ""}, errSafeModeActive
	}

	p, ok := h.plugins[pluginID]
	if !ok {
		return Endpoint{Host: "", Port: 0, Token: ""}, errPluginNotFound
	}

	sup := h.supervisors[pluginID]
	state := sup.GetState()
	if state == StateActive || state == StateIdle {
		sup.Acquire(time.Now())
		return h.endpoints[pluginID], nil
	}

	return h.startPluginProcess(ctx, p, sup)
}

func (h *Host) startPluginProcess(ctx context.Context, p DiscoveredPlugin, sup *Supervisor) (Endpoint, error) {
	token, err := GenerateAuthToken()
	if err != nil {
		return Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("generate token: %w", err)
	}

	port, err := FindAvailablePort()
	if err != nil {
		return Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("find port: %w", err)
	}

	proc, stdout, err := h.launcher(ctx, p, port, token)
	if err != nil {
		return Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("launch plugin %s: %w", p.Manifest.ID, err)
	}

	actualPort, err := WaitForHandshake(ctx, stdout)
	if err != nil {
		_ = proc.Kill()
		_, _ = proc.Wait()
		_ = stdout.Close()
		return Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("handshake plugin %s: %w", p.Manifest.ID, err)
	}

	now := time.Now()
	if err := sup.Attach(proc, now); err != nil {
		_ = proc.Kill()
		_, _ = proc.Wait()
		_ = stdout.Close()
		return Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("attach plugin %s: %w", p.Manifest.ID, err)
	}
	sup.Acquire(now)

	endpoint := Endpoint{
		Host:  "127.0.0.1",
		Port:  actualPort,
		Token: token,
	}
	h.endpoints[p.Manifest.ID] = endpoint
	return endpoint, nil
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

// EvictIdle stops idle plugins whose idle duration exceeds their configured timeout.
func (h *Host) EvictIdle(ctx context.Context, defaultThreshold, gracePeriod time.Duration) []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	evicted := make([]string, 0)

	for id, sup := range h.supervisors {
		threshold := defaultThreshold
		if p, ok := h.plugins[id]; ok && p.Manifest.Lifecycle.IdleTimeoutSeconds > 0 {
			threshold = time.Duration(p.Manifest.Lifecycle.IdleTimeoutSeconds) * time.Second
		}

		if sup.IsIdle(threshold, now) {
			_ = sup.Teardown(ctx, gracePeriod, nil)
			delete(h.endpoints, id)
			evicted = append(evicted, id)
		}
	}
	return evicted
}

// TeardownAll shuts down all active plugin processes.
func (h *Host) TeardownAll(ctx context.Context, gracePeriod time.Duration) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.bus != nil {
		h.bus.Close(ctx)
	}

	var firstErr error
	for id, sup := range h.supervisors {
		if sup.GetState() != StateDormant {
			if err := sup.Teardown(ctx, gracePeriod, nil); err != nil && firstErr == nil {
				firstErr = err
			}
			delete(h.endpoints, id)
		}
	}
	return firstErr
}

func defaultLauncher(
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

	cmd := exec.CommandContext(ctx, binPath, "--port="+strconv.Itoa(port), "--token="+token)
	cmd.Dir = p.Dir
	cmd.Env = append(
		os.Environ(),
		"NUMEN_PLUGIN_PORT="+strconv.Itoa(port),
		"NUMEN_PLUGIN_TOKEN="+token,
		"NUMEN_PLUGIN_ID="+p.Manifest.ID,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("pipe stdout: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdout.Close()
		return nil, nil, fmt.Errorf("start %s: %w", binPath, err)
	}

	return cmd.Process, stdout, nil
}
