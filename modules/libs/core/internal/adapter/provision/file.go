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
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrOffline is a dependency that is not on this machine while fetching is forbidden.
var ErrOffline = errors.New("fetching is forbidden")

// Config is what dependencies are provisioned from.
type Config struct {
	// Dir is the folder dependencies are kept in. Empty is GetRoot.
	Dir string
	// Offline forbids fetching.
	Offline bool
	// Client fetches the files. Nil follows a redirect only to https.
	Client *http.Client
	// Discovery looks for what the machine holds. Zero searches this platform's folders.
	Discovery discovery
	// Backoff is the waits between attempts. Zero is the default.
	Backoff backoff
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
		cfg.Backoff = defaultBackoff
	}
	return &FileProvisioner{cfg: cfg}
}

// CanProvision reports whether the requirement names a file to fetch, or a
// program or library the machine may already hold.
func (p *FileProvisioner) CanProvision(req port.DependencyRequirement) bool {
	return req.SourceURL != "" || req.Kind == port.DependencyKindBinary || req.Kind == port.DependencyKindRuntime
}

func (p *FileProvisioner) getDir(req port.DependencyRequirement) (string, error) {
	root := p.cfg.Dir
	if root == "" {
		var err error
		if root, err = GetRoot(); err != nil {
			return "", err
		}
	}
	return getDependencyDir(root, string(req.Kind), req.ID, req.Version), nil
}

// Status answers whether the machine or the folder holds the dependency.
func (p *FileProvisioner) Status(_ context.Context, req port.DependencyRequirement) (port.DependencyStatus, error) {
	if found := p.cfg.Discovery.find(req); found != "" {
		return readyStatus(req, filepath.Dir(found), found), nil
	}
	dir, err := p.getDir(req)
	if err != nil {
		return port.DependencyStatus{}, err
	}
	if resolved, ok := readCached(req, dir); ok {
		return port.DependencyStatus{ID: req.ID, State: port.DependencyStateReady, ProgressPercent: 100, Resolved: &resolved}, nil
	}
	return port.DependencyStatus{ID: req.ID, State: port.DependencyStateMissing}, nil
}

// Provision returns the dependency: the one the machine holds, the one in the
// folder, or the one it fetches. Processes provisioning the same dependency
// take turns.
func (p *FileProvisioner) Provision(
	ctx context.Context, req port.DependencyRequirement, onProgress func(port.DependencyStatus),
) (port.ResolvedDependency, error) {
	if onProgress == nil {
		onProgress = func(port.DependencyStatus) {}
	}
	if found := p.cfg.Discovery.find(req); found != "" {
		return newResolved(req, filepath.Dir(found), found), nil
	}
	dir, err := p.getDir(req)
	if err != nil {
		return port.ResolvedDependency{}, err
	}
	if req.SourceURL == "" {
		return port.ResolvedDependency{}, fmt.Errorf("%s %s is not on this machine and names no sourceUrl", req.Kind, req.ID)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return port.ResolvedDependency{}, err
	}
	release, err := lockFile(ctx, dir+".lock")
	if err != nil {
		return port.ResolvedDependency{}, err
	}
	defer func() { _ = release() }()

	if resolved, ok := readCached(req, dir); ok {
		return resolved, nil
	}
	if p.cfg.Offline {
		return port.ResolvedDependency{}, fmt.Errorf("%s %s: %w", req.Kind, req.ID, ErrOffline)
	}
	name, err := getFileName(req)
	if err != nil {
		return port.ResolvedDependency{}, err
	}
	primary := filepath.Join(dir, name)
	title := "Downloading " + req.ID
	err = transfer{
		client: p.cfg.Client, address: req.SourceURL, dest: primary, sum: req.ChecksumSHA256, backoff: p.cfg.Backoff,
		onProgress: func(done, total int64) {
			onProgress(port.DependencyStatus{
				ID: req.ID, State: port.DependencyStateProvisioning, BytesDownloaded: done, BytesTotal: total,
				ProgressPercent: getPercent(done, total), Message: title,
			})
		},
	}.fetch(ctx)
	if err != nil {
		return port.ResolvedDependency{}, err
	}
	if err := writeReady(ctx, dir, name); err != nil {
		return port.ResolvedDependency{}, err
	}
	return newResolved(req, dir, primary), nil
}

func getPercent(done, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(done) * 100 / float64(total)
}

// getFileName returns what the fetched file is called in the dependency's folder.
func getFileName(req port.DependencyRequirement) (string, error) {
	if named := req.Metadata["fileName"]; named != "" {
		return cleanName(named), nil
	}
	address, err := url.Parse(req.SourceURL)
	if err != nil {
		return "", err
	}
	base := path.Base(address.Path)
	if base == "" || base == "." || base == "/" {
		return cleanName(req.ID), nil
	}
	return cleanName(base), nil
}

// readCached returns the dependency a folder with the sentinel holds.
func readCached(req port.DependencyRequirement, dir string) (port.ResolvedDependency, bool) {
	if !isReady(dir) {
		return port.ResolvedDependency{}, false
	}
	name, err := readReady(dir)
	if err != nil || name == "" {
		return port.ResolvedDependency{}, false
	}
	primary := filepath.Join(dir, name)
	if _, err := os.Stat(primary); err != nil {
		return port.ResolvedDependency{}, false
	}
	return newResolved(req, dir, primary), true
}

func readyStatus(req port.DependencyRequirement, root, primary string) port.DependencyStatus {
	resolved := newResolved(req, root, primary)
	return port.DependencyStatus{ID: req.ID, State: port.DependencyStateReady, ProgressPercent: 100, Resolved: &resolved}
}

// newResolved is the dependency at primary, with the variable a process finds it by.
func newResolved(req port.DependencyRequirement, root, primary string) port.ResolvedDependency {
	return port.ResolvedDependency{
		ID: req.ID, RootPath: root, PrimaryPath: primary,
		EnvVars:  map[string]string{GetEnvName(req): primary},
		Metadata: req.Metadata,
	}
}

// GetEnvName is the variable that holds the path of the dependency: the
// requirement's envVar metadata, or NUMEN_DEPENDENCY_<ID>.
func GetEnvName(req port.DependencyRequirement) string {
	if named := req.Metadata["envVar"]; named != "" {
		return named
	}
	return "NUMEN_DEPENDENCY_" + strings.ToUpper(unsafeNameChars.ReplaceAllString(req.ID, "_"))
}
