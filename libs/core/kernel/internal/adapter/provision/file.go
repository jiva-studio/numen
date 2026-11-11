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

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// ErrNoFolder is a dependency asked for where no folder is set to keep dependencies in.
var ErrNoFolder = errors.New("no folder to keep dependencies in")

// ErrOffline is a dependency that is not on this machine while fetching is forbidden.
var ErrOffline = errors.New("fetching is forbidden")

// Config is what dependencies are provisioned from.
type Config struct {
	Dir       string
	Offline   bool
	Client    *http.Client
	Discovery discovery
	Backoff   backoff
	Clock     machineport.Clock
}

// FileProvisioner provisions a dependency that is one downloadable file with a
// pinned checksum: it looks on the machine first, then in its own folder, and
// fetches last.
type FileProvisioner struct {
	cfg Config
}

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
func (p *FileProvisioner) CanProvision(req manifest.DependencyRequirement) bool {
	return req.SourceURL != "" || req.Kind == manifest.DependencyKindBinary || req.Kind == manifest.DependencyKindRuntime
}

func (p *FileProvisioner) getDir(req manifest.DependencyRequirement) (string, error) {
	root := p.cfg.Dir
	if root == "" {
		return "", ErrNoFolder
	}
	return getDependencyDir(root, string(req.Kind), req.ID, req.Version), nil
}

// Status returns whether the machine or the folder holds the dependency.
func (p *FileProvisioner) Status(_ context.Context, req manifest.DependencyRequirement) (plugin.DependencyStatus, error) {
	if found := p.cfg.Discovery.find(req); found != "" {
		return readyStatus(req, filepath.Dir(found), found), nil
	}
	dir, err := p.getDir(req)
	if err != nil {
		return plugin.DependencyStatus{}, err
	}
	if resolved, ok := readCached(req, dir); ok {
		return plugin.DependencyStatus{ID: req.ID, State: plugin.DependencyStateReady, ProgressPercent: 100, Resolved: &resolved}, nil
	}
	return plugin.DependencyStatus{ID: req.ID, State: plugin.DependencyStateMissing}, nil
}

// Provision returns the dependency: the one the machine holds, the one in the
// folder, or the one it fetches. Processes provisioning the same dependency
// take turns.
func (p *FileProvisioner) Provision(
	ctx context.Context, req manifest.DependencyRequirement, onProgress func(plugin.DependencyStatus),
) (_ plugin.ResolvedDependency, err error) {
	if onProgress == nil {
		onProgress = func(plugin.DependencyStatus) {}
	}
	if found := p.cfg.Discovery.find(req); found != "" {
		return newResolved(req, filepath.Dir(found), found), nil
	}
	dir, err := p.getDir(req)
	if err != nil {
		return plugin.ResolvedDependency{}, err
	}
	if req.SourceURL == "" {
		return plugin.ResolvedDependency{}, fmt.Errorf("%s %s is not on this machine and names no sourceUrl", req.Kind, req.ID)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return plugin.ResolvedDependency{}, err
	}
	release, err := lockFile(ctx, dir+".lock")
	if err != nil {
		return plugin.ResolvedDependency{}, err
	}
	defer func() { err = errors.Join(err, release()) }()

	if resolved, ok := readCached(req, dir); ok {
		return resolved, nil
	}
	if p.cfg.Offline {
		return plugin.ResolvedDependency{}, fmt.Errorf("%s %s: %w", req.Kind, req.ID, ErrOffline)
	}
	name, err := getFileName(req)
	if err != nil {
		return plugin.ResolvedDependency{}, err
	}
	primary := filepath.Join(dir, name)
	title := "Downloading " + req.ID
	err = transfer{
		client: p.cfg.Client, address: req.SourceURL, dest: primary, sum: req.ChecksumSHA256, backoff: p.cfg.Backoff, clock: p.cfg.Clock,
		onProgress: func(done, total int64) {
			onProgress(plugin.DependencyStatus{
				ID: req.ID, State: plugin.DependencyStateProvisioning, BytesDownloaded: done, BytesTotal: total,
				ProgressPercent: getPercent(done, total), Message: title,
			})
		},
	}.fetch(ctx)
	if err != nil {
		return plugin.ResolvedDependency{}, err
	}
	if err := writeReady(ctx, dir, name, req.ChecksumSHA256); err != nil {
		return plugin.ResolvedDependency{}, err
	}
	return newResolved(req, dir, primary), nil
}

func getPercent(done, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(done) * 100 / float64(total)
}

func getFileName(req manifest.DependencyRequirement) (string, error) {
	name, err := pickFileName(req)
	if err != nil {
		return "", err
	}
	if isReserved(name) {
		return "", fmt.Errorf("%s %s: %q is a name the folder keeps for itself", req.Kind, req.ID, name)
	}
	return name, nil
}

func pickFileName(req manifest.DependencyRequirement) (string, error) {
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

func readCached(req manifest.DependencyRequirement, dir string) (plugin.ResolvedDependency, bool) {
	if !isReady(dir) {
		return plugin.ResolvedDependency{}, false
	}
	name, sum, err := readReady(dir)
	if err != nil || !isPlainName(name) || sum != req.ChecksumSHA256 {
		return plugin.ResolvedDependency{}, false
	}
	primary := filepath.Join(dir, name)
	if _, err := os.Stat(primary); err != nil {
		return plugin.ResolvedDependency{}, false
	}
	return newResolved(req, dir, primary), true
}

func readyStatus(req manifest.DependencyRequirement, root, primary string) plugin.DependencyStatus {
	resolved := newResolved(req, root, primary)
	return plugin.DependencyStatus{ID: req.ID, State: plugin.DependencyStateReady, ProgressPercent: 100, Resolved: &resolved}
}

func newResolved(req manifest.DependencyRequirement, root, primary string) plugin.ResolvedDependency {
	return plugin.ResolvedDependency{
		ID: req.ID, RootPath: root, PrimaryPath: primary,
		EnvVars:  map[string]string{GetEnvName(req): primary},
		Metadata: req.Metadata,
	}
}

// GetEnvName is the variable that holds the path of the dependency: the
// requirement's envVar metadata, or NUMEN_DEPENDENCY_<ID>.
func GetEnvName(req manifest.DependencyRequirement) string {
	if named := req.Metadata["envVar"]; named != "" {
		return named
	}
	return "NUMEN_DEPENDENCY_" + strings.ToUpper(unsafeNameChars.ReplaceAllString(req.ID, "_"))
}
