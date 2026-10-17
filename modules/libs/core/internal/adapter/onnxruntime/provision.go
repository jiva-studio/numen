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
	// Dir is the models home the library is kept under. Empty is GetHome.
	Dir string
	// Provider names the execution provider. Empty is the CPU.
	Provider string
	// Offline forbids fetching the archive.
	Offline bool
	// Release is the pinned archive. Zero is the release published for this platform.
	Release Release
	// Client fetches the archive. Nil follows a redirect only to https.
	Client *http.Client
}

// Provisioner makes the library ready once and hands the same answer to every
// later caller.
type Provisioner struct {
	cfg      Config
	gate     chan struct{}
	ready    bool
	library  string
	provider Provider
}

// New returns a Provisioner for the configuration.
func New(cfg Config) *Provisioner {
	return &Provisioner{cfg: cfg, gate: make(chan struct{}, 1)}
}

// Provision returns the path of the library and the name of the execution
// provider, fetching and unpacking the library when the models home does not
// hold it. A caller that arrives while another is provisioning waits for it,
// against its own context. A failure is returned to the caller that met it and
// the next call tries again.
func (p *Provisioner) Provision(ctx context.Context) (string, string, error) {
	select {
	case p.gate <- struct{}{}:
	case <-ctx.Done():
		return "", "", ctx.Err()
	}
	defer func() { <-p.gate }()
	if p.ready {
		return p.library, string(p.provider), nil
	}
	provider, err := ParseProvider(p.cfg.Provider)
	if err != nil {
		return "", "", err
	}
	if err := provider.check(runtime.GOOS, runtime.GOARCH); err != nil {
		return "", "", err
	}
	library, err := p.acquire(ctx)
	if err != nil {
		return "", "", fmt.Errorf("the onnx runtime: %w", err)
	}
	p.library, p.provider, p.ready = library, provider, true
	return library, string(provider), nil
}

// acquire returns the path of the library, verified against its pinned sum.
func (p *Provisioner) acquire(ctx context.Context) (string, error) {
	release := p.cfg.Release
	if release == (Release{}) {
		var err error
		if release, err = getRelease(runtime.GOOS, runtime.GOARCH); err != nil {
			return "", err
		}
	}
	if err := release.check(); err != nil {
		return "", err
	}
	home := p.cfg.Dir
	if home == "" {
		var err error
		if home, err = GetHome(); err != nil {
			return "", err
		}
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
