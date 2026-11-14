package pluginport

import (
	"context"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// DependencyProvisioner makes the dependencies it can handle ready on this machine.
type DependencyProvisioner interface {
	// CanProvision reports whether the provisioner handles the requirement.
	CanProvision(req manifest.DependencyRequirement) bool
	// Provision makes the dependency ready and reports each step to onProgress.
	Provision(ctx context.Context, req manifest.DependencyRequirement, onProgress func(plugin.DependencyStatus)) (plugin.ResolvedDependency, error)
}

// DependencyRegistry provisions the dependencies the plugins declare and reports each status to its observer.
type DependencyRegistry interface {
	CanProvision(req manifest.DependencyRequirement) bool
	Provision(ctx context.Context, req manifest.DependencyRequirement, onProgress func(plugin.DependencyStatus)) (plugin.ResolvedDependency, error)
	SetObserver(observer func(manifest.DependencyRequirement, plugin.DependencyStatus))
}

// DependencyReporter receives the function that observes each change of the state of a dependency.
type DependencyReporter interface {
	ObserveDependencies(observer func(manifest.DependencyRequirement, plugin.DependencyStatus))
}
