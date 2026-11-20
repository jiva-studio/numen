package pluginport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// DependencyRegistry provisions the dependencies the plugins declare and reports each status to its observer.
type DependencyRegistry interface {
	CanProvision(req manifest.DependencyRequirement) bool
	Provision(ctx context.Context, req manifest.DependencyRequirement, onProgress func(plugin.DependencyStatus)) (plugin.ResolvedDependency, error)
	// Resolve returns where the dependency is or will be and whether it is ready, without the network, a lock wait or a download.
	Resolve(req manifest.DependencyRequirement) (plugin.ResolvedDependency, plugin.DependencyState, error)
	SetObserver(observer func(manifest.DependencyRequirement, plugin.DependencyStatus))
	// Cancel stops the download of the requirement and reports whether one was in progress.
	Cancel(req manifest.DependencyRequirement) bool
}
