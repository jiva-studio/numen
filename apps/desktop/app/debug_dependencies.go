//go:build debug

package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// placeholderDependencies answers every dependency a plugin declares with an
// empty file in dir, and fetches nothing. Plugins running on mock models open
// none of them, so a run on mock models needs neither the network nor what the
// machine has installed.
type placeholderDependencies struct {
	dir string

	mu         sync.Mutex
	registered map[string]pluginport.DependencyRequirement
	observer   func(pluginport.DependencyRequirement, pluginport.DependencyStatus)
}

func newPlaceholderDependencies(dir string) *placeholderDependencies {
	return &placeholderDependencies{dir: dir, registered: make(map[string]pluginport.DependencyRequirement)}
}

var unsafeNameChars = regexp.MustCompile(`[^A-Za-z0-9._-]`)

func cleanName(name string) string {
	clean := unsafeNameChars.ReplaceAllString(name, "_")
	if strings.Trim(clean, ".") == "" {
		return "_"
	}
	return clean
}

// getEnvVars are the variables the host sets for a ready dependency, pointing at primary.
func getEnvVars(req pluginport.DependencyRequirement, primary string) map[string]string {
	if req.Kind == pluginport.DependencyKindRuntime && req.ID == pluginport.RuntimeONNX {
		return map[string]string{"NUMEN_ONNX_RUNTIME_LIB": primary, "NUMEN_ONNX_PROVIDER": "cpu"}
	}
	name := req.Metadata["envVar"]
	if name == "" {
		name = "NUMEN_DEPENDENCY_" + strings.ToUpper(cleanName(req.ID))
	}
	return map[string]string{name: primary}
}

func (p *placeholderDependencies) CanProvision(pluginport.DependencyRequirement) bool { return true }

func (p *placeholderDependencies) Provision(
	_ context.Context, req pluginport.DependencyRequirement, onProgress func(pluginport.DependencyStatus),
) (pluginport.ResolvedDependency, error) {
	root := filepath.Join(p.dir, cleanName(string(req.Kind)), cleanName(req.ID))
	if err := os.MkdirAll(root, 0o755); err != nil {
		return pluginport.ResolvedDependency{}, err
	}
	name := req.Metadata["fileName"]
	if name == "" {
		name = req.ID
	}
	primary := filepath.Join(root, cleanName(name))
	if err := os.WriteFile(primary, nil, 0o755); err != nil {
		return pluginport.ResolvedDependency{}, fmt.Errorf("placeholder for %s: %w", req.ID, err)
	}
	resolved := pluginport.ResolvedDependency{
		ID: req.ID, RootPath: root, PrimaryPath: primary,
		EnvVars: getEnvVars(req, primary), Metadata: req.Metadata,
	}
	status := readyStatus(req.ID, resolved)
	if onProgress != nil {
		onProgress(status)
	}
	p.mu.Lock()
	observer := p.observer
	p.mu.Unlock()
	if observer != nil {
		observer(req, status)
	}
	return resolved, nil
}

func readyStatus(id string, resolved pluginport.ResolvedDependency) pluginport.DependencyStatus {
	return pluginport.DependencyStatus{
		ID: id, State: pluginport.DependencyStateReady, ProgressPercent: 100, Resolved: &resolved,
	}
}

func (p *placeholderDependencies) Status(ctx context.Context, req pluginport.DependencyRequirement) (pluginport.DependencyStatus, error) {
	resolved, err := p.Provision(ctx, req, nil)
	if err != nil {
		return pluginport.DependencyStatus{}, err
	}
	return readyStatus(req.ID, resolved), nil
}

func (p *placeholderDependencies) Register(req pluginport.DependencyRequirement) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.registered[req.ID] = req
}

func (p *placeholderDependencies) GetStatus(ctx context.Context, id string) (pluginport.DependencyStatus, error) {
	req, found := p.Lookup(id)
	if !found {
		return pluginport.DependencyStatus{}, fmt.Errorf("%w: %s", pluginport.ErrNoProvisioner, id)
	}
	return p.Status(ctx, req)
}

func (p *placeholderDependencies) List(ctx context.Context) ([]pluginport.DependencyStatus, error) {
	p.mu.Lock()
	reqs := make([]pluginport.DependencyRequirement, 0, len(p.registered))
	for _, req := range p.registered {
		reqs = append(reqs, req)
	}
	p.mu.Unlock()
	sort.Slice(reqs, func(i, j int) bool { return reqs[i].ID < reqs[j].ID })
	out := make([]pluginport.DependencyStatus, 0, len(reqs))
	for _, req := range reqs {
		status, err := p.Status(ctx, req)
		if err != nil {
			return nil, err
		}
		out = append(out, status)
	}
	return out, nil
}

func (p *placeholderDependencies) SetObserver(observer func(pluginport.DependencyRequirement, pluginport.DependencyStatus)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.observer = observer
}

func (p *placeholderDependencies) Lookup(id string) (pluginport.DependencyRequirement, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	req, found := p.registered[id]
	return req, found
}
