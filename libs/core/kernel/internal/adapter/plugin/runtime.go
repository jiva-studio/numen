package plugin

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	pluginport "github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// ErrRuntimeUnavailable indicates that a plugin declares a dependency the host cannot provision.
var ErrRuntimeUnavailable = errors.New("runtime unavailable")

// SetDependencies sets what provisions the dependencies the plugins declare and
// what the plugins ask about over the host endpoint.
func (h *Host) SetDependencies(catalog pluginport.DependencyCatalog) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.dependencies = catalog
}

// GetDependencies returns what provisions the dependencies, or nil when nothing does.
func (h *Host) GetDependencies() pluginport.DependencyCatalog {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.dependencies
}

func (h *Host) getRuntimeVars(ctx context.Context, requirements []manifest.DependencyRequirement) ([]string, error) {
	var vars []string
	for _, req := range requirements {
		catalog := h.GetDependencies()
		if catalog == nil || !catalog.CanProvision(req) {
			return nil, fmt.Errorf("%w: nothing provisions %s %s", ErrRuntimeUnavailable, req.Kind, req.ID)
		}
		catalog.Register(req)
		resolved, err := catalog.Provision(ctx, req, nil)
		if err != nil {
			return nil, fmt.Errorf("provision %s: %w", req.ID, err)
		}
		for _, name := range slices.Sorted(maps.Keys(resolved.EnvVars)) {
			if !isDependencyEnvName(name) {
				return nil, fmt.Errorf("provision %s: %w: the variable name %q", req.ID, ErrRuntimeUnavailable, name)
			}
			vars = append(vars, name+"="+resolved.EnvVars[name])
		}
	}
	return vars, nil
}

var dependencyEnvName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func isDependencyEnvName(name string) bool {
	if !dependencyEnvName.MatchString(name) {
		return false
	}
	return name != "NUMEN_VAULT_PATH" && !strings.HasPrefix(name, "NUMEN_PLUGIN_") && !strings.HasPrefix(name, "NUMEN_HOST_")
}
