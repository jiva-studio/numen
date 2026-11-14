// Package provision makes the dependencies a plugin declares ready on this
// machine: a registry that routes a requirement to the provisioner that handles
// it, and the primitives those provisioners share.
package provision

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Registry routes a requirement to the first provisioner that handles it,
// shares one run between identical requests and reports each status to its
// observer.
type Registry struct {
	mu           sync.Mutex
	provisioners []pluginport.DependencyProvisioner
	flights      flightGroup
	observer     func(manifest.DependencyRequirement, plugin.DependencyStatus)
}

// NewRegistry returns a Registry that routes to the provisioners in order.
func NewRegistry(provisioners ...pluginport.DependencyProvisioner) *Registry {
	return &Registry{
		provisioners: provisioners,
	}
}

// Add appends a provisioner after those already routed to.
func (r *Registry) Add(provisioner pluginport.DependencyProvisioner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.provisioners = append(r.provisioners, provisioner)
}

func (r *Registry) Observe(req manifest.DependencyRequirement, status plugin.DependencyStatus) {
	r.mu.Lock()
	observer := r.observer
	r.mu.Unlock()
	if observer != nil {
		observer(req, status)
	}
}

// SetObserver sets the observer that receives every status reached by any
// provisioning. The observer is called from the goroutine that runs the provisioning.
func (r *Registry) SetObserver(observer func(manifest.DependencyRequirement, plugin.DependencyStatus)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observer = observer
}

// CanProvision reports whether a provisioner handles the requirement.
func (r *Registry) CanProvision(req manifest.DependencyRequirement) bool {
	_, ok := r.route(req)
	return ok
}

func (r *Registry) route(req manifest.DependencyRequirement) (pluginport.DependencyProvisioner, bool) {
	r.mu.Lock()
	provisioners := slices.Clone(r.provisioners)
	r.mu.Unlock()
	for _, provisioner := range provisioners {
		if provisioner.CanProvision(req) {
			return provisioner, true
		}
	}
	return nil, false
}

// Provision makes the dependency ready. Callers that request the same
// requirement while a run is in progress share it and each receives its progress.
func (r *Registry) Provision(
	ctx context.Context, req manifest.DependencyRequirement, onProgress func(plugin.DependencyStatus),
) (plugin.ResolvedDependency, error) {
	provisioner, ok := r.route(req)
	if !ok {
		return plugin.ResolvedDependency{}, fmt.Errorf("%w: %s %s", plugin.ErrNoProvisioner, req.Kind, req.ID)
	}
	return r.flights.do(ctx, getFlightKey(req), onProgress, func(runCtx context.Context, report func(plugin.DependencyStatus)) (plugin.ResolvedDependency, error) {
		return r.run(runCtx, provisioner, req, report)
	})
}

func (r *Registry) run(
	ctx context.Context, provisioner pluginport.DependencyProvisioner, req manifest.DependencyRequirement, report func(plugin.DependencyStatus),
) (plugin.ResolvedDependency, error) {
	note := func(status plugin.DependencyStatus) {
		status.ID = req.ID
		r.mu.Lock()
		observer := r.observer
		r.mu.Unlock()
		if observer != nil {
			observer(req, status)
		}
		report(status)
	}
	note(plugin.DependencyStatus{State: plugin.DependencyStateProvisioning, Message: "starting"})
	resolved, err := provisioner.Provision(ctx, req, func(status plugin.DependencyStatus) {
		if status.State == "" {
			status.State = plugin.DependencyStateProvisioning
		}
		note(status)
	})
	if err != nil {
		note(plugin.DependencyStatus{State: plugin.DependencyStateFailed, Error: err.Error()})
		return plugin.ResolvedDependency{}, err
	}
	note(plugin.DependencyStatus{State: plugin.DependencyStateReady, ProgressPercent: 100, Resolved: &resolved})
	return resolved, nil
}
