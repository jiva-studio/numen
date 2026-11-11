package supervision

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

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/process"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// ErrRuntimeUnavailable indicates that a plugin declares a dependency the host cannot provision.
var ErrRuntimeUnavailable = errors.New("runtime unavailable")

// GetRuntimeVars provisions the dependencies the plugin declares and returns the variables they set.
func (p *Processes) GetRuntimeVars(ctx context.Context, requirements []manifest.DependencyRequirement) ([]string, error) {
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

// GetLaunchVars returns the variables a process of the plugin is launched with.
func (p *Processes) GetLaunchVars(
	ctx context.Context, pluginID string, requirements []manifest.DependencyRequirement, vaultPath string, processPort int, token string,
) ([]string, error) {
	runtimeVars, err := p.GetRuntimeVars(ctx, requirements)
	if err != nil {
		return nil, fmt.Errorf("launch plugin %s: %w", pluginID, err)
	}
	return p.getEnvironmentVars(ctx, pluginID, vaultPath, processPort, token, runtimeVars), nil
}

func (p *Processes) getEnvironmentVars(ctx context.Context, pluginID, vaultPath string, processPort int, token string, runtimeVars []string) []string {
	hostPort, err := p.services.OpenHostEndpoint(ctx)
	if err != nil {
		p.report(fmt.Errorf("open host endpoint for plugin %s: %w", pluginID, err))
		hostPort = 0
	}
	storageVars, err := p.storage.GetVars(pluginID)
	if err != nil {
		p.report(err)
	}
	return process.GetPluginVars(pluginID, process.LaunchEnvironment{
		VaultPath: vaultPath, Port: processPort, HostPort: hostPort, Token: token, RuntimeVars: runtimeVars,
		StorageVars: storageVars,
	})
}

func (p *Processes) startBuiltIn(ctx context.Context, launch Launch, sup *lifecycle.Supervisor) (transport.Endpoint, error) {
	plug := launch.Plugin
	token, err := transport.GenerateAuthToken()
	if err != nil {
		return NoEndpoint(), fmt.Errorf("generate token: %w", err)
	}
	runtimeVars, err := exportRuntimeVars(ctx, p, launch.Requirements)
	if err != nil {
		return NoEndpoint(), fmt.Errorf("launch plugin %s: %w", plug.Manifest.ID, err)
	}
	vars := make(map[string]string)
	launchVars := p.getEnvironmentVars(ctx, plug.Manifest.ID, launch.VaultPath, 0, token, runtimeVars)
	for _, entry := range process.SanitizeEnvironment(os.Environ(), launchVars, plug.Manifest.GetRuntimeIDs()) {
		name, value, _ := strings.Cut(entry, "=")
		vars[name] = value
	}
	handler, err := launch.BuiltIn.Serve(vars)
	if err != nil {
		return NoEndpoint(), fmt.Errorf("launch plugin %s: %w", plug.Manifest.ID, err)
	}

	endpoint := transport.Endpoint{Host: "inprocess", Port: 0, Token: token}
	p.sups.Clients().Add(token, transport.NewInProcessTransport(handler))
	if err := transport.CheckPluginProtocol(ctx, p.sups.Clients().Get(endpoint), plug.Manifest, endpoint); err != nil {
		p.sups.Clients().Remove(token)
		shutdownBuiltIn(ctx, p, endpoint)
		return NoEndpoint(), err
	}

	if err := p.sups.AttachInProcess(plug.Manifest.ID, sup, endpoint, p.clock.Now()); err != nil {
		shutdownBuiltIn(ctx, p, endpoint)
		return NoEndpoint(), err
	}
	return endpoint, nil
}

func shutdownBuiltIn(ctx context.Context, p *Processes, endpoint transport.Endpoint) {
	if shutdown := createRPCShutdown(endpoint, p.sups.Clients().Get(endpoint), time.Second); shutdown != nil {
		if err := shutdown(context.WithoutCancel(ctx)); err != nil {
			p.report(fmt.Errorf("shut down the handler of built-in plugin that was not activated: %w", err))
		}
	}
}

func exportRuntimeVars(ctx context.Context, p *Processes, requirements []manifest.DependencyRequirement) ([]string, error) {
	vars, err := p.GetRuntimeVars(ctx, requirements)
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
