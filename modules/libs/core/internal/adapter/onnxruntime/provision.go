package onnxruntime

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Config is what the runtime is provisioned from.
type Config struct {
	// Dir is the models home the downloaded library is kept under. Empty is GetHome.
	Dir string
	// Provider names the execution provider. Empty is auto.
	Provider string
	// Offline forbids fetching the archive.
	Offline bool
	// Release is the pinned archive. Zero is the release published for this platform.
	Release Release
	// Client fetches the archive. Nil follows a redirect only to https.
	Client *http.Client
	// Machine lists the folders shared libraries are searched in. Nil is the
	// folders this platform keeps them in.
	Machine []string
	// Check returns an error when this machine cannot open the library at a
	// path. Nil opens it with the platform loader and asks it for the API the
	// plugins call.
	Check func(targetPath string) error
}

// Provisioner finds or fetches the library once per plugin folder and hands the
// same answer to every later caller.
type Provisioner struct {
	cfg      Config
	gate     chan struct{}
	ready    map[string]answer
	progress func(done, total int64)
}

// answer is the library a plugin folder was given, and the provider.
type answer struct {
	library  string
	provider Provider
}

// New returns a Provisioner for the configuration.
func New(cfg Config) *Provisioner {
	return &Provisioner{cfg: cfg, gate: make(chan struct{}, 1), ready: make(map[string]answer)}
}

// Provision returns the path of the library and the name of the execution
//
// The library is, in order: the one a person names in the environment, used as
// given; the first one this machine holds that opens (beside the plugin or the
// host, in the folders shared libraries live in, in the package store, in the
// folders the platform loader searches); and last the pinned release, fetched
// is the CPU build. A caller that arrives while another is provisioning waits
// for it, against its own context. A failure is returned to the caller that met
// it and the next call tries again.
func (p *Provisioner) Provision(ctx context.Context, pluginDir string) (string, string, error) {
	return p.ProvisionWithProgress(ctx, pluginDir, nil)
}

// ProvisionWithProgress is Provision that tells onProgress how many bytes of the
// archive have arrived.
func (p *Provisioner) ProvisionWithProgress(ctx context.Context, pluginDir string, onProgress func(done, total int64)) (string, string, error) {
	select {
	case p.gate <- struct{}{}:
	case <-ctx.Done():
		return "", "", ctx.Err()
	}
	defer func() { <-p.gate }()
	p.progress = onProgress
	defer func() { p.progress = nil }()
	if found, ok := p.ready[pluginDir]; ok {
		return found.library, string(found.provider), nil
	}
	provider, err := ParseProvider(p.cfg.Provider)
	if err != nil {
		return "", "", err
	}
	found, err := p.locate(ctx, pluginDir)
	if err != nil {
		return "", "", fmt.Errorf("the onnx runtime: %w%s", err, found.describeSkipped())
	}
	if found.isPinned && provider.isGPU() {
		return "", "", fmt.Errorf("the onnx runtime: the %s provider needs a GPU build, and this machine holds none that opens: install one and name it in %s, or use the cpu provider%s",
			provider, libraryEnvs[0], found.describeSkipped())
	}
	p.ready[pluginDir] = answer{library: found.library, provider: provider}
	return found.library, string(provider), nil
}

// location is the library a search settled on, whether it is the pinned
// release, and why each library the machine holds was passed over.
type location struct {
	library  string
	isPinned bool
	skipped  []string
}

// describeSkipped returns the libraries passed over and why, one per line, or
// empty when none was.
func (l location) describeSkipped() string {
	if len(l.skipped) == 0 {
		return ""
	}
	return "\nthe libraries this machine holds that do not open:\n  " + strings.Join(l.skipped, "\n  ")
}

// locate returns the library: the one named in the environment, the first one
// the machine holds that opens, or the pinned release.
func (p *Provisioner) locate(ctx context.Context, pluginDir string) (location, error) {
	if named := getEnvLibrary(); named != "" {
		return location{library: named}, nil
	}
	release := p.cfg.Release
	if release == (Release{}) {
		release, _ = getRelease(runtime.GOOS, runtime.GOARCH)
	}
	home := p.cfg.Dir
	if home == "" {
		home, _ = GetHome()
	}
	check := p.cfg.Check
	if check == nil {
		check = checkLibrary
	}
	var found location
	for _, item := range getCandidates(pluginDir, home, release.LibrarySum, p.cfg.Machine) {
		if err := check(item); err != nil {
			found.skipped = append(found.skipped, fmt.Sprintf("%s: %v", item, err))
			continue
		}
		found.library = item
		return found, nil
	}
	library, err := p.acquire(ctx, release, home)
	found.library, found.isPinned = library, true
	return found, err
}

// acquire returns the path of the pinned library, verified against its sum.
func (p *Provisioner) acquire(ctx context.Context, release Release, home string) (string, error) {
	if release == (Release{}) {
		_, err := getRelease(runtime.GOOS, runtime.GOARCH)
		return "", err
	}
	if err := release.check(); err != nil {
		return "", err
	}
	if home == "" {
		_, err := GetHome()
		return "", err
	}
	dir := filepath.Join(home, "onnxruntime", release.Sum[:12])
	library := filepath.Join(dir, release.Library)
	if verifyFile(library, release.LibrarySum) == nil {
		return library, nil
	}
	archive := filepath.Join(dir, release.getArchiveName())
	if err := p.fetchArchive(ctx, release, dir, archive); err != nil {
		return "", err
	}
	if err := unpack(archive, release.Library, library); err != nil {
		return "", err
	}
	if err := verifyFile(library, release.LibrarySum); err != nil {
		os.Remove(library)
		return "", err
	}
	return library, nil
}

// fetchArchive leaves the archive at archive holding the pinned sum: it keeps
// one that does, and replaces one that does not unless fetching is forbidden.
func (p *Provisioner) fetchArchive(ctx context.Context, release Release, dir, archive string) error {
	if _, err := os.Stat(archive); err == nil {
		if verifyFile(archive, release.Sum) == nil {
			return nil
		}
		if err := os.Remove(archive); err != nil {
			return err
		}
	}
	if p.cfg.Offline {
		return fmt.Errorf("%s is not on this machine, and %s forbids fetching it", release.Address, EnvOffline)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	client := p.cfg.Client
	if client == nil {
		client = newClient()
	}
	return download(ctx, client, release, archive, p.progress)
}
