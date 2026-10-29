package plugin

import (
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/manifestfile"
	"net/http"
)

// builtInPlugin is a plugin linked into the host's binary: the plugin its manifest describes and
// what makes the handler that serves its backend from the variables a launched process of it is
// given.
type builtInPlugin struct {
	plugin manifestfile.DiscoveredPlugin
	serve  func(vars map[string]string) (http.Handler, error)
}

// getBuiltInLocked returns the built-in plugin registered under an id.
func (h *Host) getBuiltInLocked(pluginID string) (builtInPlugin, bool) {
	builtIn, ok := h.builtIns[pluginID]
	return builtIn, ok
}
