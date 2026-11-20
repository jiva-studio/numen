package manifest

import (
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
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

// ErrInvalidDependency is a dependency a manifest declares that the host cannot act on.
var ErrInvalidDependency = errors.New("invalid dependency")

var (
	dependencyChecksumPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	platformPattern           = regexp.MustCompile(`^[a-z0-9]+/[a-z0-9]+$`)
	environmentEntryPattern   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*\*?$`)
)

// DependencyRequirement is one dependency a plugin declares. Sources holds the file fetched on each
// platform, at most one per platform. Environment holds the names of host variables, or prefixes
// ending in *, passed to the plugin process. Search holds glob patterns of files on this machine
// that are the dependency.
type DependencyRequirement struct {
	ID             string
	Kind           DependencyKind
	Version        string
	SourceURL      string
	ChecksumSHA256 string
	Metadata       map[string]string
	Sources        []DependencySource
	Environment    []string
	Search         []string

	PluginDir string
}

// DependencySource is the file a dependency is fetched from on one platform, os/arch as in
// linux/amd64. ArchiveMember names the file unpacked from the archive at SourceURL, and
// MemberSHA256 is the sha256 it holds.
type DependencySource struct {
	Platform       string
	SourceURL      string
	ChecksumSHA256 string
	ArchiveMember  string
	MemberSHA256   string
}

// GetSource returns the file the dependency is fetched from on the platform goos/goarch: its entry
// of Sources, or SourceURL for every platform.
func (req DependencyRequirement) GetSource(goos, goarch string) (DependencySource, bool) {
	if len(req.Sources) > 0 {
		platform := goos + "/" + goarch
		i := slices.IndexFunc(req.Sources, func(source DependencySource) bool { return source.Platform == platform })
		if i < 0 {
			return DependencySource{}, false
		}
		return req.Sources[i], true
	}
	if req.SourceURL == "" {
		return DependencySource{}, false
	}
	return DependencySource{SourceURL: req.SourceURL, ChecksumSHA256: req.ChecksumSHA256}, true
}

// GetIdentity returns the string that tells one download from another: the kind, id, version, source on this platform and plugin
// directory. Requirements with the same identity are one download.
func (req DependencyRequirement) GetIdentity() string {
	source, _ := req.GetSource(runtime.GOOS, runtime.GOARCH)
	return strings.Join([]string{
		string(req.Kind), req.ID, req.Version, source.SourceURL, source.ChecksumSHA256, source.ArchiveMember, source.MemberSHA256, req.PluginDir,
	}, "\x00")
}

// GetDependencyKinds is every kind a manifest may declare.
func GetDependencyKinds() []DependencyKind {
	return []DependencyKind{DependencyKindRuntime, DependencyKindModel, DependencyKindBinary, DependencyKindDataset}
}

// GetDependencyRequirements is the requirements of the plugin under the settings in force: its
// dependencies. A dependency an option of a setting provisions is a requirement only while that
// option is in force.
func (m Manifest) GetDependencyRequirements(pluginDir string, values map[string]any) []DependencyRequirement {
	optional, active := m.getOptionalDependencies(values)
	out := make([]DependencyRequirement, 0, len(m.Dependencies))
	for _, req := range m.Dependencies {
		if optional[req.ID] && !active[req.ID] {
			continue
		}
		req.PluginDir = pluginDir
		out = append(out, req)
	}
	return out
}

// GetDependencyEnvironment is the host variables, names and prefixes ending in *, that the
// dependencies of the plugin pass to its process.
func (m Manifest) GetDependencyEnvironment() []string {
	var entries []string
	for _, req := range m.GetDependencyRequirements("", nil) {
		for _, entry := range req.Environment {
			if !slices.Contains(entries, entry) {
				entries = append(entries, entry)
			}
		}
	}
	return entries
}

func (m Manifest) listDependencyIssues() []error {
	var issues []error
	observed := make(map[string]bool, len(m.Dependencies))
	for _, req := range m.Dependencies {
		switch {
		case req.ID == "":
			issues = append(issues, fmt.Errorf("%w: id cannot be empty", ErrInvalidDependency))
		case observed[req.ID]:
			issues = append(issues, fmt.Errorf("%w: %s is declared twice", ErrInvalidDependency, req.ID))
		case !slices.Contains(GetDependencyKinds(), req.Kind):
			issues = append(issues, fmt.Errorf("%w: %s has kind %q, expected one of %v", ErrInvalidDependency, req.ID, req.Kind, GetDependencyKinds()))
		}
		observed[req.ID] = true
		issues = append(issues, listDependencySourceIssues(req)...)
		issues = append(issues, listPlatformSourceIssues(req)...)
		issues = append(issues, listEnvironmentIssues(req)...)
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
	if len(req.Sources) > 0 {
		return []error{fmt.Errorf("%w: %s declares both sourceUrl and sources", ErrInvalidDependency, req.ID)}
	}
	return listSourceIssues(req.ID, DependencySource{SourceURL: req.SourceURL, ChecksumSHA256: req.ChecksumSHA256})
}

func listPlatformSourceIssues(req DependencyRequirement) []error {
	var issues []error
	observed := make(map[string]bool, len(req.Sources))
	for _, source := range req.Sources {
		name := req.ID + " " + source.Platform
		switch {
		case !platformPattern.MatchString(source.Platform):
			issues = append(issues, fmt.Errorf("%w: %s source platform %q is not os/arch", ErrInvalidDependency, req.ID, source.Platform))
		case observed[source.Platform]:
			issues = append(issues, fmt.Errorf("%w: %s declares platform %s twice", ErrInvalidDependency, req.ID, source.Platform))
		}
		observed[source.Platform] = true
		if source.SourceURL == "" {
			issues = append(issues, fmt.Errorf("%w: %s has no sourceUrl", ErrInvalidDependency, name))
			continue
		}
		issues = append(issues, listSourceIssues(name, source)...)
	}
	return issues
}

func listSourceIssues(name string, source DependencySource) []error {
	var issues []error
	if !domain.IsHTTPSAddress(source.SourceURL) {
		issues = append(issues, fmt.Errorf("%w: %s sourceUrl %q is not an https address", ErrInvalidDependency, name, source.SourceURL))
	}
	if !dependencyChecksumPattern.MatchString(source.ChecksumSHA256) {
		issues = append(issues, fmt.Errorf("%w: %s checksumSha256 is not 64 lower-case hex digits", ErrInvalidDependency, name))
	}
	switch {
	case source.ArchiveMember == "" && source.MemberSHA256 != "":
		issues = append(issues, fmt.Errorf("%w: %s has a memberSha256 and no archiveMember", ErrInvalidDependency, name))
	case source.ArchiveMember != "" && !dependencyChecksumPattern.MatchString(source.MemberSHA256):
		issues = append(issues, fmt.Errorf("%w: %s memberSha256 is not 64 lower-case hex digits", ErrInvalidDependency, name))
	}
	return issues
}

func listEnvironmentIssues(req DependencyRequirement) []error {
	var issues []error
	for _, entry := range req.Environment {
		if !environmentEntryPattern.MatchString(entry) {
			issues = append(issues, fmt.Errorf("%w: %s environment %q is not a variable name or a prefix ending in *", ErrInvalidDependency, req.ID, entry))
		}
	}
	for _, pattern := range req.Search {
		if pattern == "" {
			issues = append(issues, fmt.Errorf("%w: %s has an empty search pattern", ErrInvalidDependency, req.ID))
		}
	}
	return issues
}

// NewModelFile declares a downloadable model file identified by its SHA-256.
func NewModelFile(address, digest string) (DependencyRequirement, error) {
	source := DependencySource{SourceURL: address, ChecksumSHA256: digest}
	if err := errors.Join(listSourceIssues("model file", source)...); err != nil {
		return DependencyRequirement{}, err
	}
	return DependencyRequirement{ID: digest, Kind: DependencyKindModel, SourceURL: address, ChecksumSHA256: digest}, nil
}
