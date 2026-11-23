package provision

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestResolveReturnsTheFolderPathOfAMissingDependencyWithoutDownloading(t *testing.T) {
	data := bytes.Repeat([]byte("model-bytes"), 100)
	server, requests := serveFiles(t, map[string][]byte{"/w.bin": data})
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = server.Client()
	req := newModelRequirement(server.URL+"/w.bin", data)

	resolved, state, err := provisioner.Resolve(req)

	if err != nil {
		t.Fatal(err)
	}
	if state != plugin.DependencyStateMissing {
		t.Errorf("state = %q, want missing", state)
	}
	dir, dirErr := provisioner.getDir(req)
	if dirErr != nil {
		t.Fatal(dirErr)
	}
	if want := filepath.Join(dir, "w.bin"); resolved.PrimaryPath != want {
		t.Errorf("planned path = %q, want %q", resolved.PrimaryPath, want)
	}
	if resolved.EnvVars[GetEnvName(req)] != resolved.PrimaryPath {
		t.Errorf("env = %v, want %s to hold the planned path", resolved.EnvVars, GetEnvName(req))
	}
	if got := requests.Load(); got != 0 {
		t.Errorf("Resolve made %d requests, want none", got)
	}
	if _, statErr := os.Stat(resolved.PrimaryPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("Resolve left a file at the planned path: %v", statErr)
	}
}

func TestResolveReturnsTheFileOfADownloadedDependencyAsReady(t *testing.T) {
	data := bytes.Repeat([]byte("model-bytes"), 100)
	server, requests := serveFiles(t, map[string][]byte{"/w.bin": data})
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = server.Client()
	req := newModelRequirement(server.URL+"/w.bin", data)
	provisioned, err := provisioner.Provision(t.Context(), req, nil)
	if err != nil {
		t.Fatal(err)
	}
	before := requests.Load()

	resolved, state, err := provisioner.Resolve(req)

	if err != nil {
		t.Fatal(err)
	}
	if state != plugin.DependencyStateReady {
		t.Errorf("state = %q, want ready", state)
	}
	if resolved.PrimaryPath != provisioned.PrimaryPath || resolved.RootPath != provisioned.RootPath {
		t.Errorf("resolved %+v, want the paths of %+v", resolved, provisioned)
	}
	if resolved.EnvVars[GetEnvName(req)] != provisioned.PrimaryPath {
		t.Errorf("env = %v", resolved.EnvVars)
	}
	if got := requests.Load(); got != before {
		t.Errorf("Resolve made %d requests", got-before)
	}
}

func TestResolveReturnsThePlannedPathThatProvisionFillsLater(t *testing.T) {
	data := bytes.Repeat([]byte("model-bytes"), 100)
	server, _ := serveFiles(t, map[string][]byte{"/w.bin": data})
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = server.Client()
	req := newModelRequirement(server.URL+"/w.bin", data)

	planned, _, err := provisioner.Resolve(req)
	if err != nil {
		t.Fatal(err)
	}
	provisioned, err := provisioner.Provision(t.Context(), req, nil)
	if err != nil {
		t.Fatal(err)
	}

	if planned.PrimaryPath == "" || planned.PrimaryPath != provisioned.PrimaryPath {
		t.Errorf("planned %q, provisioned %q", planned.PrimaryPath, provisioned.PrimaryPath)
	}
}

func TestResolveReturnsThePlannedPathOfAFileUnpackedFromAnArchive(t *testing.T) {
	library := []byte("library bytes")
	archive := buildTgz(t, map[string][]byte{"pack/libsample.so": library})
	server, requests := serveFiles(t, map[string][]byte{"/pack.tgz": archive})
	req := newRuntimeRequirement([]manifest.DependencySource{{
		Platform: getPlatform(), SourceURL: server.URL + "/pack.tgz", ChecksumSHA256: sumOf(archive),
		ArchiveMember: "pack/libsample.so", MemberSHA256: sumOf(library),
	}})
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = server.Client()

	planned, state, err := provisioner.Resolve(req)
	if err != nil {
		t.Fatal(err)
	}
	if state != plugin.DependencyStateMissing || requests.Load() != 0 {
		t.Fatalf("state %q after %d requests, want missing and none", state, requests.Load())
	}
	provisioned, err := provisioner.Provision(t.Context(), req, nil)
	if err != nil {
		t.Fatal(err)
	}

	if planned.PrimaryPath == "" || planned.PrimaryPath != provisioned.PrimaryPath {
		t.Errorf("planned %q, provisioned %q", planned.PrimaryPath, provisioned.PrimaryPath)
	}
}

func TestResolveReturnsTheFileTheSearchPatternFindsAsReady(t *testing.T) {
	found := filepath.Join(t.TempDir(), "libsample.so")
	if err := os.WriteFile(found, []byte("library"), 0o644); err != nil {
		t.Fatal(err)
	}
	server, requests := serveFiles(t, nil)
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = server.Client()
	provisioner.cfg.GetEnv = func(name string) string {
		if name == "SAMPLE_HOME" {
			return filepath.Dir(found)
		}
		return ""
	}
	req := newRuntimeRequirement([]manifest.DependencySource{{Platform: getPlatform(), SourceURL: server.URL + "/lib.so", ChecksumSHA256: sumOf([]byte("x"))}})
	req.Search = []string{"${SAMPLE_HOME}/libsample.so"}

	resolved, state, err := provisioner.Resolve(req)

	if err != nil {
		t.Fatal(err)
	}
	if state != plugin.DependencyStateReady || resolved.PrimaryPath != found {
		t.Errorf("resolved %q as %q, want %q ready", resolved.PrimaryPath, state, found)
	}
	if resolved.EnvVars["SAMPLE_RUNTIME_LIB"] != found || requests.Load() != 0 {
		t.Errorf("env %v after %d requests", resolved.EnvVars, requests.Load())
	}
}

func TestResolveDoesNotWaitForTheLockOfARunningDownload(t *testing.T) {
	data := []byte("model-bytes")
	provisioner := newTestProvisioner(t, false)
	req := newModelRequirement("https://models.invalid/w.bin", data)
	dir, err := provisioner.getDir(req)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	release, err := lockFile(t.Context(), dir+".lock")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := release(); err != nil {
			t.Error(err)
		}
	})
	done := make(chan error, 1)

	go func() {
		_, _, resolveErr := provisioner.Resolve(req)
		done <- resolveErr
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Resolve = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Resolve waited for the lock of a running download")
	}
}

func TestResolveDoesNotFailForADependencyOfAnotherPlatform(t *testing.T) {
	req := newRuntimeRequirement([]manifest.DependencySource{
		{Platform: "plan9/mips64", SourceURL: "https://models.invalid/there.so", ChecksumSHA256: sumOf([]byte("x"))},
	})

	_, state, err := newTestProvisioner(t, false).Resolve(req)

	if err != nil {
		t.Fatalf("Resolve = %v, want the platform error left to the first use", err)
	}
	if state != plugin.DependencyStateMissing {
		t.Errorf("state = %q, want missing", state)
	}
}

func TestResolveDoesNotFailOnAnOfflineMachine(t *testing.T) {
	provisioner := newTestProvisioner(t, true)

	resolved, state, err := provisioner.Resolve(newModelRequirement("https://models.invalid/w.bin", []byte("x")))

	if err != nil {
		t.Fatalf("Resolve = %v, want the offline error left to the first use", err)
	}
	if state != plugin.DependencyStateMissing || resolved.PrimaryPath == "" {
		t.Errorf("resolved %+v as %q, want a planned path and missing", resolved, state)
	}
}

func TestResolveReturnsMissingForAFolderWithoutItsReadyMarker(t *testing.T) {
	data := []byte("model-bytes")
	provisioner := newTestProvisioner(t, false)
	req := newModelRequirement("https://models.invalid/w.bin", data)
	dir, err := provisioner.getDir(req)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "w.bin"), data[:4], 0o644); err != nil {
		t.Fatal(err)
	}

	_, state, err := provisioner.Resolve(req)

	if err != nil {
		t.Fatal(err)
	}
	if state != plugin.DependencyStateMissing {
		t.Errorf("state = %q, want missing for a partial file", state)
	}
}

func TestPlaceholderResolveReturnsItsPlaceholderFileAsReady(t *testing.T) {
	provisioner := NewPlaceholderProvisioner(t.TempDir())
	req := manifest.DependencyRequirement{ID: "pp-det", Kind: manifest.DependencyKindModel, SourceURL: "https://models.invalid/det.onnx"}

	resolved, state, err := provisioner.Resolve(req)

	if err != nil {
		t.Fatal(err)
	}
	if state != plugin.DependencyStateReady {
		t.Errorf("state = %q, want ready", state)
	}
	provisioned, err := provisioner.Provision(t.Context(), req, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.PrimaryPath == "" || resolved.PrimaryPath != provisioned.PrimaryPath {
		t.Errorf("resolved %q, provisioned %q", resolved.PrimaryPath, provisioned.PrimaryPath)
	}
	if resolved.EnvVars[GetEnvName(req)] != resolved.PrimaryPath {
		t.Errorf("env = %v", resolved.EnvVars)
	}
}

func TestRegistryResolveAsksTheRoutedProvisionerWithoutRunningIt(t *testing.T) {
	models := &fakeProvisioner{kind: manifest.DependencyKindModel}
	other := &fakeProvisioner{kind: manifest.DependencyKindDataset}
	registry := NewObservedRegistry(nil, other, models)
	req := manifest.DependencyRequirement{ID: "m", Kind: manifest.DependencyKindModel}

	resolved, state, err := registry.Resolve(req)

	if err != nil {
		t.Fatal(err)
	}
	if resolved.PrimaryPath != "/planned/m" || state != plugin.DependencyStateMissing {
		t.Errorf("resolved %+v as %q, want what the model provisioner planned", resolved, state)
	}
	if models.calls.Load() != 0 || other.calls.Load() != 0 {
		t.Errorf("Resolve ran a download: models %d, other %d", models.calls.Load(), other.calls.Load())
	}
	if models.resolves.Load() != 1 || other.resolves.Load() != 0 {
		t.Errorf("resolves: models %d, other %d, want 1 and 0", models.resolves.Load(), other.resolves.Load())
	}
}

func TestRegistryResolveFailsWhereNothingProvisionsTheKind(t *testing.T) {
	registry := NewObservedRegistry(nil, &fakeProvisioner{kind: manifest.DependencyKindModel})

	_, _, err := registry.Resolve(manifest.DependencyRequirement{ID: "d", Kind: manifest.DependencyKindDataset})

	if !errors.Is(err, plugin.ErrNoProvisioner) {
		t.Errorf("err = %v, want ErrNoProvisioner", err)
	}
}

func TestRegistryResolveDoesNotWaitForARunningDownload(t *testing.T) {
	provisioner := &fakeProvisioner{kind: manifest.DependencyKindModel, gate: make(chan struct{})}
	registry := NewObservedRegistry(nil, provisioner)
	req := manifest.DependencyRequirement{ID: "m", Kind: manifest.DependencyKindModel, Version: "1"}
	downloading := make(chan error, 1)
	go func() {
		_, err := registry.Provision(context.WithoutCancel(t.Context()), req, nil)
		downloading <- err
	}()
	for !registry.flights.isRunning(req.GetIdentity()) {
		time.Sleep(time.Millisecond)
	}
	resolved := make(chan error, 1)

	go func() {
		_, _, err := registry.Resolve(req)
		resolved <- err
	}()

	select {
	case err := <-resolved:
		if err != nil {
			t.Errorf("Resolve = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Resolve waited for the download in progress")
	}
	close(provisioner.gate)
	if err := <-downloading; err != nil {
		t.Error(err)
	}
}

func TestResolvedEnvironmentNamesOnlyTheDependencyVariable(t *testing.T) {
	provisioner := newTestProvisioner(t, false)
	req := newModelRequirement("https://models.invalid/w.bin", []byte("x"))

	resolved, _, err := provisioner.Resolve(req)

	if err != nil {
		t.Fatal(err)
	}
	for name := range resolved.EnvVars {
		if !strings.HasPrefix(name, "NUMEN_DEPENDENCY_") {
			t.Errorf("Resolve sets %s", name)
		}
	}
}
