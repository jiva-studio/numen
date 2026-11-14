package onnxruntime

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// ErrOffline is a runtime that is not on this machine while fetching is forbidden.
var ErrOffline = errors.New("fetching is forbidden")

// Config is what the runtime is provisioned from.
type Config struct {
	Dir      string
	Provider string
	Offline  bool
	Release  Release
	Client   *http.Client
	Machine  []string
	Check    func(targetPath string) error
	Clock    machineport.Clock
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

// Provision returns the path of the library and the name of the execution provider.
// The library is, in order: the one named in the environment, used as given;
// the first one this machine holds that opens; and last the pinned release, fetched.
// A caller that arrives while another is provisioning waits for it, against its
// own context. A failure goes to the caller that met it; the next call tries again.
func (p *Provisioner) Provision(ctx context.Context, pluginDir string) (string, string, error) {
	return p.ProvisionWithProgress(ctx, pluginDir, nil)
}

// ProvisionWithProgress is Provision that reports to onProgress how many bytes of the
// archive have been received.
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
			provider, getLibraryEnvs()[0], found.describeSkipped())
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

func (l location) describeSkipped() string {
	if len(l.skipped) == 0 {
		return ""
	}
	return "\nthe libraries this machine holds that do not open:\n  " + strings.Join(l.skipped, "\n  ")
}

func (p *Provisioner) locate(ctx context.Context, pluginDir string) (location, error) {
	if named := getEnvLibrary(); named != "" {
		return location{library: named}, nil
	}
	release := p.cfg.Release
	var releaseErr error
	if release == (Release{}) {
		release, releaseErr = getRelease(runtime.GOOS, runtime.GOARCH)
	}
	home := p.cfg.Dir
	var homeErr error
	if home == "" {
		home, homeErr = GetHome()
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
	found.isPinned = true
	if err := errors.Join(releaseErr, homeErr); err != nil {
		return found, err
	}
	library, err := p.acquire(ctx, release, home)
	found.library = library
	return found, err
}

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
		return "", errors.Join(err, os.Remove(library))
	}
	return library, nil
}

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
		return fmt.Errorf("%s is not on this machine: %w", release.Address, ErrOffline)
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
