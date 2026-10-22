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

	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
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
func (p *FileProvisioner) CanProvision(req pluginport.DependencyRequirement) bool {
	return req.SourceURL != "" || req.Kind == pluginport.DependencyKindBinary || req.Kind == pluginport.DependencyKindRuntime
}

func (p *FileProvisioner) getDir(req pluginport.DependencyRequirement) (string, error) {
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
) (pluginport.ResolvedDependency, error) {
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
	defer func() { _ = release() }()

	if resolved, ok := readCached(req, dir); ok {
		return resolved, nil
	}
	if p.cfg.Offline {
		return pluginport.ResolvedDependency{}, fmt.Errorf("%s %s: %w", req.Kind, req.ID, ErrOffline)
	}
	name, err := getFileName(req)
	if err != nil {
		return pluginport.ResolvedDependency{}, err
	}
	primary := filepath.Join(dir, name)
	title := "Downloading " + req.ID
	err = transfer{
		client: p.cfg.Client, address: req.SourceURL, dest: primary, sum: req.ChecksumSHA256, backoff: p.cfg.Backoff,
		onProgress: func(done, total int64) {
			onProgress(pluginport.DependencyStatus{
				ID: req.ID, State: pluginport.DependencyStateProvisioning, BytesDownloaded: done, BytesTotal: total,
				ProgressPercent: getPercent(done, total), Message: title,
			})
		},
	}.fetch(ctx)
	if err != nil {
		return pluginport.ResolvedDependency{}, err
	}
	if err := writeReady(ctx, dir, name, req.ChecksumSHA256); err != nil {
		return pluginport.ResolvedDependency{}, err
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
func getFileName(req pluginport.DependencyRequirement) (string, error) {
	name := cleanName(req.ID)
	if named := req.Metadata["fileName"]; named != "" {
		name = cleanName(named)
	} else {
		address, err := url.Parse(req.SourceURL)
		if err != nil {
			return "", err
		}
		if base := path.Base(address.Path); base != "" && base != "." && base != "/" {
			name = cleanName(base)
		}
	}
	if isReserved(name) {
		return "", fmt.Errorf("%s %s: %q is a name the folder keeps for itself", req.Kind, req.ID, name)
	}
	return name, nil
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
