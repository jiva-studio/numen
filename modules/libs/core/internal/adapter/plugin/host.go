package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	errPluginNotFound  = errors.New("plugin not found")
	errEmptyEntrypoint = errors.New("plugin entrypoint is empty")
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
}

// NewHost creates a new plugin host for the specified target application and search paths.
func NewHost(target string, dirs ...string) *Host {
	return &Host{
		mu:          sync.RWMutex{},
		target:      target,
		dirs:        dirs,
		plugins:     make(map[string]DiscoveredPlugin),
		supervisors: make(map[string]*Supervisor),
		endpoints:   make(map[string]Endpoint),
		launcher:    defaultLauncher,
	}
}

// SetLauncher overrides the process launcher for testing.
func (h *Host) SetLauncher(l Launcher) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.launcher = l
}

// Discover scans configured directories and registers eligible plugins.
func (h *Host) Discover() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	all, err := DiscoverPlugins(h.dirs)
	if err != nil {
		return fmt.Errorf("discover plugins: %w", err)
	}

	discovered := FilterPluginsByTarget(all, h.target)

	for _, p := range discovered {
		h.plugins[p.Manifest.ID] = p
		if _, exists := h.supervisors[p.Manifest.ID]; !exists {
			h.supervisors[p.Manifest.ID] = NewSupervisor()
		}
	}
	return nil
}

// GetPlugins returns all discovered plugins registered with the host.
func (h *Host) GetPlugins() []DiscoveredPlugin {
	h.mu.RLock()
	defer h.mu.RUnlock()

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
	p, ok := h.plugins[id]
	return p, ok
}

// FindPluginByCapability finds the first registered plugin that declares the given capability.
func (h *Host) FindPluginByCapability(capability string) (DiscoveredPlugin, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, p := range h.plugins {
		if p.Manifest.HasCapability(capability) {
			return p, true
		}
	}
	return emptyDiscoveredPlugin(), false
}

// TriggerEvent activates all dormant or idle plugins registered for the given event.
func (h *Host) TriggerEvent(ctx context.Context, event string) ([]string, error) {
	h.mu.RLock()
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

	binPath := filepath.Join(p.Dir, entrypoint)
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
