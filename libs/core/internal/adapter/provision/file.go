package provision

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// ErrNoFolder is a dependency asked for where no folder is set to keep dependencies in.
var ErrNoFolder = errors.New("no folder to keep dependencies in")

// ErrOffline is a dependency that is not on this machine while fetching is forbidden.
var ErrOffline = errors.New("fetching is forbidden")

// Config is what dependencies are provisioned from.
type Config struct {
	// Dir is the folder dependencies are kept in. Empty keeps none: nothing is provisioned.
	Dir string
	// Offline forbids fetching.
	Offline bool
	// Client fetches the files. Nil follows a redirect only to https.
	Client *http.Client
	// Discovery looks for what the machine holds. Zero searches this platform's folders.
	Discovery discovery
	// Backoff is the waits between attempts. Zero is the default.
	Backoff backoff
	// Clock paces the progress reports.
	Clock machineport.Clock
}

// FileProvisioner provisions a dependency that is one downloadable file with a
// pinned checksum: it looks on the machine first, then in its own folder, and
// fetches last.
type FileProvisioner struct {
	cfg Config
}

// NewFileProvisioner returns a provisioner for the configuration.
func NewFileProvisioner(cfg Config) *FileProvisioner {
	if cfg.Client == nil {
		cfg.Client = newClient()
	}
	if cfg.Discovery.dirs == nil && cfg.Discovery.ldconfig == nil {
		cfg.Discovery = newDiscovery()
	}
	if cfg.Backoff.attempts == 0 {
		cfg.Backoff = newDefaultBackoff()
	}
	return &FileProvisioner{cfg: cfg}
}

// CanProvision reports whether the requirement names a file to fetch, or a
// program or library the machine may already hold.
func (p *FileProvisioner) CanProvision(req pluginport.DependencyRequirement) bool {
	return req.SourceURL != "" || req.Kind == pluginport.DependencyKindBinary || req.Kind == pluginport.DependencyKindRuntime
}

func (p *FileProvisioner) getDir(req pluginport.DependencyRequirement) (string, error) {
	root := p.cfg.Dir
	if root == "" {
		return "", ErrNoFolder
	}
	return getDependencyDir(root, string(req.Kind), req.ID, req.Version), nil
}

// Status answers whether the machine or the folder holds the dependency.
func (p *FileProvisioner) Status(_ context.Context, req pluginport.DependencyRequirement) (pluginport.DependencyStatus, error) {
	if found := p.cfg.Discovery.find(req); found != "" {
		return readyStatus(req, filepath.Dir(found), found), nil
	}
	dir, err := p.getDir(req)
	if err != nil {
		return pluginport.DependencyStatus{}, err
	}
	if resolved, ok := readCached(req, dir); ok {
		return pluginport.DependencyStatus{ID: req.ID, State: pluginport.DependencyStateReady, ProgressPercent: 100, Resolved: &resolved}, nil
	}
	return pluginport.DependencyStatus{ID: req.ID, State: pluginport.DependencyStateMissing}, nil
}

// Provision returns the dependency: the one the machine holds, the one in the
// folder, or the one it fetches. Processes provisioning the same dependency
// take turns.
func (p *FileProvisioner) Provision(
	ctx context.Context, req pluginport.DependencyRequirement, onProgress func(pluginport.DependencyStatus),
) (_ pluginport.ResolvedDependency, err error) {
	if onProgress == nil {
		onProgress = func(pluginport.DependencyStatus) {}
	}
	if found := p.cfg.Discovery.find(req); found != "" {
		return newResolved(req, filepath.Dir(found), found), nil
	}
	dir, err := p.getDir(req)
	if err != nil {
		return pluginport.ResolvedDependency{}, err
	}
	if req.SourceURL == "" {
		return pluginport.ResolvedDependency{}, fmt.Errorf("%s %s is not on this machine and names no sourceUrl", req.Kind, req.ID)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return pluginport.ResolvedDependency{}, err
	}
	release, err := lockFile(ctx, dir+".lock")
	if err != nil {
		return pluginport.ResolvedDependency{}, err
	}
	defer func() { err = errors.Join(err, release()) }()

	if resolved, ok := readCached(req, dir); ok {
		return resolved, nil
	}
	if p.cfg.Offline {
		return pluginport.ResolvedDependency{}, newDownloadError(req, ErrOffline)
	}
	name, err := getFileName(req)
	if err != nil {
		return pluginport.ResolvedDependency{}, err
	}
	primary := filepath.Join(dir, name)
	title := "Downloading " + req.ID
	err = transfer{
		client: p.cfg.Client, address: req.SourceURL, dest: primary, sum: req.ChecksumSHA256, backoff: p.cfg.Backoff, clock: p.cfg.Clock,
		onProgress: func(done, total int64) {
			onProgress(pluginport.DependencyStatus{
				ID: req.ID, State: pluginport.DependencyStateProvisioning, BytesDownloaded: done, BytesTotal: total,
				ProgressPercent: getPercent(done, total), Message: title,
			})
		},
	}.fetch(ctx)
	if err != nil {
		return pluginport.ResolvedDependency{}, newDownloadError(req, err)
	}
	if err := writeReady(ctx, dir, name, req.ChecksumSHA256); err != nil {
		return pluginport.ResolvedDependency{}, err
	}
	return newResolved(req, dir, primary), nil
}

// newDownloadError is a failed fetch of the dependency. The cause does not name the address.
func newDownloadError(req pluginport.DependencyRequirement, cause error) error {
	return fmt.Errorf("download %s %s from %s: %w", req.Kind, req.ID, redactAddress(req.SourceURL), cause)
}

// redactAddress returns the address without its credentials, query and fragment, which can hold
// tokens. An address that does not parse loses the same parts by position and is quoted, so its
// control characters are escaped.
func redactAddress(raw string) string {
	address, err := url.Parse(raw)
	if err != nil {
		return strconv.Quote(redactUnparsedAddress(raw))
	}
	address.User, address.RawQuery, address.ForceQuery, address.Fragment, address.RawFragment = nil, "", false, "", ""
	return address.String()
}

// redactUnparsedAddress cuts the address at its first ? or #, and drops what its authority holds
// up to the last @.
func redactUnparsedAddress(raw string) string {
	if end := strings.IndexAny(raw, "?#"); end >= 0 {
		raw = raw[:end]
	}
	scheme, rest, ok := strings.Cut(raw, "://")
	if !ok {
		return raw
	}
	authority, tail := rest, ""
	if slash := strings.IndexByte(rest, '/'); slash >= 0 {
		authority, tail = rest[:slash], rest[slash:]
	}
	if atSign := strings.LastIndexByte(authority, '@'); atSign >= 0 {
		authority = authority[atSign+1:]
	}
	return scheme + "://" + authority + tail
}

func getPercent(done, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(done) * 100 / float64(total)
}

// getFileName returns what the fetched file is called in the dependency's folder.
func getFileName(req pluginport.DependencyRequirement) (string, error) {
	name, err := pickFileName(req)
	if err != nil {
		return "", err
	}
	if isReserved(name) {
		return "", fmt.Errorf("%s %s: %q is a name the folder keeps for itself", req.Kind, req.ID, name)
	}
	return name, nil
}

// pickFileName is the name the requirement states, else the last segment of its
// address, else its identifier.
func pickFileName(req pluginport.DependencyRequirement) (string, error) {
	if named := req.Metadata["fileName"]; named != "" {
		return cleanName(named), nil
	}
	address, err := url.Parse(req.SourceURL)
	if err != nil {
		return "", err
	}
	if base := path.Base(address.Path); base != "" && base != "." && base != "/" {
		return cleanName(base), nil
	}
	return cleanName(req.ID), nil
}

// readCached returns the dependency a folder with the sentinel holds, when the
// folder was fetched with the checksum the requirement pins.
func readCached(req pluginport.DependencyRequirement, dir string) (pluginport.ResolvedDependency, bool) {
	if !isReady(dir) {
		return pluginport.ResolvedDependency{}, false
	}
	name, sum, err := readReady(dir)
	if err != nil || !isPlainName(name) || sum != req.ChecksumSHA256 {
		return pluginport.ResolvedDependency{}, false
	}
	primary := filepath.Join(dir, name)
	if _, err := os.Stat(primary); err != nil {
		return pluginport.ResolvedDependency{}, false
	}
	return newResolved(req, dir, primary), true
}

func readyStatus(req pluginport.DependencyRequirement, root, primary string) pluginport.DependencyStatus {
	resolved := newResolved(req, root, primary)
	return pluginport.DependencyStatus{ID: req.ID, State: pluginport.DependencyStateReady, ProgressPercent: 100, Resolved: &resolved}
}

// newResolved is the dependency at primary, with the variable a process finds it by.
func newResolved(req pluginport.DependencyRequirement, root, primary string) pluginport.ResolvedDependency {
	return pluginport.ResolvedDependency{
		ID: req.ID, RootPath: root, PrimaryPath: primary,
		EnvVars:  map[string]string{GetEnvName(req): primary},
		Metadata: req.Metadata,
	}
}

// GetEnvName is the variable that holds the path of the dependency: the
// requirement's envVar metadata, or NUMEN_DEPENDENCY_<ID>.
func GetEnvName(req pluginport.DependencyRequirement) string {
	if named := req.Metadata["envVar"]; named != "" {
		return named
	}
	return "NUMEN_DEPENDENCY_" + strings.ToUpper(unsafeNameChars.ReplaceAllString(req.ID, "_"))
}
