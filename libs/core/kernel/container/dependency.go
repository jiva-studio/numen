package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// DependencyReporter receives the function that observes each change of the state of a dependency.
type DependencyReporter interface {
	ObserveDependencies(observer func(manifest.DependencyRequirement, plugin.DependencyStatus))
	// CancelDependency stops the download of the requirement, if one runs.
	CancelDependency(req manifest.DependencyRequirement)
	// CancelDependencies stops every download in progress.
	CancelDependencies()
}
