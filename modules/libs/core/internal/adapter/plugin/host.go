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
)

var (
	errPluginNotFound  = errors.New("plugin not found")
	errEmptyEntrypoint = errors.New("plugin entrypoint is empty")
	errSafeModeActive  = errors.New("plugin host is running in safe mode")
)

// Launcher is a function that spawns a plugin process and provides its stdout stream.
type Launcher func(ctx context.Context, p DiscoveredPlugin, port int, token string) (*os.Process, io.ReadCloser, error)

// PersistHandler persists a changed plugin setting to host configuration.
type PersistHandler func(pluginID string, key string, value any) error

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
	}
	h.bus = NewEventBus(h, 50*time.Millisecond, 250*time.Millisecond)
	h.startEvictLoop()
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

	if h.safeMode {
		h.mu.Unlock()
		return Endpoint{Host: "", Port: 0, Token: ""}, errSafeModeActive
	}

	p, ok := h.plugins[pluginID]
	if !ok {
		p, ok = h.resolveMonolithPlugin(pluginID)
		if !ok {
			h.mu.Unlock()
			return Endpoint{Host: "", Port: 0, Token: ""}, errPluginNotFound
		}
		h.plugins[pluginID] = p
		if _, hasSup := h.supervisors[pluginID]; !hasSup {
			sup := NewSupervisor()
			sup.SetPermissions(p.Manifest.Permissions)
			h.supervisors[pluginID] = sup
		}
	}

	sup := h.supervisors[pluginID]
	state := sup.GetState()
	if state == StateActive || state == StateIdle {
		sup.Acquire(time.Now())
		endpoint := h.endpoints[pluginID]
		h.mu.Unlock()
		return endpoint, nil
	}

	endpoint, err := h.activatePlugin(ctx, p, sup)
	h.mu.Unlock()
	if err != nil {
		return endpoint, err
	}

	_ = h.NotifyPluginConfig(ctx, p.Manifest.ID)
	return endpoint, nil
}

func (h *Host) startPluginProcess(ctx context.Context, p DiscoveredPlugin, sup *Supervisor) (Endpoint, error) {
	token, err := GenerateAuthToken()
	if err != nil {
		return Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("generate token: %w", err)
	}

	launcher := h.launcher
	if launcher == nil {
		launcher = h.defaultLauncher
	}

	proc, stdout, err := launcher(ctx, p, 0, token)
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

func isWhitelistedEnv(key string) bool {
	upper := strings.ToUpper(key)
	switch upper {
	case "PATH", "HOME", "USER", "TMP", "TEMP", "SYSTEMROOT", "SYSTEMDRIVE", "APPDATA", "LOCALAPPDATA", "LANG":
		return true
	}
	return strings.HasPrefix(upper, "LC_")
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

	cmd := exec.CommandContext(ctx, binPath, "--port="+strconv.Itoa(port), "--token="+token)
	cmd.Dir = p.Dir
	configJSON, _ := h.getPluginConfigJSONLocked(p.Manifest.ID)

	var env []string
	for _, entry := range os.Environ() {
		key, _, found := strings.Cut(entry, "=")
		if found && isWhitelistedEnv(key) {
			env = append(env, entry)
		}
	}
	env = append(
		env,
		"NUMEN_VAULT_PATH="+h.vaultPath,
		"NUMEN_PLUGIN_PORT="+strconv.Itoa(port),
		"NUMEN_PLUGIN_TOKEN="+token,
		"NUMEN_PLUGIN_ID="+p.Manifest.ID,
		"NUMEN_PLUGIN_CONFIG="+configJSON,
	)
	cmd.Env = env

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
	values := make(map[string]string, len(rawSettings))
	for k, v := range rawSettings {
		if b, err := json.Marshal(v); err == nil {
			values[k] = string(b)
		} else {
			values[k] = fmt.Sprint(v)
		}
	}
	h.mu.RUnlock()

	httpClient := endpoint.HTTPClient()
	if endpoint.Transport == nil {
		httpClient.Timeout = 3 * time.Second
	}
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
	_, err = client.Configure(ctx, req)
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

// GetPluginInfos returns wire specifications of all registered plugins and their settings schemas.
func (h *Host) GetPluginInfos() []*v1.PluginInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]*v1.PluginInfo, 0, len(h.plugins))
	for _, p := range h.plugins {
		sup, hasSup := h.supervisors[p.Manifest.ID]
		isActive := hasSup && (sup.GetState() == StateActive || sup.GetState() == StateIdle)

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
		Plugins: s.host.GetPluginInfos(),
	}), nil
}
