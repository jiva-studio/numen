package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"path"
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
// that are the dependency. SizeBytes is the size of the file fetched.
type DependencyRequirement struct {
	ID             string
	Kind           DependencyKind
	Version        string
	SourceURL      string
	SourceURLs     []string
	ChecksumSHA256 string
	Metadata       map[string]string
	Sources        []DependencySource
	Environment    []string
	Search         []string
	SizeBytes      int64

	PluginDir string
}

// DependencySource is the file a dependency is fetched from on one platform, os/arch as in
// linux/amd64. ArchiveMember names the file unpacked from the archive at SourceURL, and
// MemberSHA256 is the sha256 it holds.
type DependencySource struct {
	Platform       string
	SourceURL      string
	SourceURLs     []string
	ChecksumSHA256 string
	ArchiveMember  string
	MemberSHA256   string
}

// GetURLs returns the addresses the source is fetched from, in the order they are tried.
func (s DependencySource) GetURLs() []string {
	if len(s.SourceURLs) > 0 {
		return s.SourceURLs
	}
	if s.SourceURL != "" {
		return []string{s.SourceURL}
	}
	return nil
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
		source := req.Sources[i]
		if source.SourceURL == "" && len(source.SourceURLs) > 0 {
			source.SourceURL = source.SourceURLs[0]
		}
		return source, true
	}
	if req.SourceURL == "" && len(req.SourceURLs) == 0 {
		return DependencySource{}, false
	}
	sourceURL := req.SourceURL
	if sourceURL == "" {
		sourceURL = req.SourceURLs[0]
	}
	return DependencySource{SourceURL: sourceURL, SourceURLs: req.SourceURLs, ChecksumSHA256: req.ChecksumSHA256}, true
}

// GetIdentity returns the string that tells one download from another: the kind, id, version, source on this platform and plugin
// directory. The checksum alone distinguishes a model file stored by its checksum. Requirements with the same identity are
// one download.
func (req DependencyRequirement) GetIdentity() string {
	if req.Kind == DependencyKindModel && req.ChecksumSHA256 != "" && req.ID == req.ChecksumSHA256 && req.Version == "" && len(req.Sources) == 0 {
		return string(req.Kind) + "\x00" + req.ChecksumSHA256
	}
	source, _ := req.GetSource(runtime.GOOS, runtime.GOARCH)
	return strings.Join([]string{
		string(req.Kind), req.ID, req.Version, strings.Join(source.GetURLs(), "\x00"), source.ChecksumSHA256, source.ArchiveMember, source.MemberSHA256, req.PluginDir,
	}, "\x00")
}

// GetDependencyKinds is every kind a manifest may declare.
func GetDependencyKinds() []DependencyKind {
	return []DependencyKind{DependencyKindRuntime, DependencyKindModel, DependencyKindBinary, DependencyKindDataset}
}

// GetDependencyRequirements is the requirements of the plugin under the settings in force: its
// dependencies other than models, which the plugin fetches on request. A dependency an option of a
// setting provisions is a requirement only while that option is in force.
func (m Manifest) GetDependencyRequirements(pluginDir string, values map[string]any) []DependencyRequirement {
	optional, active := m.getOptionalDependencies(values)
	out := make([]DependencyRequirement, 0, len(m.Dependencies))
	for _, req := range m.Dependencies {
		if req.Kind == DependencyKindModel || (optional[req.ID] && !active[req.ID]) {
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
		issues = append(issues, listSizeIssues(req)...)
		issues = append(issues, listPlatformSourceIssues(req)...)
		issues = append(issues, listEnvironmentIssues(req)...)
	}
	return issues
}

func listDependencySourceIssues(req DependencyRequirement) []error {
	if req.SourceURL == "" && len(req.SourceURLs) == 0 {
		if req.ChecksumSHA256 != "" {
			return []error{fmt.Errorf("%w: %s has a checksum and no sourceUrl", ErrInvalidDependency, req.ID)}
		}
		return nil
	}
	if req.SourceURL != "" && len(req.SourceURLs) > 0 {
		return []error{fmt.Errorf("%w: %s declares both sourceUrl and sourceUrls", ErrInvalidDependency, req.ID)}
	}
	if len(req.Sources) > 0 {
		return []error{fmt.Errorf("%w: %s declares both sourceUrl and sources", ErrInvalidDependency, req.ID)}
	}
	return listSourceIssues(req.ID, DependencySource{SourceURL: req.SourceURL, SourceURLs: req.SourceURLs, ChecksumSHA256: req.ChecksumSHA256})
}

func listSizeIssues(req DependencyRequirement) []error {
	if req.SizeBytes < 0 || (req.Kind == DependencyKindModel && req.SizeBytes == 0 && (req.SourceURL != "" || len(req.SourceURLs) > 0 || len(req.Sources) > 0)) {
		return []error{fmt.Errorf("%w: %s needs a sizeBytes above 0", ErrInvalidDependency, req.ID)}
	}
	return nil
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
		switch {
		case source.SourceURL == "" && len(source.SourceURLs) == 0:
			issues = append(issues, fmt.Errorf("%w: %s has no sourceUrl", ErrInvalidDependency, name))
			continue
		case source.SourceURL != "" && len(source.SourceURLs) > 0:
			issues = append(issues, fmt.Errorf("%w: %s declares both sourceUrl and sourceUrls", ErrInvalidDependency, name))
			continue
		}
		issues = append(issues, listSourceIssues(name, source)...)
	}
	return issues
}

func listSourceIssues(name string, source DependencySource) []error {
	var issues []error
	for _, address := range source.GetURLs() {
		if !domain.IsHTTPSAddress(address) {
			issues = append(issues, fmt.Errorf("%w: %s sourceUrl %q is not an https address", ErrInvalidDependency, name, address))
		}
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

// NewModelFile declares a downloadable model file. A profile without a pin identifies the source by its URL.
func NewModelFile(address, digest string) (DependencyRequirement, error) {
	if digest == "" {
		if !domain.IsHTTPSAddress(address) {
			return DependencyRequirement{}, fmt.Errorf("%w: model file source must use HTTPS", ErrInvalidDependency)
		}
		key := sha256.Sum256([]byte(address))
		return DependencyRequirement{ID: "source-" + hex.EncodeToString(key[:]), Kind: DependencyKindModel, SourceURL: address}, nil
	}
	source := DependencySource{SourceURL: address, ChecksumSHA256: digest}
	if err := errors.Join(listSourceIssues("model file", source)...); err != nil {
		return DependencyRequirement{}, err
	}
	return DependencyRequirement{
		ID: digest, Kind: DependencyKindModel, SourceURL: address, ChecksumSHA256: digest,
		Metadata: map[string]string{metadataTitle: getModelTitle(address)},
	}, nil
}

const metadataTitle = "title"

// GetTitle is the name a person reads for the dependency: its title metadata, or its id.
func (req DependencyRequirement) GetTitle() string {
	if title := req.Metadata[metadataTitle]; title != "" {
		return title
	}
	return req.ID
}

func getModelTitle(address string) string {
	_, rest, _ := strings.Cut(address, "://")
	rest, _, _ = strings.Cut(rest, "?")
	host, route, _ := strings.Cut(rest, "/")
	name := strings.TrimSuffix(path.Base(route), path.Ext(route))
	if name != "inference" && name != "model" {
		return name + " model"
	}
	segments := strings.Split(route, "/")
	for index, segment := range segments {
		if segment == "resolve" && index > 0 {
			return segments[index-1] + " model"
		}
	}
	return host + " model"
}

// FindModelDependency returns the dependency of kind model that the manifest declares under id.
func (m Manifest) FindModelDependency(id string) (DependencyRequirement, bool) {
	i := slices.IndexFunc(m.Dependencies, func(req DependencyRequirement) bool {
		return id != "" && req.ID == id && req.Kind == DependencyKindModel
	})
	if i < 0 {
		return DependencyRequirement{}, false
	}
	return m.Dependencies[i], true
}

// NewDeclaredModelFile returns the requirement that fetches a model a manifest declares. It is stored
// by its checksum, as NewModelFile stores a pinned file, so that every declaration of one file and
// every fetch of it by address share one download. It keeps the sources, the size and the title.
func NewDeclaredModelFile(req DependencyRequirement) (DependencyRequirement, error) {
	if req.Kind != DependencyKindModel || !dependencyChecksumPattern.MatchString(req.ChecksumSHA256) {
		return DependencyRequirement{}, fmt.Errorf("%w: model %s has no checksumSha256", ErrInvalidDependency, req.ID)
	}
	metadata := maps.Collect(maps.All(req.Metadata))
	metadata[metadataTitle] = req.GetTitle()
	return DependencyRequirement{ID: req.ChecksumSHA256, Kind: DependencyKindModel, SourceURL: req.SourceURL, SourceURLs: slices.Clone(req.SourceURLs),
		ChecksumSHA256: req.ChecksumSHA256, SizeBytes: req.SizeBytes, Metadata: metadata}, nil
}
