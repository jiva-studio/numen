package plugin

import (
	"context"
	"crypto/subtle"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/agenttoolsbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/process"
	completionport "github.com/jiva-studio/numen/libs/core/usecase/completion/completioncontract"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// GetPluginIDByToken returns the id of the running plugin the token was given
// to. An empty token names none.
func (h *Host) GetPluginIDByToken(token string) (string, bool) {
	if token == "" {
		return "", false
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	var pluginID string
	var found bool
	for id, endpoint := range h.endpoints {
		if subtle.ConstantTimeCompare([]byte(endpoint.Token), []byte(token)) == 1 {
			pluginID, found = id, true
		}
	}
	return pluginID, found
}

func (h *Host) SetPluginVault(vault pluginport.PluginVault) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.vault = vault
}

func (h *Host) GetPluginVault() pluginport.PluginVault {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.vault
}

// SetPluginTexts is the text versions the plugins work through the host.
func (h *Host) SetPluginTexts(texts pluginport.PluginTexts) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.texts = texts
}

// GetPluginTexts is the text versions the plugins work, or nil.
func (h *Host) GetPluginTexts() pluginport.PluginTexts {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.texts
}

// SetPluginWindow is the main window the plugins move and read the tabs of. Nil is none.
func (h *Host) SetPluginWindow(window pluginport.PluginWindow) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.window = window
}

// GetPluginWindow is the main window the plugins move and read the tabs of, or nil.
func (h *Host) GetPluginWindow() pluginport.PluginWindow {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.window
}

// SetCompleter sets what answers the completions plugins ask for. The host
// endpoint starts serving when the first plugin launches after it is set.
func (h *Host) SetCompleter(completer completionport.Completer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.completer = completer
}

func (h *Host) GetCompleter() completionport.Completer {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.completer
}

// SetPluginSearch is the search index and the search floor the plugins work through the host.
func (h *Host) SetPluginSearch(search pluginport.PluginSearch) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.search = search
}

// GetPluginSearch is the search index the plugins work, or nil.
func (h *Host) GetPluginSearch() pluginport.PluginSearch {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.search
}

// openHostEndpoint returns the port the host endpoint serves on, starting it on
// the first call after a completer or a vault is set. It returns zero while
// neither is set.
func (h *Host) openHostEndpoint(ctx context.Context) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.hostEndpoint != nil {
		return h.hostEndpoint.GetPort(), nil
	}
	if (h.completer == nil && h.vault == nil && h.dependencies == nil && h.work == nil && h.search == nil) || h.isClosed {
		return 0, nil
	}
	endpoint, err := hostendpoint.Open(ctx, h.hostAddress, h)
	if err != nil {
		return 0, err
	}
	h.hostEndpoint = endpoint
	return endpoint.GetPort(), nil
}

// closeHostEndpoint stops the host endpoint if it is serving.
func (h *Host) closeHostEndpoint(ctx context.Context) error {
	h.mu.Lock()
	endpoint := h.hostEndpoint
	h.hostEndpoint = nil
	h.mu.Unlock()
	if endpoint == nil {
		return nil
	}
	return endpoint.Close(ctx)
}

// getLaunchVars is the environment variables the host sets for a plugin process
// about to start. The host endpoint is started for it if a completer is set. An
// endpoint that cannot start is reported and leaves the plugin without
// NUMEN_HOST_PORT; the plugins that never call the host launch all the same. Storage directories
// that cannot be created are reported and left out the same way. A
// runtime the plugin declares and the host cannot provision stops the launch.
func (h *Host) getLaunchVars(ctx context.Context, pluginID string, requirements []pluginport.DependencyRequirement, processPort int, token string) ([]string, error) {
	runtimeVars, err := h.getRuntimeVars(ctx, requirements)
	if err != nil {
		return nil, fmt.Errorf("launch plugin %s: %w", pluginID, err)
	}
	return h.getEnvironmentVars(ctx, pluginID, processPort, token, runtimeVars), nil
}

// getEnvironmentVars is the environment variables of a plugin about to start, with the variables of
// the runtimes it was provisioned.
func (h *Host) getEnvironmentVars(ctx context.Context, pluginID string, processPort int, token string, runtimeVars []string) []string {
	hostPort, err := h.openHostEndpoint(ctx)
	if err != nil {
		h.reportError(fmt.Errorf("open host endpoint for plugin %s: %w", pluginID, err))
		hostPort = 0
	}
	storageVars, err := h.getStorageVars(pluginID)
	if err != nil {
		h.reportError(err)
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return process.GetPluginVars(pluginID, process.LaunchEnvironment{
		VaultPath: h.vaultPath, Port: processPort, HostPort: hostPort, Token: token, RuntimeVars: runtimeVars,
		StorageVars: storageVars,
	})
}

// RegisterPlugin adds a plugin with the supervisor that tracks it, as discovery does for a plugin found on disk.
func (h *Host) RegisterPlugin(p manifest.Discovered, sup *lifecycle.Supervisor) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.plugins[p.Manifest.ID] = p
	h.supervisors[p.Manifest.ID] = sup
}

// GetSupervisor returns the supervisor that tracks a plugin.
func (h *Host) GetSupervisor(pluginID string) (*lifecycle.Supervisor, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	sup, ok := h.supervisors[pluginID]
	return sup, ok
}

// HasEndpoint reports whether a plugin has the endpoint of a running process registered.
func (h *Host) HasEndpoint(pluginID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.endpoints[pluginID]
	return ok
}

// SetPluginWork sets the queue and the list of tasks the plugins use.
func (h *Host) SetPluginWork(work pluginport.PluginWork) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.work = work
}

func (h *Host) GetPluginWork() pluginport.PluginWork {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.work
}

// clearPluginWork takes out the tasks and ends the turns of a plugin that
// stopped.
func (h *Host) clearPluginWork(pluginID string) {
	if work := h.GetPluginWork(); work != nil {
		work.ClearPlugin(pluginID)
	}
}

// clearPluginTasks forgets the open tasks of a plugin that stopped.
func (h *Host) clearPluginTasks(pluginID string) {
	h.mu.RLock()
	endpoint := h.hostEndpoint
	h.mu.RUnlock()
	if endpoint != nil {
		endpoint.ClearPluginTasks(pluginID)
	}
}

// GetPluginTools is the tools the plugins with the tools capability offer on the full server, which
// a plugin serves to programs outside Numen.
func (h *Host) GetPluginTools() hostendpoint.PluginTools {
	return agenttoolsbridge.NewCatalog(h, agenttoolsbridge.Full)
}
