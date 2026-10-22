package port

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
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

// DependencyState is where a dependency stands on the machine.
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
	ID             string            `json:"id"`
	Kind           DependencyKind    `json:"kind"`
	Version        string            `json:"version,omitempty"`
	SourceURL      string            `json:"sourceUrl,omitempty"`
	ChecksumSHA256 string            `json:"checksumSha256,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`

	// PluginDir is the folder of the plugin that declares the requirement. A
	// provisioner looks beside it first.
	PluginDir string `json:"-"`
}

// ResolvedDependency is where a ready dependency sits on disk and how a process reaches it.
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
	// Status answers where the dependency stands without installing it.
	Status(ctx context.Context, req DependencyRequirement) (DependencyStatus, error)
}

// DependencyCatalog is the provisioners together: it routes a requirement to
// the provisioner that handles it, shares one run between identical requests
// and answers for every requirement it has been told of.
type DependencyCatalog interface {
	DependencyProvisioner
	// Register records a requirement so that it is answered for by id.
	Register(req DependencyRequirement)
	// GetStatus answers where the registered dependency with the id stands.
	GetStatus(ctx context.Context, id string) (DependencyStatus, error)
	// List answers where every registered dependency stands.
	List(ctx context.Context) ([]DependencyStatus, error)
	// SetObserver sets what is told each status a provisioning reaches.
	SetObserver(observer func(DependencyRequirement, DependencyStatus))
	// Lookup returns the registered requirement with the id.
	Lookup(id string) (DependencyRequirement, bool)
}

// GetDependencyKinds is every kind a manifest may declare.
func GetDependencyKinds() []DependencyKind {
	return []DependencyKind{DependencyKindRuntime, DependencyKindModel, DependencyKindBinary, DependencyKindDataset}
}

// GetDependencyRequirements is the requirements of the plugin: its declared
// runtimes, each as a runtime requirement, then its dependencies.
func (m PluginManifest) GetDependencyRequirements(pluginDir string) []DependencyRequirement {
	out := make([]DependencyRequirement, 0, len(m.Runtimes)+len(m.Dependencies))
	for _, name := range m.Runtimes {
		out = append(out, DependencyRequirement{ID: name, Kind: DependencyKindRuntime, PluginDir: pluginDir})
	}
	for _, req := range m.Dependencies {
		req.PluginDir = pluginDir
		out = append(out, req)
	}
	return out
}

// GetRuntimeIDs is the ids of the runtimes the plugin requires.
func (m PluginManifest) GetRuntimeIDs() []string {
	var ids []string
	for _, req := range m.GetDependencyRequirements("") {
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
	if address, err := url.Parse(req.SourceURL); err != nil || address.Scheme != "https" || address.Host == "" {
		issues = append(issues, fmt.Errorf("%w: %s sourceUrl %q is not an https address", ErrInvalidDependency, req.ID, req.SourceURL))
	}
	if !dependencyChecksumPattern.MatchString(req.ChecksumSHA256) {
		issues = append(issues, fmt.Errorf("%w: %s checksumSha256 is not 64 lower-case hex digits", ErrInvalidDependency, req.ID))
	}
	return issues
}
