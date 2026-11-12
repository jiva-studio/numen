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
// shares one run between identical requests and reports the status of every
// registered requirement.
type Registry struct {
	mu           sync.Mutex
	provisioners []pluginport.DependencyProvisioner
	requirements map[string]manifest.DependencyRequirement
	order        []string
	last         map[string]plugin.DependencyStatus
	flights      flightGroup
	observer     func(manifest.DependencyRequirement, plugin.DependencyStatus)
}

// NewRegistry returns a Registry that routes to the provisioners in order.
func NewRegistry(provisioners ...pluginport.DependencyProvisioner) *Registry {
	return &Registry{
		provisioners: provisioners,
		requirements: make(map[string]manifest.DependencyRequirement),
		last:         make(map[string]plugin.DependencyStatus),
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

// Register records the requirement under its id.
func (r *Registry) Register(req manifest.DependencyRequirement) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.requirements[req.ID]; !ok {
		r.order = append(r.order, req.ID)
	}
	r.requirements[req.ID] = req
}

func (r *Registry) Lookup(id string) (manifest.DependencyRequirement, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	req, ok := r.requirements[id]
	return req, ok
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
	r.Register(req)
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
		r.last[req.ID] = status
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

// Status returns the status of the dependency without installing it.
func (r *Registry) Status(ctx context.Context, req manifest.DependencyRequirement) (plugin.DependencyStatus, error) {
	provisioner, ok := r.route(req)
	if !ok {
		return plugin.DependencyStatus{}, fmt.Errorf("%w: %s %s", plugin.ErrNoProvisioner, req.Kind, req.ID)
	}
	if r.flights.isRunning(getFlightKey(req)) {
		r.mu.Lock()
		last := r.last[req.ID]
		r.mu.Unlock()
		last.ID = req.ID
		last.State = plugin.DependencyStateProvisioning
		return last, nil
	}
	status, err := provisioner.Status(ctx, req)
	if err != nil {
		return plugin.DependencyStatus{}, err
	}
	status.ID = req.ID
	if status.State == plugin.DependencyStateMissing {
		r.mu.Lock()
		last, hasLast := r.last[req.ID]
		r.mu.Unlock()
		if hasLast && last.State == plugin.DependencyStateFailed {
			return last, nil
		}
	}
	return status, nil
}

// GetStatus returns the status of the registered dependency with the id. An
// unregistered id has the missing state.
func (r *Registry) GetStatus(ctx context.Context, id string) (plugin.DependencyStatus, error) {
	req, ok := r.Lookup(id)
	if !ok {
		return plugin.DependencyStatus{ID: id, State: plugin.DependencyStateMissing}, nil
	}
	return r.Status(ctx, req)
}

// List returns the status of every registered dependency, in registration order.
func (r *Registry) List(ctx context.Context) ([]plugin.DependencyStatus, error) {
	r.mu.Lock()
	ids := slices.Clone(r.order)
	r.mu.Unlock()
	out := make([]plugin.DependencyStatus, 0, len(ids))
	for _, id := range ids {
		status, err := r.GetStatus(ctx, id)
		if err != nil {
			status = plugin.DependencyStatus{ID: id, State: plugin.DependencyStateFailed, Error: err.Error()}
		}
		out = append(out, status)
	}
	return out, nil
}
