package pluginport

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// DependencyKind is what a dependency is.
type DependencyKind string

// The kinds a manifest declares a dependency as.
const (
	DependencyKindRuntime DependencyKind = "runtime"
	DependencyKindModel   DependencyKind = "model"
	DependencyKindBinary  DependencyKind = "binary"
	DependencyKindDataset DependencyKind = "dataset"
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

// ErrInvalidDependency is a dependency a manifest declares that the host cannot act on.
var ErrInvalidDependency = errors.New("invalid dependency")

// ErrNoProvisioner is a requirement that no provisioner can make ready.
var ErrNoProvisioner = errors.New("no provisioner for the dependency")

var dependencyChecksumPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// DependencyRequirement is one dependency a plugin declares.
type DependencyRequirement struct {
	ID             string
	Kind           DependencyKind
	Version        string
	SourceURL      string
	ChecksumSHA256 string
	Metadata       map[string]string

	// PluginDir is the folder of the plugin that declares the requirement. A
	// provisioner looks beside it first.
	PluginDir string
}

// ResolvedDependency is the install location of a ready dependency and how a process uses it.
type ResolvedDependency struct {
	ID string
	// RootPath is the folder that holds everything the dependency installed.
	RootPath string
	// PrimaryPath is the file a plugin opens or runs.
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
	// Error is set while State is failed.
	Error string
	// Resolved is set while State is ready.
	Resolved *ResolvedDependency
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

// GetDependencyKinds is every kind a manifest may declare.
func GetDependencyKinds() []DependencyKind {
	return []DependencyKind{DependencyKindRuntime, DependencyKindModel, DependencyKindBinary, DependencyKindDataset}
}

// GetDependencyRequirements is the requirements of the plugin under the settings in force: its
// declared runtimes, each as a runtime requirement, then its dependencies. A dependency an option of a
// setting provisions is a requirement only while that option is in force.
func (m PluginManifest) GetDependencyRequirements(pluginDir string, values map[string]any) []DependencyRequirement {
	optional, active := m.getOptionalDependencies(values)
	out := make([]DependencyRequirement, 0, len(m.Runtimes)+len(m.Dependencies))
	for _, name := range m.Runtimes {
		out = append(out, DependencyRequirement{ID: name, Kind: DependencyKindRuntime, PluginDir: pluginDir})
	}
	for _, req := range m.Dependencies {
		if optional[req.ID] && !active[req.ID] {
			continue
		}
		req.PluginDir = pluginDir
		out = append(out, req)
	}
	return out
}

// GetRuntimeIDs is the ids of the runtimes the plugin requires.
func (m PluginManifest) GetRuntimeIDs() []string {
	var ids []string
	for _, req := range m.GetDependencyRequirements("", nil) {
		if req.Kind == DependencyKindRuntime && !slices.Contains(ids, req.ID) {
			ids = append(ids, req.ID)
		}
	}
	return ids
}

func (m PluginManifest) listDependencyIssues() []error {
	var issues []error
	observed := make(map[string]bool, len(m.Dependencies))
	for _, req := range m.Dependencies {
		switch {
		case req.ID == "":
			issues = append(issues, fmt.Errorf("%w: id cannot be empty", ErrInvalidDependency))
		case observed[req.ID]:
			issues = append(issues, fmt.Errorf("%w: %s is declared twice", ErrInvalidDependency, req.ID))
		case req.Kind == "" || !slices.Contains(GetDependencyKinds(), req.Kind):
			issues = append(issues, fmt.Errorf("%w: %s has kind %q, expected one of %v", ErrInvalidDependency, req.ID, req.Kind, GetDependencyKinds()))
		}
		observed[req.ID] = true
		issues = append(issues, listDependencySourceIssues(req)...)
	}
	return issues
}

func listDependencySourceIssues(req DependencyRequirement) []error {
	if req.SourceURL == "" {
		if req.ChecksumSHA256 != "" {
			return []error{fmt.Errorf("%w: %s has a checksum and no sourceUrl", ErrInvalidDependency, req.ID)}
		}
		return nil
	}
	var issues []error
	if !domain.IsHTTPSAddress(req.SourceURL) {
		issues = append(issues, fmt.Errorf("%w: %s sourceUrl %q is not an https address", ErrInvalidDependency, req.ID, req.SourceURL))
	}
	if !dependencyChecksumPattern.MatchString(req.ChecksumSHA256) {
		issues = append(issues, fmt.Errorf("%w: %s checksumSha256 is not 64 lower-case hex digits", ErrInvalidDependency, req.ID))
	}
	return issues
}
