package provision

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// PlaceholderProvisioner answers a dependency with an empty file and fetches
// nothing. It is what the host provisions with while the plugins run on mock
// models, which open no model and load no runtime. A program the machine holds
// is still the machine's own.
type PlaceholderProvisioner struct {
	dir       string
	discovery discovery
	// runtimeVars are the variables each runtime is named to a plugin by, by runtime id.
	runtimeVars map[string]func(primary string) map[string]string
}

// NewPlaceholderProvisioner keeps its placeholders in dir. runtimeVars names, by
// runtime id, the variables a runtime is handed to a plugin by; a runtime it
// does not name is handed by the variable GetEnvName names.
func NewPlaceholderProvisioner(dir string, runtimeVars map[string]func(primary string) map[string]string) *PlaceholderProvisioner {
	return &PlaceholderProvisioner{dir: dir, discovery: newDiscovery(), runtimeVars: runtimeVars}
}

// CanProvision reports true: every dependency has a placeholder.
func (p *PlaceholderProvisioner) CanProvision(pluginport.DependencyRequirement) bool { return true }

// Provision returns the program the machine holds, or a placeholder.
func (p *PlaceholderProvisioner) Provision(
	_ context.Context, req pluginport.DependencyRequirement, _ func(pluginport.DependencyStatus),
) (pluginport.ResolvedDependency, error) {
	if req.Kind == pluginport.DependencyKindBinary {
		if found := p.discovery.find(req); found != "" {
			return newResolved(req, filepath.Dir(found), found), nil
		}
	}
	root := getDependencyDir(p.dir, string(req.Kind), req.ID, req.Version)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return pluginport.ResolvedDependency{}, err
	}
	name := req.Metadata["fileName"]
	if !isPlainName(name) {
		name = cleanName(req.ID)
	}
	primary := filepath.Join(root, name)
	if err := os.WriteFile(primary, nil, 0o600); err != nil {
		return pluginport.ResolvedDependency{}, fmt.Errorf("placeholder for %s: %w", req.ID, err)
	}
	resolved := newResolved(req, root, primary)
	if vars, ok := p.runtimeVars[req.ID]; ok && req.Kind == pluginport.DependencyKindRuntime {
		resolved.EnvVars = vars(primary)
	}
	return resolved, nil
}

// Status answers ready, with the placeholder made.
func (p *PlaceholderProvisioner) Status(ctx context.Context, req pluginport.DependencyRequirement) (pluginport.DependencyStatus, error) {
	resolved, err := p.Provision(ctx, req, nil)
	if err != nil {
		return pluginport.DependencyStatus{}, err
	}
	return pluginport.DependencyStatus{
		ID: req.ID, State: pluginport.DependencyStateReady, ProgressPercent: 100, Resolved: &resolved,
	}, nil
}
