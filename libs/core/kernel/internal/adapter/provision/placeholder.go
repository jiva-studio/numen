package provision

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// PlaceholderProvisioner answers a dependency with an empty file and fetches
// nothing. It is what the host provisions with while the plugins run on mock
// models, which open no model and load no runtime. A program the machine holds
// is still the machine's own.
type PlaceholderProvisioner struct {
	dir         string
	discovery   discovery
	runtimeVars map[string]func(primary string) map[string]string
}

// NewPlaceholderProvisioner keeps its placeholders in dir. runtimeVars names, by
// runtime id, the variables a runtime is handed to a plugin by; a runtime it
// does not name is handed by the variable GetEnvName names.
func NewPlaceholderProvisioner(dir string, runtimeVars map[string]func(primary string) map[string]string) *PlaceholderProvisioner {
	return &PlaceholderProvisioner{dir: dir, discovery: newDiscovery(), runtimeVars: runtimeVars}
}

// CanProvision reports true: every dependency has a placeholder.
func (p *PlaceholderProvisioner) CanProvision(manifest.DependencyRequirement) bool { return true }

// Provision returns the program the machine holds, or a placeholder.
func (p *PlaceholderProvisioner) Provision(
	_ context.Context, req manifest.DependencyRequirement, _ func(plugin.DependencyStatus),
) (plugin.ResolvedDependency, error) {
	if req.Kind == manifest.DependencyKindBinary {
		if found := p.discovery.find(req); found != "" {
			return newResolvedDependency(req, filepath.Dir(found), found), nil
		}
	}
	root := getDependencyDir(p.dir, string(req.Kind), req.ID, req.Version)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return plugin.ResolvedDependency{}, err
	}
	name := req.Metadata["fileName"]
	if !isPlainName(name) {
		name = cleanName(req.ID)
	}
	primary := filepath.Join(root, name)
	if err := os.WriteFile(primary, nil, 0o600); err != nil {
		return plugin.ResolvedDependency{}, fmt.Errorf("placeholder for %s: %w", req.ID, err)
	}
	resolved := newResolvedDependency(req, root, primary)
	if vars, ok := p.runtimeVars[req.ID]; ok && req.Kind == manifest.DependencyKindRuntime {
		resolved.EnvVars = vars(primary)
	}
	return resolved, nil
}

// Status answers ready, with the placeholder made.
func (p *PlaceholderProvisioner) Status(ctx context.Context, req manifest.DependencyRequirement) (plugin.DependencyStatus, error) {
	resolved, err := p.Provision(ctx, req, nil)
	if err != nil {
		return plugin.DependencyStatus{}, err
	}
	return plugin.DependencyStatus{
		ID: req.ID, State: plugin.DependencyStateReady, ProgressPercent: 100, Resolved: &resolved,
	}, nil
}

// GetPlaceholderDir returns where placeholders are kept: the numen folder of the user's cache, or a
// temporary folder while the machine names no cache, or a folder under the working folder where it cannot make
// one, which still lets a mock run start.
func GetPlaceholderDir() string {
	dir, err := os.UserCacheDir()
	if err == nil {
		return filepath.Join(dir, "numen", "placeholder-dependencies")
	}
	if dir, err = os.MkdirTemp("", "numen-placeholder-dependencies-"); err != nil {
		return "numen-placeholder-dependencies"
	}
	return dir
}
