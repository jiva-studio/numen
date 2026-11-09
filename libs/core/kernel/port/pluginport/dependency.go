package pluginport

import (
	"context"
)

// DependencyState is the install state of a dependency on this machine.
type DependencyState string

// The states a dependency is reported in.
const (
	DependencyStateMissing      DependencyState = "missing"
	DependencyStateProvisioning DependencyState = "provisioning"
	DependencyStateReady        DependencyState = "ready"
	DependencyStateFailed       DependencyState = "failed"
)

// ResolvedDependency is the install location of a ready dependency and how a process uses it.
type ResolvedDependency struct {
	ID          string
	RootPath    string
	PrimaryPath string
	EnvVars     map[string]string
	Metadata    map[string]string
}

// DependencyStatus is the state of one dependency at one moment.
type DependencyStatus struct {
	ID              string
	State           DependencyState
	BytesDownloaded int64
	BytesTotal      int64
	ProgressPercent float64
	Message         string
	Error           string
	Resolved        *ResolvedDependency
}

// DependencyProvisioner makes the dependencies it can handle ready on this machine.
type DependencyProvisioner interface {
	// CanProvision reports whether the provisioner handles the requirement.
	CanProvision(req DependencyRequirement) bool
	// Provision makes the dependency ready and reports each step to onProgress.
	Provision(ctx context.Context, req DependencyRequirement, onProgress func(DependencyStatus)) (ResolvedDependency, error)
	// Status reports the state of the dependency without installing it.
	Status(ctx context.Context, req DependencyRequirement) (DependencyStatus, error)
}

// DependencyCatalog is the provisioners combined: it routes a requirement to the
// provisioner that handles it, shares one run between identical requests and reports
// on every requirement it has seen.
type dependencyRegistry interface {
	Register(req DependencyRequirement)
	Lookup(id string) (DependencyRequirement, bool)
}

type DependencyCatalog interface {
	DependencyProvisioner
	dependencyRegistry
	GetStatus(ctx context.Context, id string) (DependencyStatus, error)
	List(ctx context.Context) ([]DependencyStatus, error)
	SetObserver(observer func(DependencyRequirement, DependencyStatus))
}
