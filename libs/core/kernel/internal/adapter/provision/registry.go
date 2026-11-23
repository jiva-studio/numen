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
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Registry routes a requirement to the first provisioner that handles it,
// shares one run between identical requests and reports each status to its
// observer.
type Registry struct {
	mu           sync.Mutex
	provisioners []DependencyProvisioner
	flights      flightGroup
	observe     func(manifest.DependencyRequirement, plugin.DependencyStatus)
}

// NewObservedRegistry returns a Registry that sends every status reached by any provisioning to observer.
// The observer is called from the goroutine that runs the provisioning.
func NewObservedRegistry(observe func(manifest.DependencyRequirement, plugin.DependencyStatus), provisioners ...DependencyProvisioner) *Registry {
	return &Registry{provisioners: provisioners, observe: observe}
}

// Add appends a provisioner after those already routed to.
func (r *Registry) Add(provisioner DependencyProvisioner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.provisioners = append(r.provisioners, provisioner)
}

func (r *Registry) Observe(req manifest.DependencyRequirement, status plugin.DependencyStatus) {
	if r.observe != nil {
		r.observe(req, status)
	}
}

// CanProvision reports whether a provisioner handles the requirement.
func (r *Registry) CanProvision(req manifest.DependencyRequirement) bool {
	_, ok := r.route(req)
	return ok
}

func (r *Registry) route(req manifest.DependencyRequirement) (DependencyProvisioner, bool) {
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

// Provision makes the dependency ready; a dependency that is ready already is returned without a run. Callers that request the same
// requirement while a run is in progress share it and each receives its progress.
func (r *Registry) Provision(
	ctx context.Context, req manifest.DependencyRequirement, onProgress func(plugin.DependencyStatus),
) (plugin.ResolvedDependency, error) {
	provisioner, ok := r.route(req)
	if !ok {
		return plugin.ResolvedDependency{}, fmt.Errorf("%w: %s %s", plugin.ErrNoProvisioner, req.Kind, req.ID)
	}
	if resolved, state, err := provisioner.Resolve(req); err == nil && state == plugin.DependencyStateReady {
		return resolved, nil
	}
	return r.flights.do(ctx, req.GetIdentity(), onProgress, func(runCtx context.Context, report func(plugin.DependencyStatus)) (plugin.ResolvedDependency, error) {
		return r.run(runCtx, provisioner, req, report)
	})
}

func (r *Registry) run(
	ctx context.Context, provisioner DependencyProvisioner, req manifest.DependencyRequirement, report func(plugin.DependencyStatus),
) (plugin.ResolvedDependency, error) {
	note := func(status plugin.DependencyStatus) {
		status.ID = req.ID
		if r.observe != nil {
			r.observe(req, status)
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
	if ctx.Err() != nil {
		note(plugin.DependencyStatus{State: plugin.DependencyStateMissing, Message: "canceled"})
		return plugin.ResolvedDependency{}, ctx.Err()
	}
	if err != nil {
		note(plugin.DependencyStatus{State: plugin.DependencyStateFailed, Error: err.Error()})
		return plugin.ResolvedDependency{}, err
	}
	note(plugin.DependencyStatus{State: plugin.DependencyStateReady, ProgressPercent: 100, Resolved: &resolved})
	return resolved, nil
}

// Cancel stops the run of the requirement and reports whether one was in progress. Every caller waiting on
// the run receives context.Canceled, the observer receives the state missing, and what the run had downloaded
// stays for the next run to continue from.
func (r *Registry) Cancel(req manifest.DependencyRequirement) bool {
	return r.flights.cancel(req.GetIdentity())
}

// CancelAll stops every run in progress as Cancel does.
func (r *Registry) CancelAll() {
	r.flights.cancelAll()
}

// Resolve asks the provisioner that handles the requirement where the dependency is or will be. It does not
// join a run in progress and waits for none.
func (r *Registry) Resolve(req manifest.DependencyRequirement) (plugin.ResolvedDependency, plugin.DependencyState, error) {
	provisioner, ok := r.route(req)
	if !ok {
		return plugin.ResolvedDependency{}, "", fmt.Errorf("%w: %s %s", plugin.ErrNoProvisioner, req.Kind, req.ID)
	}
	return provisioner.Resolve(req)
}
