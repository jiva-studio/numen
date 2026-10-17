package plugin

import (
	"context"
	"reflect"
)

// Discover scans the configured directories and brings the registered plugins in line with what is on disk.
func (h *Host) Discover() error {
	all, problems := DiscoverPlugins(h.dirs)
	discovered := FilterPluginsByTarget(all, h.target)

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
	stale := h.registerDiscoveredLocked(discovered)
	h.mu.Unlock()

	for _, problem := range problems {
		h.reportError(problem.Err)
	}
	for _, id := range stale {
		h.retirePlugin(id)
	}
	return nil
}

// registerDiscoveredLocked registers the discovered plugins and returns the ids to stop, which are the plugins
// that left the disk and the running ones whose manifest changed.
func (h *Host) registerDiscoveredLocked(discovered []DiscoveredPlugin) []string {
	present := make(map[string]bool, len(discovered))
	var stale []string
	for _, p := range discovered {
		id := p.Manifest.ID
		present[id] = true
		if known, ok := h.plugins[id]; ok && !reflect.DeepEqual(known, p) {
			stale = append(stale, id)
		}
		h.plugins[id] = p
		h.onDisk[id] = true
		h.registerSupervisorLocked(p)
	}
	for id := range h.onDisk {
		if !present[id] {
			delete(h.plugins, id)
			delete(h.onDisk, id)
			stale = append(stale, id)
		}
	}
	return stale
}

func (h *Host) registerSupervisorLocked(p DiscoveredPlugin) {
	id := p.Manifest.ID
	sup, exists := h.supervisors[id]
	if !exists {
		sup = NewSupervisor()
		sup.SetOnExit(func() { h.dropEndpointOfStoppedPlugin(id, sup) })
		h.supervisors[id] = sup
	}
	sup.SetPermissions(p.Manifest.Permissions)
	memLimit := h.processLimits.MemoryLimitBytes
	if p.Manifest.Lifecycle.MemoryLimitBytes > 0 {
		memLimit = p.Manifest.Lifecycle.MemoryLimitBytes
	}
	sup.SetMemoryLimit(memLimit)
}

// retirePlugin stops a plugin whose manifest changed or vanished and forgets its supervisor once it is unregistered.
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
