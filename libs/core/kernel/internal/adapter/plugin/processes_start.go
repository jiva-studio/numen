package plugin

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/process"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// ErrRuntimeUnavailable indicates that a plugin declares a dependency the host cannot provision.
var ErrRuntimeUnavailable = errors.New("runtime unavailable")

func (p *Processes) getRuntimeVars(ctx context.Context, requirements []manifest.DependencyRequirement) ([]string, error) {
	var vars []string
	for _, req := range requirements {
		catalog := p.services.GetDependencies()
		if catalog == nil || !catalog.CanProvision(req) {
			return nil, fmt.Errorf("%w: nothing provisions %s %s", ErrRuntimeUnavailable, req.Kind, req.ID)
		}
		resolved, err := catalog.Provision(ctx, req, nil)
		if err != nil {
			return nil, fmt.Errorf("provision %s: %w", req.ID, err)
		}
		for _, name := range slices.Sorted(maps.Keys(resolved.EnvVars)) {
			if !isDependencyEnvName(name) {
				return nil, fmt.Errorf("provision %s: %w: the variable name %q", req.ID, ErrRuntimeUnavailable, name)
			}
			vars = append(vars, name+"="+resolved.EnvVars[name])
		}
	}
	return vars, nil
}

var dependencyEnvName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func isDependencyEnvName(name string) bool {
	if !dependencyEnvName.MatchString(name) {
		return false
	}
	return name != "NUMEN_VAULT_PATH" && !strings.HasPrefix(name, "NUMEN_PLUGIN_") && !strings.HasPrefix(name, "NUMEN_HOST_")
}

func (p *Processes) getLaunchVars(ctx context.Context, pluginID string, requirements []manifest.DependencyRequirement, processPort int, token string) ([]string, error) {
	runtimeVars, err := p.getRuntimeVars(ctx, requirements)
	if err != nil {
		return nil, fmt.Errorf("launch plugin %s: %w", pluginID, err)
	}
	return p.getEnvironmentVars(ctx, pluginID, processPort, token, runtimeVars), nil
}

func (p *Processes) getEnvironmentVars(ctx context.Context, pluginID string, processPort int, token string, runtimeVars []string) []string {
	hostPort, err := p.services.OpenHostEndpoint(ctx)
	if err != nil {
		p.errs.report(fmt.Errorf("open host endpoint for plugin %s: %w", pluginID, err))
		hostPort = 0
	}
	storageVars, err := p.storage.GetVars(pluginID)
	if err != nil {
		p.errs.report(err)
	}
	p.t.mu.RLock()
	defer p.t.mu.RUnlock()
	return process.GetPluginVars(pluginID, process.LaunchEnvironment{
		VaultPath: p.t.vaultPath, Port: processPort, HostPort: hostPort, Token: token, RuntimeVars: runtimeVars,
		StorageVars: storageVars,
	})
}

func (p *Processes) startBuiltIn(ctx context.Context, plug manifest.Discovered, builtIn builtInPlugin, sup *lifecycle.Supervisor) (plugindomain.Endpoint, error) {
	token, err := transport.GenerateAuthToken()
	if err != nil {
		return emptyEndpoint(), fmt.Errorf("generate token: %w", err)
	}
	runtimeVars, err := p.exportRuntimeVars(ctx, p.getRequirements(plug))
	if err != nil {
		return emptyEndpoint(), fmt.Errorf("launch plugin %s: %w", plug.Manifest.ID, err)
	}
	vars := make(map[string]string)
	launchVars := p.getEnvironmentVars(ctx, plug.Manifest.ID, 0, token, runtimeVars)
	for _, entry := range process.SanitizeEnvironment(os.Environ(), launchVars, plug.Manifest.GetRuntimeIDs()) {
		name, value, _ := strings.Cut(entry, "=")
		vars[name] = value
	}
	handler, err := builtIn.serve(vars)
	if err != nil {
		return emptyEndpoint(), fmt.Errorf("launch plugin %s: %w", plug.Manifest.ID, err)
	}

	endpoint := plugindomain.Endpoint{
		Host:  "inprocess",
		Port:  0,
		Token: token,
	}
	p.t.clients.Add(token, transport.NewInProcessTransport(handler))
	if err := transport.CheckPluginProtocol(ctx, p.t.clients.Get(endpoint), plug.Manifest, endpoint); err != nil {
		p.t.clients.Remove(token)
		p.shutdownBuiltIn(ctx, endpoint)
		return emptyEndpoint(), err
	}

	now := p.clock.Now()
	p.t.mu.Lock()
	if sup.GetState() != lifecycle.StateActivating {
		p.t.mu.Unlock()
		p.shutdownBuiltIn(ctx, endpoint)
		return emptyEndpoint(), lifecycle.ErrActivationAborted
	}
	p.t.endpoints[plug.Manifest.ID] = endpoint
	sup.ActivateInProcess(now)
	p.t.mu.Unlock()
	return endpoint, nil
}

func (p *Processes) shutdownBuiltIn(ctx context.Context, endpoint plugindomain.Endpoint) {
	if shutdown := createRPCShutdown(endpoint, p.t.clients.Get(endpoint), time.Second); shutdown != nil {
		if err := shutdown(context.WithoutCancel(ctx)); err != nil {
			p.errs.report(fmt.Errorf("shut down the handler of built-in plugin that was not activated: %w", err))
		}
	}
}

func (p *Processes) exportRuntimeVars(ctx context.Context, requirements []manifest.DependencyRequirement) ([]string, error) {
	vars, err := p.getRuntimeVars(ctx, requirements)
	if err != nil {
		return nil, err
	}
	for _, entry := range vars {
		name, value, _ := strings.Cut(entry, "=")
		if err := os.Setenv(name, value); err != nil {
			return nil, err
		}
	}
	return vars, nil
}
