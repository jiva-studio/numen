package plugin

import (
	"context"
	"errors"
	"fmt"
	"sync"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// activationPlugins is what an activation reads of the set of plugins: which one a start is for, how to launch it and the hold
// against a scan of the folders.
type activationPlugins interface {
	getKnownPlugin(pluginID string) (manifest.Discovered, bool)
	view() (setView, func())
	getLaunch(p manifest.Discovered) supervision.Launch
}

// hostState reports whether the host is shut down or in safe mode.
type hostState interface {
	IsClosed() bool
	IsSafeMode() bool
}

// enablementChecker reports whether the settings of a plugin leave it enabled.
type enablementChecker interface {
	IsPluginEnabled(pluginID string) bool
}

// startHold holds the set of plugins against a scan of the folders while a plugin starts.
type startHold interface {
	holdStart()
	releaseStart()
}

// configPusher gives a plugin its configuration around the start of its process.
type configPusher interface {
	awaitFirstConfigure(pluginID string)
	configureActivated(ctx context.Context, pluginID string) error
}

// Activation starts the process of a plugin when something asks for it, counts the callers that
// hold it, and says which plugin a caller is.
type Activation struct {
	set        activationPlugins
	enablement enablementChecker
	scan       startHold
	mode       hostState
	sups       *supervision.Supervisors
	processes  *supervision.Processes
	failures   *supervision.Failures
	config     configPusher
	clock      machineport.Clock

	startMu sync.Mutex
	onStart func(pluginID string)
}

// ActivationDeps are what an Activation starts a plugin from: the set of plugins, the answer whether a plugin is enabled,
// the gate against a scan, the mode of the host, the supervisors and the processes of the plugins, their failures, the push
// of the configuration and the clock.
type ActivationDeps struct {
	Set         activationPlugins
	Enablement  enablementChecker
	Scan        startHold
	Mode        hostState
	Supervisors *supervision.Supervisors
	Processes   *supervision.Processes
	Failures    *supervision.Failures
	Config      configPusher
	Clock       machineport.Clock
}

// NewActivation creates the activation of the plugins with none running.
func NewActivation(d ActivationDeps) *Activation {
	return &Activation{
		set: d.Set, enablement: d.Enablement, scan: d.Scan, mode: d.Mode, sups: d.Supervisors, processes: d.Processes,
		failures: d.Failures, config: d.Config, clock: d.Clock,
	}
}

// SetOnStart sets what is called after the process of a plugin started and took its configuration; it returns at
// once and does its work on a goroutine of its own. It replaces what was set before.
func (a *Activation) SetOnStart(onStart func(pluginID string)) {
	a.startMu.Lock()
	defer a.startMu.Unlock()
	a.onStart = onStart
}

func (a *Activation) reportStart(pluginID string) {
	a.startMu.Lock()
	onStart := a.onStart
	a.startMu.Unlock()
	if onStart != nil {
		onStart(pluginID)
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
		return errPluginNotFound
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
		a.sups.NotifyChange()
	}
}

// Release decrements the active consumer reference count for a plugin.
func (a *Activation) Release(pluginID string) {
	sup, ok := a.sups.Get(pluginID)
	if ok {
		sup.Release(a.clock.Now())
		a.sups.NotifyChange()
	}
}

func (a *Activation) prepareSupervisor(pluginID string) (manifest.Discovered, *lifecycle.Supervisor, error) {
	if a.mode.IsClosed() {
		return manifest.Discovered{}, nil, errHostClosed
	}
	if a.mode.IsSafeMode() {
		return manifest.Discovered{}, nil, &plugindomain.UnavailableError{PluginID: pluginID, Reason: plugindomain.Disabled, Err: ErrSafeModeActive}
	}

	p, ok := a.set.getKnownPlugin(pluginID)
	if !ok {
		return manifest.Discovered{}, nil, &plugindomain.UnavailableError{
			PluginID: pluginID, Reason: plugindomain.NotInstalled, Err: errPluginNotFound,
		}
	}

	sup := a.sups.Register(p)

	if !a.enablement.IsPluginEnabled(pluginID) {
		return manifest.Discovered{}, nil, &plugindomain.UnavailableError{PluginID: pluginID, Reason: plugindomain.Disabled}
	}
	v, release := a.set.view()
	hasBackend := v.hasBackend(p)
	release()
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
	return a.ensureActiveDuringDiscovery(ctx, pluginID)
}

func (a *Activation) ensureActiveDuringDiscovery(ctx context.Context, pluginID string) (transport.Endpoint, error) {
	a.scan.holdStart()
	defer a.scan.releaseStart()

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
			sup.SetDormant(errHostClosed)
			return supervision.NoEndpoint(), errHostClosed
		}

		return a.runActivation(ctx, p, sup)
	}
}

func (a *Activation) runActivation(ctx context.Context, p manifest.Discovered, sup *lifecycle.Supervisor) (transport.Endpoint, error) {
	endpoint, err := a.processes.Start(ctx, a.set.getLaunch(p), sup)
	if err != nil {
		a.failures.FailActivation(ctx, p.Manifest.ID, p.Manifest.Name, sup, err)
		return endpoint, err
	}
	a.config.awaitFirstConfigure(p.Manifest.ID)
	a.failures.Clear(p.Manifest.ID, sup)
	if err := a.config.configureActivated(ctx, p.Manifest.ID); err != nil {
		return endpoint, err
	}
	a.reportStart(p.Manifest.ID)
	return endpoint, nil
}
