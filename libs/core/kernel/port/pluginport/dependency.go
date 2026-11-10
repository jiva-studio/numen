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
	// Status reports the state of the dependency without installing it.
	Status(ctx context.Context, req manifest.DependencyRequirement) (plugin.DependencyStatus, error)
}

// DependencyCatalog is the provisioners combined: it routes a requirement to the
// provisioner that handles it, shares one run between identical requests and reports
// on every requirement it has seen.
type dependencyRegistry interface {
	Register(req manifest.DependencyRequirement)
	Lookup(id string) (manifest.DependencyRequirement, bool)
}

type DependencyCatalog interface {
	DependencyProvisioner
	dependencyRegistry
	GetStatus(ctx context.Context, id string) (plugin.DependencyStatus, error)
	List(ctx context.Context) ([]plugin.DependencyStatus, error)
	SetObserver(observer func(manifest.DependencyRequirement, plugin.DependencyStatus))
}
