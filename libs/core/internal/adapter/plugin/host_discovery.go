package plugin

import (
	"context"
	"errors"
	"fmt"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
	"path/filepath"
	"reflect"
)

// Discover scans the configured directories and synchronizes the registered plugins, and the ports bound
// for their listeners, with what is on disk. A plugin whose directory could not be read stays
// registered.
func (h *Host) Discover() error {
	if err := h.discoverPlugins(); err != nil {
		return err
	}
	h.syncAllListeners(context.Background())
	return nil
}

func (h *Host) discoverPlugins() error {
	h.discoverMu.Lock()
	defer h.discoverMu.Unlock()

	all, problems := manifestjson.Discover(h.dirs)
	discovered := manifest.FilterByTarget(all, h.target)

	h.mu.Lock()
	if h.isClosed {
		h.mu.Unlock()
		return errHostClosed
	}
	if h.safeMode {
		h.mu.Unlock()
		return nil
	}
	h.problems = problems
	h.installed = all
	h.hasScanned = true
	discovered, refused := h.refuseBuiltInProviderIDsLocked(discovered)
	stalePlugins := h.registerDiscoveredLocked(discovered)
	h.mu.Unlock()

	for _, problem := range problems {
		h.reportError(problem.Err)
	}
	for _, err := range refused {
		h.reportError(err)
	}
	for _, id := range stalePlugins {
		h.retirePlugin(id)
		h.mu.RLock()
		isInstalled := h.onDisk[id]
		h.mu.RUnlock()
		if !isInstalled {
			h.removeCacheDir(id)
		}
	}
	h.pruneSecretPermissions()
	return nil
}

func (h *Host) registerDiscoveredLocked(discovered []manifest.Discovered) []string {
	present := make(map[string]bool, len(discovered))
	var stale []string
	for _, p := range discovered {
		id := p.Manifest.ID
		present[id] = true
		if recognized, ok := h.plugins[id]; ok && !reflect.DeepEqual(recognized, p) {
			stale = append(stale, id)
		}
		h.plugins[id] = p
		h.onDisk[id] = true
		h.registerSupervisorLocked(p)
	}
	for id := range h.onDisk {
		if !present[id] && !h.isCoveredByProblemLocked(id) {
			h.removeDiskPluginLocked(id)
			stale = append(stale, id)
		}
	}
	return stale
}

func (h *Host) removeDiskPluginLocked(pluginID string) {
	delete(h.onDisk, pluginID)
	if builtIn, ok := h.getBuiltInLocked(pluginID); ok {
		h.plugins[pluginID] = builtIn.plugin
		h.registerSupervisorLocked(builtIn.plugin)
		return
	}
	delete(h.plugins, pluginID)
}

func (h *Host) isCoveredByProblemLocked(pluginID string) bool {
	dir := h.plugins[pluginID].Dir
	for _, problem := range h.problems {
		if problem.Dir == dir || problem.Dir == filepath.Dir(dir) {
			return true
		}
	}
	return false
}

func (h *Host) registerSupervisorLocked(p manifest.Discovered) {
	id := p.Manifest.ID
	sup, exists := h.supervisors[id]
	if !exists {
		sup = lifecycle.New(h.clock)
		sup.SetOnExit(func() { h.dropEndpointOfStoppedPlugin(id) })
		sup.SetOnError(h.reportError)
		h.supervisors[id] = sup
	}
	sup.SetPermissions(p.Manifest.Permissions)
	sup.SetActivationPolicy(h.activation)
}

func (h *Host) retirePlugin(pluginID string) {
	if err := h.stopPlugin(context.Background(), pluginID); err != nil {
		h.reportError(err)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, registered := h.plugins[pluginID]; !registered {
		delete(h.supervisors, pluginID)
	}
}

var errProviderIDTaken = errors.New("a built-in plugin of this id provides connection types")

// refuseBuiltInProviderIDsLocked leaves out the plugins found on disk under the id of a built-in plugin
// that provides connection types, because the keys of a connection go to the plugin that provides its
// type. It returns the plugins to register and the refusals.
func (h *Host) refuseBuiltInProviderIDsLocked(discovered []manifest.Discovered) ([]manifest.Discovered, []error) {
	var kept []manifest.Discovered
	var refused []error
	for _, p := range discovered {
		builtIn, isBuiltIn := h.getBuiltInLocked(p.Manifest.ID)
		if isBuiltIn && len(builtIn.plugin.Manifest.Contributes.ConnectionTypes) > 0 {
			refused = append(refused, fmt.Errorf("%w: refused the plugin in %s", errProviderIDTaken, p.Dir))
			continue
		}
		kept = append(kept, p)
	}
	return kept, refused
}
