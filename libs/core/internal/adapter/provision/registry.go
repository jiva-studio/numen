// Package provision makes the dependencies a plugin declares ready on this
// machine: a registry that routes a requirement to the provisioner that handles
// it, and the primitives those provisioners share.
package provision

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// Registry routes a requirement to the first provisioner that handles it,
// shares one run between identical requests and answers for every requirement
// it has been told of.
type Registry struct {
	mu           sync.Mutex
	provisioners []pluginport.DependencyProvisioner
	requirements map[string]pluginport.DependencyRequirement
	order        []string
	last         map[string]pluginport.DependencyStatus
	flights      flightGroup
	observer     func(pluginport.DependencyRequirement, pluginport.DependencyStatus)
}

// NewRegistry returns a Registry that routes to the provisioners in order.
func NewRegistry(provisioners ...pluginport.DependencyProvisioner) *Registry {
	return &Registry{
		provisioners: provisioners,
		requirements: make(map[string]pluginport.DependencyRequirement),
		last:         make(map[string]pluginport.DependencyStatus),
	}
}

// Add appends a provisioner after those already routed to.
func (r *Registry) Add(provisioner pluginport.DependencyProvisioner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.provisioners = append(r.provisioners, provisioner)
}

// Observe tells the observer a status, as a provisioning does.
func (r *Registry) Observe(req pluginport.DependencyRequirement, status pluginport.DependencyStatus) {
	r.mu.Lock()
	observer := r.observer
	r.mu.Unlock()
	if observer != nil {
		observer(req, status)
	}
}

// SetObserver sets what is told, for every provisioning, each status it
// reaches. The observer is called from the goroutine that runs the provisioning.
func (r *Registry) SetObserver(observer func(pluginport.DependencyRequirement, pluginport.DependencyStatus)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observer = observer
}

// CanProvision reports whether a provisioner handles the requirement.
func (r *Registry) CanProvision(req pluginport.DependencyRequirement) bool {
	_, ok := r.route(req)
	return ok
}

func (r *Registry) route(req pluginport.DependencyRequirement) (pluginport.DependencyProvisioner, bool) {
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

// Register records the requirement so that it is answered for by its id.
func (r *Registry) Register(req pluginport.DependencyRequirement) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.requirements[req.ID]; !ok {
		r.order = append(r.order, req.ID)
	}
	r.requirements[req.ID] = req
}

// Lookup returns the registered requirement with the id.
func (r *Registry) Lookup(id string) (pluginport.DependencyRequirement, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	req, ok := r.requirements[id]
	return req, ok
}

// Provision makes the dependency ready. Callers that ask for the same
// requirement while one run is under way share it and each is told its progress.
func (r *Registry) Provision(
	ctx context.Context, req pluginport.DependencyRequirement, onProgress func(pluginport.DependencyStatus),
) (pluginport.ResolvedDependency, error) {
	provisioner, ok := r.route(req)
	if !ok {
		return pluginport.ResolvedDependency{}, fmt.Errorf("%w: %s %s", pluginport.ErrNoProvisioner, req.Kind, req.ID)
	}
	r.Register(req)
	return r.flights.do(ctx, getFlightKey(req), onProgress, func(runCtx context.Context, report func(pluginport.DependencyStatus)) (pluginport.ResolvedDependency, error) {
		return r.run(runCtx, provisioner, req, report)
	})
}

func (r *Registry) run(
	ctx context.Context, provisioner pluginport.DependencyProvisioner, req pluginport.DependencyRequirement, report func(pluginport.DependencyStatus),
) (pluginport.ResolvedDependency, error) {
	note := func(status pluginport.DependencyStatus) {
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
	note(pluginport.DependencyStatus{State: pluginport.DependencyStateProvisioning, Message: "starting"})
	resolved, err := provisioner.Provision(ctx, req, func(status pluginport.DependencyStatus) {
		if status.State == "" {
			status.State = pluginport.DependencyStateProvisioning
		}
		note(status)
	})
	if err != nil {
		note(pluginport.DependencyStatus{State: pluginport.DependencyStateFailed, Error: err.Error()})
		return pluginport.ResolvedDependency{}, err
	}
	note(pluginport.DependencyStatus{State: pluginport.DependencyStateReady, ProgressPercent: 100, Resolved: &resolved})
	return resolved, nil
}

// Status answers where the dependency stands without installing it.
func (r *Registry) Status(ctx context.Context, req pluginport.DependencyRequirement) (pluginport.DependencyStatus, error) {
	provisioner, ok := r.route(req)
	if !ok {
		return pluginport.DependencyStatus{}, fmt.Errorf("%w: %s %s", pluginport.ErrNoProvisioner, req.Kind, req.ID)
	}
	if r.flights.isRunning(getFlightKey(req)) {
		r.mu.Lock()
		last := r.last[req.ID]
		r.mu.Unlock()
		last.ID = req.ID
		last.State = pluginport.DependencyStateProvisioning
		return last, nil
	}
	status, err := provisioner.Status(ctx, req)
	if err != nil {
		return pluginport.DependencyStatus{}, err
	}
	status.ID = req.ID
	if status.State == pluginport.DependencyStateMissing {
		r.mu.Lock()
		last, hasLast := r.last[req.ID]
		r.mu.Unlock()
		if hasLast && last.State == pluginport.DependencyStateFailed {
			return last, nil
		}
	}
	return status, nil
}

// GetStatus answers where the registered dependency with the id stands. An id
// nobody registered is missing.
func (r *Registry) GetStatus(ctx context.Context, id string) (pluginport.DependencyStatus, error) {
	req, ok := r.Lookup(id)
	if !ok {
		return pluginport.DependencyStatus{ID: id, State: pluginport.DependencyStateMissing}, nil
	}
	return r.Status(ctx, req)
}

// List answers where every registered dependency stands, in the order they were registered.
func (r *Registry) List(ctx context.Context) ([]pluginport.DependencyStatus, error) {
	r.mu.Lock()
	ids := slices.Clone(r.order)
	r.mu.Unlock()
	out := make([]pluginport.DependencyStatus, 0, len(ids))
	for _, id := range ids {
		status, err := r.GetStatus(ctx, id)
		if err != nil {
			status = pluginport.DependencyStatus{ID: id, State: pluginport.DependencyStateFailed, Error: err.Error()}
		}
		out = append(out, status)
	}
	return out, nil
}
