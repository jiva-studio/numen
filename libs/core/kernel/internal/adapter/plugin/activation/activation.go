// Package activation starts the process of a plugin when something asks for it, and starts it once again on request.
package activation

import (
	"context"
	"errors"
	"fmt"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/hostmode"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/pluginset"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Activation starts the process of a plugin when something asks for it, counts the callers that
// hold it, and says which plugin a caller is.
type Activation struct {
	set       *pluginset.Set
	mode      modeState
	sups      *supervision.Supervisors
	processes *supervision.Processes
	failures  *supervision.Failures
	config    configurer
	clock     machineport.Clock
}

// modeState is what the activation reads of the mode of the host.
type modeState interface {
	IsClosed() bool
	IsSafeMode() bool
}

// configurer gives a plugin its configuration around the start of its process.
type configurer interface {
	AwaitFirstConfigure(pluginID string)
	ConfigureActivated(ctx context.Context, pluginID string) error
}

// Deps are what an activation reads: the set of plugins, the mode of the host, the supervisors, the processes, the failures,
// the configuration push and the clock.
type Deps struct {
	Set       *pluginset.Set
	Mode      modeState
	Sups      *supervision.Supervisors
	Processes *supervision.Processes
	Failures  *supervision.Failures
	Config    configurer
	Clock     machineport.Clock
}

// New returns the activation over deps.
func New(deps Deps) *Activation {
	return &Activation{
		set: deps.Set, mode: deps.Mode, sups: deps.Sups, processes: deps.Processes, failures: deps.Failures,
		config: deps.Config, clock: deps.Clock,
	}
}

// ClientLookup finds the HTTP client for calls to the plugin process an endpoint belongs to.
func (a *Activation) ClientLookup() transport.ClientLookup {
	return transport.NewClientLookup(a.sups.Clients())
}

// CheckPluginPermission verifies that the specified plugin holds the required permission.
func (a *Activation) CheckPluginPermission(pluginID, permission string) error {
	sup, ok := a.sups.Get(pluginID)
	if !ok {
		return pluginset.ErrNotFound
	}
	return sup.Permissions.Check(permission)
}

// GetPluginIDByToken returns the id of the running plugin the token was given
// to. An empty token names none.
func (a *Activation) GetPluginIDByToken(token string) (string, bool) {
	if token == "" {
		return "", false
	}
	return a.sups.FindByToken(token)
}

// GetSupervisor returns the supervisor that tracks a plugin.
func (a *Activation) GetSupervisor(pluginID string) (*lifecycle.Supervisor, bool) {
	return a.sups.Get(pluginID)
}

// HasEndpoint reports whether a plugin has the endpoint of a running process registered.
func (a *Activation) HasEndpoint(pluginID string) bool {
	_, ok := a.sups.GetEndpoint(pluginID)
	return ok
}

// Acquire increments the active consumer reference count for a plugin.
func (a *Activation) Acquire(pluginID string) {
	sup, ok := a.sups.Get(pluginID)
	if ok {
		sup.Acquire(a.clock.Now())
	}
}

// Release decrements the active consumer reference count for a plugin.
func (a *Activation) Release(pluginID string) {
	sup, ok := a.sups.Get(pluginID)
	if ok {
		sup.Release(a.clock.Now())
	}
}

func (a *Activation) prepareSupervisor(pluginID string) (manifest.Discovered, *lifecycle.Supervisor, error) {
	if a.mode.IsClosed() {
		return manifest.Discovered{}, nil, hostmode.ErrHostClosed
	}
	if a.mode.IsSafeMode() {
		return manifest.Discovered{}, nil, &plugindomain.UnavailableError{PluginID: pluginID, Reason: plugindomain.Disabled, Err: hostmode.ErrSafeModeActive}
	}

	p, ok := a.set.GetKnown(pluginID)
	if !ok {
		return manifest.Discovered{}, nil, &plugindomain.UnavailableError{
			PluginID: pluginID, Reason: plugindomain.NotInstalled, Err: pluginset.ErrNotFound,
		}
	}

	sup := a.sups.Register(p)

	v, release := a.set.View()
	isEnabled, hasBackend := v.IsEnabled(pluginID), v.HasBackend(p)
	release()
	if !isEnabled {
		return manifest.Discovered{}, nil, &plugindomain.UnavailableError{PluginID: pluginID, Reason: plugindomain.Disabled}
	}
	if !hasBackend {
		return manifest.Discovered{}, nil, &plugindomain.UnavailableError{PluginID: pluginID, Reason: plugindomain.WithoutBackend}
	}

	return p, sup, nil
}

func (a *Activation) awaitActivation(ctx context.Context, pluginID string, sup *lifecycle.Supervisor) (transport.Endpoint, bool, error) {
	waitCh := sup.ActivatingChan()
	select {
	case <-ctx.Done():
		return supervision.NoEndpoint(), false, ctx.Err()
	case <-waitCh:
	}

	if endpoint, ok := a.sups.AcquireRunning(pluginID, sup, a.clock.Now()); ok {
		return endpoint, true, nil
	}
	if err := sup.ActivationError(); err != nil {
		return supervision.NoEndpoint(), false, err
	}
	return supervision.NoEndpoint(), false, nil
}

func (a *Activation) settlePluginInState(
	ctx context.Context,
	pluginID string,
	sup *lifecycle.Supervisor,
	state lifecycle.ProcessState,
) (transport.Endpoint, bool, error) {
	switch state {
	case lifecycle.StateDeactivating:
		err := sup.WaitWhileDeactivating(ctx)
		return supervision.NoEndpoint(), err != nil, err
	case lifecycle.StateActivating:
		endpoint, ready, err := a.awaitActivation(ctx, pluginID, sup)
		return endpoint, ready || err != nil, err
	default:
		if err := sup.Teardown(ctx, 0, nil); err != nil && !errors.Is(err, lifecycle.ErrProcessNotRunning) {
			return supervision.NoEndpoint(), true, fmt.Errorf("retire unreachable plugin %s: %w", pluginID, err)
		}
		return supervision.NoEndpoint(), false, nil
	}
}

// EnsureActive ensures the plugin process is running and returns its endpoint.
func (a *Activation) EnsureActive(ctx context.Context, pluginID string) (transport.Endpoint, error) {
	endpoint, err := a.ensureActiveDuringDiscovery(ctx, pluginID)
	a.failures.Deliver()
	return endpoint, err
}

func (a *Activation) ensureActiveDuringDiscovery(ctx context.Context, pluginID string) (transport.Endpoint, error) {
	scan := a.set.Scan()
	scan.BeginActivation()
	defer scan.EndActivation()

	for {
		p, sup, err := a.prepareSupervisor(pluginID)
		if err != nil {
			return supervision.NoEndpoint(), err
		}

		claim := a.sups.Begin(pluginID, sup, a.clock.Now())
		if claim.Running {
			return claim.Endpoint, nil
		}
		if !claim.Started {
			endpoint, isDone, err := a.settlePluginInState(ctx, pluginID, sup, claim.State)
			if isDone {
				return endpoint, err
			}
			continue
		}
		if a.mode.IsClosed() {
			sup.SetDormant(hostmode.ErrHostClosed)
			return supervision.NoEndpoint(), hostmode.ErrHostClosed
		}

		return a.RunActivation(ctx, p, sup)
	}
}

// RunActivation starts the process of a plugin whose supervisor the caller claimed for activating, and gives the plugin its
// configuration.
func (a *Activation) RunActivation(ctx context.Context, p manifest.Discovered, sup *lifecycle.Supervisor) (transport.Endpoint, error) {
	endpoint, err := a.processes.Start(ctx, a.set.GetLaunch(p), sup)
	if err != nil {
		a.failures.FailActivation(ctx, p.Manifest.ID, p.Manifest.Name, sup, err)
		return endpoint, err
	}
	a.config.AwaitFirstConfigure(p.Manifest.ID)
	a.failures.Clear(p.Manifest.ID, sup)
	if err := a.config.ConfigureActivated(ctx, p.Manifest.ID); err != nil {
		return endpoint, err
	}
	return endpoint, nil
}
