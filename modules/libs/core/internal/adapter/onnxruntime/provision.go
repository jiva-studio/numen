package onnxruntime

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
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
}

// Provisioner finds or fetches the library once per plugin folder and hands the
// same answer to every later caller.
type Provisioner struct {
	cfg   Config
	gate  chan struct{}
	ready map[string]answer
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
// provider for the plugin that stands in pluginDir.
//
// The library is, in order: the one a person names in the environment, one this
// machine holds (beside the plugin or the host, in the folders shared libraries
// live in, in the package store), and last the pinned release, fetched into the
// models home. A GPU provider is refused for the pinned release, which is the
// CPU build. A caller that arrives while another is provisioning waits for it,
// against its own context. A failure is returned to the caller that met it and
// the next call tries again.
func (p *Provisioner) Provision(ctx context.Context, pluginDir string) (string, string, error) {
	select {
	case p.gate <- struct{}{}:
	case <-ctx.Done():
		return "", "", ctx.Err()
	}
	defer func() { <-p.gate }()
	if found, ok := p.ready[pluginDir]; ok {
		return found.library, string(found.provider), nil
	}
	provider, err := ParseProvider(p.cfg.Provider)
	if err != nil {
		return "", "", err
	}
	library, isPinned, err := p.locate(ctx, pluginDir)
	if err != nil {
		return "", "", fmt.Errorf("the onnx runtime: %w", err)
	}
	if isPinned && provider.isGPU() {
		return "", "", fmt.Errorf("the onnx runtime: the %s provider needs a GPU build, and this machine holds none: install one and name it in %s, or use the cpu provider",
			provider, libraryEnvs[0])
	}
	p.ready[pluginDir] = answer{library: library, provider: provider}
	return library, string(provider), nil
}

// locate returns the library and whether it is the pinned release.
func (p *Provisioner) locate(ctx context.Context, pluginDir string) (string, bool, error) {
	if named := getEnvLibrary(); named != "" {
		return named, false, nil
	}
	release := p.cfg.Release
	if release == (Release{}) {
		release, _ = getRelease(runtime.GOOS, runtime.GOARCH)
	}
	home := p.cfg.Dir
	if home == "" {
		home, _ = GetHome()
	}
	if at := findOnMachine(pluginDir, home, release.LibrarySum, p.cfg.Machine); at != "" {
		return at, false, nil
	}
	library, err := p.acquire(ctx, release, home)
	return library, true, err
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
	return download(ctx, client, release, archive)
}
