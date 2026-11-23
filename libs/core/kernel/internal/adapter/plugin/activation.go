package plugin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// PluginEnablement reports whether the settings of a plugin leave it enabled.
type PluginEnablement interface {
	// IsPluginEnabled reports whether the settings of the plugin leave it enabled.
	IsPluginEnabled(pluginID string) bool
}

// activationPlugins is what an activation reads of the set of plugins: which one a start is for, how to launch it and the hold
// against a scan of the folders.
type activationPlugins interface {
	getKnownPlugin(pluginID string) (manifest.Discovered, bool)
	getView() (setView, func())
	getLaunch(p manifest.Discovered) supervision.Launch
}

// hostState reports whether the host is shut down or in safe mode.
type hostState interface {
	IsClosed() bool
	IsSafeMode() bool
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
	enablement PluginEnablement
	scan       startHold
	mode       hostState
	sups       *supervision.Supervisors
	processes  *supervision.Processes
	failures   *supervision.Failures
	config     configPusher
	clock      machineport.Clock

	startMu sync.Mutex
	handleStart func(pluginID string)
}

// ActivationDeps are what an Activation starts a plugin from: the set of plugins, the answer whether a plugin is enabled,
// the gate against a scan, the mode of the host, the supervisors and the processes of the plugins, their failures, the push
// of the configuration and the clock.
type ActivationDeps struct {
	Set         activationPlugins
	Enablement  PluginEnablement
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
func (a *Activation) SetOnStart(handleStart func(pluginID string)) {
	a.startMu.Lock()
	defer a.startMu.Unlock()
	a.handleStart = handleStart
}

func (a *Activation) reportStart(pluginID string) {
	a.startMu.Lock()
	handleStart := a.handleStart
	a.startMu.Unlock()
	if handleStart != nil {
		handleStart(pluginID)
	}
}

// ClientLookup finds the HTTP client for calls to the plugin process an endpoint belongs to.
func (a *Activation) ClientLookup() transport.ClientLookup {
	return transport.NewClientLookup(a.sups.Clients())
}

// GetClient returns the client that calls the plugin an endpoint belongs to, served in this process or by a process of
// its own. The bridges and the proxies that call a plugin through the activation take it from here.
func (a *Activation) GetClient(endpoint transport.Endpoint) *http.Client {
	return a.ClientLookup().GetClient(endpoint)
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

// AcquireOwn takes an own lease of the holder on a plugin whose process runs under the token, for work the
// plugin started on its own. It does not cancel a release of the process. ReleaseOwn returns the lease.
func (a *Activation) AcquireOwn(pluginID, token string, holder plugindomain.Holder) bool {
	return a.sups.AcquireOwn(pluginID, token, holder, a.clock.Now())
}

// ReleaseOwn returns an own lease of the holder on the process of the plugin that runs under the token.
func (a *Activation) ReleaseOwn(pluginID, token string, holder plugindomain.Holder) {
	a.sups.Release(pluginID, token, lifecycle.Lease{Kind: lifecycle.OwnLease, Holder: holder}, a.clock.Now())
}

// AcquireRunningForQuestion takes a question lease of the holder on a plugin whose process runs and returns
// the endpoint of the process. A plugin whose process does not run is not started and returns false.
// ReleaseQuestion returns the lease.
func (a *Activation) AcquireRunningForQuestion(pluginID string, holder plugindomain.Holder) (transport.Endpoint, bool) {
	sup, ok := a.sups.Get(pluginID)
	if !ok {
		return supervision.CreateNoEndpoint(), false
	}
	return a.sups.AcquireRunning(pluginID, sup, createQuestionLease(holder), a.clock.Now())
}

// Release returns a use lease of the holder on the process of a plugin that runs under the token, the token of
// the endpoint the lease was taken with. A released process that holds no lease after it is stopped at once.
func (a *Activation) Release(pluginID, token string, holder plugindomain.Holder) {
	a.sups.Release(pluginID, token, useLease(holder), a.clock.Now())
}

// ReleaseQuestion returns a question lease of the holder on the process of a plugin that runs under the token.
// A process that holds no lease after it and was released or never used since it started is stopped at once.
func (a *Activation) ReleaseQuestion(pluginID, token string, holder plugindomain.Holder) {
	a.sups.Release(pluginID, token, createQuestionLease(holder), a.clock.Now())
}

func useLease(holder plugindomain.Holder) lifecycle.Lease {
	return lifecycle.Lease{Kind: lifecycle.UseLease, Holder: holder}
}

func createQuestionLease(holder plugindomain.Holder) lifecycle.Lease {
	return lifecycle.Lease{Kind: lifecycle.QuestionLease, Holder: holder}
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
	v, release := a.set.getView()
	hasBackend := v.hasBackend(p)
	release()
	if !hasBackend {
		return manifest.Discovered{}, nil, &plugindomain.UnavailableError{PluginID: pluginID, Reason: plugindomain.WithoutBackend}
	}

	return p, sup, nil
}

func (a *Activation) awaitActivation(
	ctx context.Context, pluginID string, sup *lifecycle.Supervisor, lease lifecycle.Lease,
) (transport.Endpoint, bool, error) {
	waitCh := sup.GetActivatingChan()
	select {
	case <-ctx.Done():
		return supervision.CreateNoEndpoint(), false, ctx.Err()
	case <-waitCh:
	}

	if endpoint, ok := a.sups.AcquireRunning(pluginID, sup, lease, a.clock.Now()); ok {
		return endpoint, true, nil
	}
	if err := sup.GetActivationError(); err != nil {
		return supervision.CreateNoEndpoint(), false, err
	}
	return supervision.CreateNoEndpoint(), false, nil
}

func (a *Activation) settlePluginInState(
	ctx context.Context,
	pluginID string,
	sup *lifecycle.Supervisor,
	state lifecycle.ProcessState,
	lease lifecycle.Lease,
) (transport.Endpoint, bool, error) {
	switch state {
	case lifecycle.StateDeactivating:
		err := sup.WaitWhileDeactivating(ctx)
		return supervision.CreateNoEndpoint(), err != nil, err
	case lifecycle.StateActivating:
		endpoint, ready, err := a.awaitActivation(ctx, pluginID, sup, lease)
		return endpoint, ready || err != nil, err
	default:
		if err := sup.Teardown(ctx, 0, nil); err != nil && !errors.Is(err, lifecycle.ErrProcessNotRunning) {
			return supervision.CreateNoEndpoint(), true, fmt.Errorf("retire unreachable plugin %s: %w", pluginID, err)
		}
		return supervision.CreateNoEndpoint(), false, nil
	}
}

// EnsureActive starts the plugin where its process does not run, takes a use lease of the holder on it and
// returns its endpoint. It then downloads what the plugin declares and is not on this machine, unless the context comes from
// withoutProvisioning; a failed download releases the lease, is returned and is no failure of the plugin. Release returns the lease.
func (a *Activation) EnsureActive(ctx context.Context, pluginID string, holder plugindomain.Holder) (transport.Endpoint, error) {
	endpoint, err := a.ensureActiveDuringDiscovery(ctx, pluginID, useLease(holder))
	if err != nil || !isProvisioning(ctx) {
		return endpoint, err
	}
	if err := ensureProvisioned(ctx, a.set, a.processes, pluginID); err != nil {
		a.Release(pluginID, endpoint.Token, holder)
		return supervision.CreateNoEndpoint(), err
	}
	return endpoint, nil
}

type withoutProvisioningKey struct{}

func withoutProvisioning(ctx context.Context) context.Context {
	return context.WithValue(ctx, withoutProvisioningKey{}, struct{}{})
}

func isProvisioning(ctx context.Context) bool {
	return ctx.Value(withoutProvisioningKey{}) == nil
}

func ensureProvisioned(ctx context.Context, set activationPlugins, processes *supervision.Processes, pluginID string) error {
	p, ok := set.getKnownPlugin(pluginID)
	if !ok {
		return nil
	}
	requirements := set.getLaunch(p).Requirements
	if len(requirements) == 0 {
		return nil
	}
	return processes.EnsureProvisioned(ctx, requirements)
}

// EnsureActiveForQuestion starts the plugin where its process does not run, takes a question lease of the
// holder on it and returns its endpoint. ReleaseQuestion returns the lease.
func (a *Activation) EnsureActiveForQuestion(
	ctx context.Context, pluginID string, holder plugindomain.Holder,
) (transport.Endpoint, error) {
	return a.ensureActiveDuringDiscovery(ctx, pluginID, createQuestionLease(holder))
}

func (a *Activation) ensureActiveDuringDiscovery(
	ctx context.Context, pluginID string, lease lifecycle.Lease,
) (transport.Endpoint, error) {
	a.scan.holdStart()
	defer a.scan.releaseStart()

	for {
		p, sup, err := a.prepareSupervisor(pluginID)
		if err != nil {
			return supervision.CreateNoEndpoint(), err
		}

		claim := a.sups.Begin(pluginID, sup, lease, a.clock.Now())
		if claim.Running {
			return claim.Endpoint, nil
		}
		if !claim.Started {
			endpoint, isDone, err := a.settlePluginInState(ctx, pluginID, sup, claim.State, lease)
			if isDone {
				return endpoint, err
			}
			continue
		}
		if a.mode.IsClosed() {
			sup.SetDormant(errHostClosed)
			return supervision.CreateNoEndpoint(), errHostClosed
		}

		return a.runActivation(ctx, p, sup, lease)
	}
}

func (a *Activation) runActivation(
	ctx context.Context, p manifest.Discovered, sup *lifecycle.Supervisor, lease lifecycle.Lease,
) (transport.Endpoint, error) {
	launch := a.set.getLaunch(p)
	launch.Lease = lease
	endpoint, err := a.processes.Start(ctx, launch, sup)
	if err != nil {
		a.failures.FailActivation(ctx, p.Manifest.ID, p.Manifest.Name, sup, err)
		return endpoint, err
	}
	a.config.awaitFirstConfigure(p.Manifest.ID)
	a.failures.Clear(p.Manifest.ID, sup)
	if err := a.config.configureActivated(ctx, p.Manifest.ID); err != nil {
		a.sups.Release(p.Manifest.ID, endpoint.Token, lease, a.clock.Now())
		return supervision.CreateNoEndpoint(), err
	}
	a.reportStart(p.Manifest.ID)
	return endpoint, nil
}
