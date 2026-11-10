package plugin

import (
	"context"
	"crypto/subtle"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/agenttoolsbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/process"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
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

func (h *Host) SetPluginVault(vault *hostendpoint.VaultAccess) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.vault = vault
}

func (h *Host) GetPluginVault() *hostendpoint.VaultAccess {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.vault
}

// SetPluginTexts is the text versions the plugins work through the host.
func (h *Host) SetPluginTexts(texts hostendpoint.TextVersions) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.texts = texts
}

// GetPluginTexts is the text versions the plugins work, or nil.
func (h *Host) GetPluginTexts() hostendpoint.TextVersions {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.texts
}

// SetPluginWindow sets the window exposed to plugins. Nil means none.
func (h *Host) SetPluginWindow(window hostendpoint.WindowControl) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.window = window
}

// GetPluginWindow returns the window exposed to plugins, or nil.
func (h *Host) GetPluginWindow() hostendpoint.WindowControl {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.window
}

// SetPluginVaultList sets the list of vaults exposed to plugins. Nil means none.
func (h *Host) SetPluginVaultList(list hostendpoint.VaultList) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.vaultList = list
}

// GetPluginVaultList returns the list of vaults exposed to plugins, or nil.
func (h *Host) GetPluginVaultList() hostendpoint.VaultList {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.vaultList
}

// SetCompleter sets what answers the completions plugins ask for. The host
// endpoint starts serving when the first plugin launches after it is set.
func (h *Host) SetCompleter(completer hostendpoint.Completer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.completer = completer
}

func (h *Host) GetCompleter() hostendpoint.Completer {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.completer
}

// SetPluginSearch is the search index and the search floor the plugins work through the host.
func (h *Host) SetPluginSearch(search *hostendpoint.SearchAccess) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.search = search
}

// GetPluginSearch is the search index the plugins work, or nil.
func (h *Host) GetPluginSearch() *hostendpoint.SearchAccess {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.search
}

// GetEndpointCallers is what the host lends the endpoint plugins call it through.
func (h *Host) GetEndpointCallers() hostendpoint.Callers {
	return hostendpoint.Callers{
		Identity: h, Leases: h, Completion: h, Vault: h, Texts: h, Search: h, Window: h, Work: h, Dependencies: h, Tools: h,
	}
}

func (h *Host) openHostEndpoint(ctx context.Context) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.hostEndpoint != nil {
		return h.hostEndpoint.GetPort(), nil
	}
	if (h.completer == nil && h.vault == nil && h.dependencies == nil && h.work == nil && h.search == nil) || h.isClosed {
		return 0, nil
	}
	endpoint, err := hostendpoint.Open(ctx, h.hostAddress, h.GetEndpointCallers())
	if err != nil {
		return 0, err
	}
	h.hostEndpoint = endpoint
	return endpoint.GetPort(), nil
}

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

func (h *Host) getLaunchVars(ctx context.Context, pluginID string, requirements []manifest.DependencyRequirement, processPort int, token string) ([]string, error) {
	runtimeVars, err := h.getRuntimeVars(ctx, requirements)
	if err != nil {
		return nil, fmt.Errorf("launch plugin %s: %w", pluginID, err)
	}
	return h.getEnvironmentVars(ctx, pluginID, processPort, token, runtimeVars), nil
}

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
func (h *Host) SetPluginWork(work hostendpoint.WorkQueue) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.work = work
}

func (h *Host) GetPluginWork() hostendpoint.WorkQueue {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.work
}

func (h *Host) clearPluginWork(pluginID string) {
	if work := h.GetPluginWork(); work != nil {
		work.ClearPlugin(pluginID)
	}
}

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
