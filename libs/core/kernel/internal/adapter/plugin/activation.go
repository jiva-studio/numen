package plugin

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Activation starts the process of a plugin when something asks for it, counts the callers that
// hold it, and says which plugin a caller is.
type Activation struct {
	t                *table
	clock            machineport.Clock
	failures         *Failures
	processes        *Processes
	config           *ConfigPush
	discovery        *Discovery
	refreshListeners func(ctx context.Context, pluginID string)
}

func newActivation(t *table, clock machineport.Clock, failures *Failures, processes *Processes) *Activation {
	return &Activation{t: t, clock: clock, failures: failures, processes: processes}
}

// CheckPluginPermission verifies that the specified plugin holds the required permission.
func (a *Activation) CheckPluginPermission(pluginID, permission string) error {
	a.t.mu.RLock()
	defer a.t.mu.RUnlock()
	sup, ok := a.t.supervisors[pluginID]
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
	a.t.mu.RLock()
	defer a.t.mu.RUnlock()
	var pluginID string
	var found bool
	for id, endpoint := range a.t.endpoints {
		if subtle.ConstantTimeCompare([]byte(endpoint.Token), []byte(token)) == 1 {
			pluginID, found = id, true
		}
	}
	return pluginID, found
}

// GetSupervisor returns the supervisor that tracks a plugin.
func (a *Activation) GetSupervisor(pluginID string) (*lifecycle.Supervisor, bool) {
	a.t.mu.RLock()
	defer a.t.mu.RUnlock()
	sup, ok := a.t.supervisors[pluginID]
	return sup, ok
}

// HasEndpoint reports whether a plugin has the endpoint of a running process registered.
func (a *Activation) HasEndpoint(pluginID string) bool {
	a.t.mu.RLock()
	defer a.t.mu.RUnlock()
	_, ok := a.t.endpoints[pluginID]
	return ok
}

// Acquire increments the active consumer reference count for a plugin.
func (a *Activation) Acquire(pluginID string) {
	sup, ok := a.GetSupervisor(pluginID)
	if ok {
		sup.Acquire(a.clock.Now())
	}
}

// Release decrements the active consumer reference count for a plugin.
func (a *Activation) Release(pluginID string) {
	sup, ok := a.GetSupervisor(pluginID)
	if ok {
		sup.Release(a.clock.Now())
	}
}

func (a *Activation) prepareSupervisorLocked(pluginID string) (manifest.Discovered, *lifecycle.Supervisor, error) {
	if a.t.isClosed {
		return manifest.Discovered{}, nil, errHostClosed
	}
	if a.t.safeMode {
		return manifest.Discovered{}, nil, &plugindomain.UnavailableError{PluginID: pluginID, Reason: plugindomain.Disabled, Err: ErrSafeModeActive}
	}

	p, ok := a.t.plugins[pluginID]
	if !ok {
		return manifest.Discovered{}, nil, &plugindomain.UnavailableError{
			PluginID: pluginID, Reason: plugindomain.NotInstalled, Err: errPluginNotFound,
		}
	}

	a.discovery.registerSupervisorLocked(p)
	sup := a.t.supervisors[pluginID]

	if !a.t.isPluginEnabledLocked(pluginID) {
		return manifest.Discovered{}, nil, &plugindomain.UnavailableError{PluginID: pluginID, Reason: plugindomain.Disabled}
	}
	if _, isBuiltIn := a.t.getBuiltInLocked(pluginID); !isBuiltIn && !p.Manifest.HasBackend() {
		return manifest.Discovered{}, nil, &plugindomain.UnavailableError{PluginID: pluginID, Reason: plugindomain.WithoutBackend}
	}

	return p, sup, nil
}

func (a *Activation) awaitActivation(ctx context.Context, pluginID string, sup *lifecycle.Supervisor) (plugindomain.Endpoint, bool, error) {
	waitCh := sup.ActivatingChan()
	select {
	case <-ctx.Done():
		return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, false, ctx.Err()
	case <-waitCh:
	}

	a.t.mu.Lock()
	defer a.t.mu.Unlock()

	if endpoint, ok := a.acquireRunningLocked(pluginID, sup); ok {
		return endpoint, true, nil
	}
	if err := sup.ActivationError(); err != nil {
		return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, false, err
	}
	return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, false, nil
}

func (a *Activation) acquireRunningLocked(pluginID string, sup *lifecycle.Supervisor) (plugindomain.Endpoint, bool) {
	endpoint := a.t.endpoints[pluginID]
	if isEmptyEndpoint(endpoint) || endpoint.Token == "" || !sup.Acquire(a.clock.Now()) {
		return emptyEndpoint(), false
	}
	return endpoint, true
}

func (a *Activation) settlePluginInState(
	ctx context.Context,
	pluginID string,
	sup *lifecycle.Supervisor,
	state lifecycle.ProcessState,
) (plugindomain.Endpoint, bool, error) {
	switch state {
	case lifecycle.StateDeactivating:
		err := sup.WaitWhileDeactivating(ctx)
		return emptyEndpoint(), err != nil, err
	case lifecycle.StateActivating:
		endpoint, ready, err := a.awaitActivation(ctx, pluginID, sup)
		return endpoint, ready || err != nil, err
	default:
		if err := sup.Teardown(ctx, 0, nil); err != nil && !errors.Is(err, lifecycle.ErrProcessNotRunning) {
			return emptyEndpoint(), true, fmt.Errorf("retire unreachable plugin %s: %w", pluginID, err)
		}
		return emptyEndpoint(), false, nil
	}
}

// EnsureActive ensures the plugin process is running and returns its endpoint.
func (a *Activation) EnsureActive(ctx context.Context, pluginID string) (plugindomain.Endpoint, error) {
	endpoint, err := a.ensureActiveDuringDiscovery(ctx, pluginID)
	a.failures.deliver()
	return endpoint, err
}

func (a *Activation) ensureActiveDuringDiscovery(ctx context.Context, pluginID string) (plugindomain.Endpoint, error) {
	a.t.discovering.RLock()
	defer a.t.discovering.RUnlock()

	for {
		a.t.mu.Lock()
		p, sup, err := a.prepareSupervisorLocked(pluginID)
		if err != nil {
			a.t.mu.Unlock()
			return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, err
		}

		if endpoint, ok := a.acquireRunningLocked(pluginID, sup); ok {
			a.t.mu.Unlock()
			return endpoint, nil
		}

		if state := sup.GetState(); state != lifecycle.StateDormant {
			a.t.mu.Unlock()
			endpoint, isDone, err := a.settlePluginInState(ctx, pluginID, sup, state)
			if isDone {
				return endpoint, err
			}
			continue
		}

		sup.SetActivating()
		a.t.mu.Unlock()

		return a.runActivation(ctx, p, sup)
	}
}

func (a *Activation) runActivation(ctx context.Context, p manifest.Discovered, sup *lifecycle.Supervisor) (plugindomain.Endpoint, error) {
	endpoint, err := a.processes.start(ctx, p, sup)
	if err != nil {
		a.failures.failActivation(ctx, p.Manifest.ID, sup, err)
		return endpoint, err
	}
	a.config.awaitFirstConfigure(p.Manifest.ID)
	a.failures.clear(p.Manifest.ID, sup)
	if err := a.config.configureActivated(ctx, p.Manifest.ID); err != nil {
		return endpoint, err
	}
	return endpoint, nil
}

// RetryPlugin starts a plugin once, as any use of it does, and answers why the start failed where it did.
func (a *Activation) RetryPlugin(ctx context.Context, pluginID string) (*plugindomain.Failure, error) {
	a.t.mu.RLock()
	_, recognized := a.t.plugins[pluginID]
	a.t.mu.RUnlock()
	if !recognized {
		return nil, &plugindomain.UnavailableError{PluginID: pluginID, Reason: plugindomain.NotInstalled, Err: errPluginNotFound}
	}
	a.failures.forgetTargetReport(pluginID)
	defer a.refreshListeners(ctx, pluginID)

	_, err := a.EnsureActive(ctx, pluginID)
	if err == nil {
		a.Release(pluginID)
		return nil, nil
	}
	var unavailable *plugindomain.UnavailableError
	if errors.As(err, &unavailable) || ctx.Err() != nil {
		return nil, err
	}
	a.t.mu.RLock()
	failure := a.failures.getFailureLocked(pluginID)
	a.t.mu.RUnlock()
	return failure, nil
}
