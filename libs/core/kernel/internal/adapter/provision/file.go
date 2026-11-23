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
	"runtime"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// ErrNoFolder is a dependency asked for where no folder is set to keep dependencies in.
var ErrNoFolder = errors.New("no folder to keep dependencies in")

// ErrOffline is a dependency that is not on this machine while fetching is forbidden.
var ErrOffline = errors.New("fetching is forbidden")

// Config is what dependencies are provisioned from. Getenv reads the host variables a search
// pattern names; nil reads the environment of this process.
type Config struct {
	Dir       string
	Offline   bool
	Client    *http.Client
	Discovery discovery
	Backoff   backoff
	Clock     machineport.Clock
	GetEnv    func(string) string
}

// FileProvisioner provisions a dependency that is one downloadable file with a
// pinned checksum, or one file unpacked from a downloaded archive: it looks on the
// machine first, then in its own folder, and fetches last.
type FileProvisioner struct {
	cfg Config
}

func NewFileProvisioner(cfg Config) *FileProvisioner {
	if cfg.Client == nil {
		cfg.Client = newClient()
	}
	if cfg.Discovery.dirs == nil && cfg.Discovery.runLdconfig == nil {
		cfg.Discovery = newDiscovery()
	}
	if cfg.Backoff.attempts == 0 {
		cfg.Backoff = newDefaultBackoff()
	}
	if cfg.GetEnv == nil {
		cfg.GetEnv = os.Getenv
	}
	return &FileProvisioner{cfg: cfg}
}

// CanProvision reports whether the requirement names a file to fetch, or a
// program or library the machine may already hold.
func (p *FileProvisioner) CanProvision(req manifest.DependencyRequirement) bool {
	return req.SourceURL != "" || len(req.SourceURLs) > 0 || len(req.Sources) > 0 || req.Kind == manifest.DependencyKindBinary || req.Kind == manifest.DependencyKindRuntime
}

func (p *FileProvisioner) getDir(req manifest.DependencyRequirement) (string, error) {
	root := p.cfg.Dir
	if root == "" {
		return "", ErrNoFolder
	}
	return getDependencyDir(root, string(req.Kind), req.ID, req.Version), nil
}

// Provision returns the dependency: the one the search patterns find, the one the
// machine holds, the one in the folder, or the one it fetches for this platform.
// Processes provisioning the same dependency take turns.
func (p *FileProvisioner) Provision(
	ctx context.Context, req manifest.DependencyRequirement, onProgress func(plugin.DependencyStatus),
) (_ plugin.ResolvedDependency, err error) {
	if onProgress == nil {
		onProgress = func(plugin.DependencyStatus) {}
	}
	if found := findInSearch(req.Search, p.cfg.GetEnv); found != "" {
		return newResolvedDependency(req, filepath.Dir(found), found), nil
	}
	if found := p.cfg.Discovery.find(req); found != "" {
		return newResolvedDependency(req, filepath.Dir(found), found), nil
	}
	dir, err := p.getDir(req)
	if err != nil {
		return plugin.ResolvedDependency{}, err
	}
	source, ok := req.GetSource(runtime.GOOS, runtime.GOARCH)
	if !ok {
		return plugin.ResolvedDependency{}, fmt.Errorf("%s %s is not on this machine and names no source for %s/%s", req.Kind, req.ID, runtime.GOOS, runtime.GOARCH)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return plugin.ResolvedDependency{}, err
	}
	release, err := lockFile(ctx, dir+".lock")
	if err != nil {
		return plugin.ResolvedDependency{}, err
	}
	defer func() { err = errors.Join(err, release()) }()

	sum := getReadySum(source)
	if resolved, ok := p.getCachedFile(req, dir, sum); ok {
		return resolved, nil
	}
	if p.cfg.Offline {
		return plugin.ResolvedDependency{}, fmt.Errorf("%s %s: %w", req.Kind, req.ID, ErrOffline)
	}
	return p.store(ctx, req, source, dir, onProgress)
}

func (p *FileProvisioner) store(
	ctx context.Context, req manifest.DependencyRequirement, source manifest.DependencySource, dir string, onProgress func(plugin.DependencyStatus),
) (plugin.ResolvedDependency, error) {
	sum := getReadySum(source)
	primary, actual, err := p.fetch(ctx, req, source, dir, onProgress)
	if err != nil {
		return plugin.ResolvedDependency{}, err
	}
	if source.ChecksumSHA256 == "" {
		return p.storeModelFile(ctx, req, dir, primary, actual)
	}
	primary, err = prepareFetched(ctx, req, source, dir, primary)
	if err != nil {
		return plugin.ResolvedDependency{}, err
	}
	if err := writeReadyMarker(ctx, dir, filepath.Base(primary), sum); err != nil {
		return plugin.ResolvedDependency{}, err
	}
	return newResolvedDependency(req, dir, primary), nil
}

func prepareFetched(ctx context.Context, req manifest.DependencyRequirement, source manifest.DependencySource, dir, primary string) (string, error) {
	if source.ArchiveMember != "" {
		member, err := unpackMember(ctx, primary, source.ArchiveMember, source.MemberSHA256, dir)
		return member, errors.Join(err, os.Remove(primary))
	}
	if req.Kind == manifest.DependencyKindBinary {
		return primary, os.Chmod(primary, 0o755)
	}
	return primary, nil
}

func (p *FileProvisioner) fetch(
	ctx context.Context, req manifest.DependencyRequirement, source manifest.DependencySource, dir string, onProgress func(plugin.DependencyStatus),
) (string, string, error) {
	name, err := getFileName(req, source)
	if err != nil {
		return "", "", err
	}
	primary := filepath.Join(dir, name)
	title := "Downloading " + req.GetTitle()
	var actual string
	var errs []error
	for _, address := range source.GetURLs() {
		err = transfer{
			handleDigest: func(sum string) { actual = sum },
			client:   p.cfg.Client, address: address, dest: primary, sum: source.ChecksumSHA256, backoff: p.cfg.Backoff, clock: p.cfg.Clock,
			handleProgress: func(done, total int64) {
				onProgress(plugin.DependencyStatus{
					ID: req.ID, State: plugin.DependencyStateProvisioning, BytesDownloaded: done, BytesTotal: total,
					ProgressPercent: getPercent(done, total), Message: title,
				})
			},
		}.fetch(ctx)
		if err == nil {
			return primary, actual, nil
		}
		errs = append(errs, err)
		if ctx.Err() != nil {
			break
		}
	}
	return primary, actual, errors.Join(errs...)
}

func getReadySum(source manifest.DependencySource) string {
	if source.ArchiveMember != "" {
		return source.MemberSHA256
	}
	return source.ChecksumSHA256
}

func getPercent(done, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(done) * 100 / float64(total)
}

func getFileName(req manifest.DependencyRequirement, source manifest.DependencySource) (string, error) {
	name, err := pickFileName(req, source)
	if err != nil {
		return "", err
	}
	if isReserved(name) {
		return "", fmt.Errorf("%s %s: %q is a name the folder keeps for itself", req.Kind, req.ID, name)
	}
	if isBareProgramName(req, source, name) {
		name += ".exe"
	}
	return name, nil
}

func isBareProgramName(req manifest.DependencyRequirement, source manifest.DependencySource, name string) bool {
	return runtime.GOOS == windowsOS && req.Kind == manifest.DependencyKindBinary && source.ArchiveMember == "" &&
		!strings.HasSuffix(strings.ToLower(name), ".exe")
}

func pickFileName(req manifest.DependencyRequirement, source manifest.DependencySource) (string, error) {
	if named := req.Metadata["fileName"]; named != "" && source.ArchiveMember == "" {
		return cleanName(named), nil
	}
	address, err := url.Parse(source.SourceURL)
	if err != nil {
		return "", err
	}
	if base := path.Base(address.Path); base != "" && base != "." && base != "/" {
		return cleanName(base), nil
	}
	return cleanName(req.ID), nil
}

func readCachedDependency(req manifest.DependencyRequirement, dir, sum string) (plugin.ResolvedDependency, bool) {
	if !isReady(dir) {
		return plugin.ResolvedDependency{}, false
	}
	name, found, err := readReadyMarker(dir)
	if err != nil || !isPlainName(name) || (sum != "" && found != sum) || found == "" {
		return plugin.ResolvedDependency{}, false
	}
	primary := filepath.Join(dir, name)
	if _, err := os.Stat(primary); err != nil {
		return plugin.ResolvedDependency{}, false
	}
	return newResolvedDependency(req, dir, primary), true
}

func newResolvedDependency(req manifest.DependencyRequirement, root, primary string) plugin.ResolvedDependency {
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
	return "NUMEN_DEPENDENCY_" + strings.ToUpper(nonEnvNameChars.ReplaceAllString(req.ID, "_"))
}

// Resolve returns where the dependency is or will be, and whether it is ready there. It reads the
// machine and the folder only: it takes no lock, fetches nothing and leaves the errors of a missing
// source, an offline machine and a failed download to Provision.
func (p *FileProvisioner) Resolve(req manifest.DependencyRequirement) (plugin.ResolvedDependency, plugin.DependencyState, error) {
	if found := findInSearch(req.Search, p.cfg.GetEnv); found != "" {
		return newResolvedDependency(req, filepath.Dir(found), found), plugin.DependencyStateReady, nil
	}
	if found := p.cfg.Discovery.find(req); found != "" {
		return newResolvedDependency(req, filepath.Dir(found), found), plugin.DependencyStateReady, nil
	}
	dir, err := p.getDir(req)
	if err != nil {
		return plugin.ResolvedDependency{}, "", err
	}
	source, ok := req.GetSource(runtime.GOOS, runtime.GOARCH)
	if !ok {
		return newResolvedDependency(req, dir, filepath.Join(dir, cleanName(req.ID))), plugin.DependencyStateMissing, nil
	}
	if resolved, ok := p.getCachedFile(req, dir, getReadySum(source)); ok {
		return resolved, plugin.DependencyStateReady, nil
	}
	name, err := getFileName(req, source)
	if err != nil {
		return plugin.ResolvedDependency{}, "", err
	}
	if source.ArchiveMember != "" {
		name = cleanName(path.Base(source.ArchiveMember))
	}
	return newResolvedDependency(req, dir, filepath.Join(dir, name)), plugin.DependencyStateMissing, nil
}
