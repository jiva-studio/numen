package onnxruntime_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/onnxruntime"
)

var libraryBytes = []byte("a shared library")

func sum(raw []byte) string {
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func buildTgz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	zipped := gzip.NewWriter(&out)
	packed := tar.NewWriter(zipped)
	if err := packed.WriteHeader(&tar.Header{Name: "onnxruntime-1.23.0/lib/" + name, Typeflag: tar.TypeReg, Mode: 0o755, Size: int64(len(content))}); err != nil {
		t.Fatal(err)
	}
	if _, err := packed.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := packed.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zipped.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func buildZip(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	packed := zip.NewWriter(&out)
	file, err := packed.Create("onnxruntime-win/lib/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := packed.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// serve answers every request with the body and counts the requests.
func serve(t *testing.T, body []byte) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	return server, &hits
}

func newConfig(server *httptest.Server, archive []byte) onnxruntime.Config {
	return onnxruntime.Config{
		Dir:    "",
		Client: server.Client(),
		Release: onnxruntime.Release{
			Address:    server.URL + "/onnxruntime.tgz",
			Sum:        sum(archive),
			Library:    "libonnxruntime.so",
			LibrarySum: sum(libraryBytes),
		},
	}
}

func TestProvisionFetchesAndUnpacksThePinnedLibrary(t *testing.T) {
	archive := buildTgz(t, "libonnxruntime.so.1.23.0", libraryBytes)
	server, hits := serve(t, archive)
	cfg := newConfig(server, archive)
	cfg.Dir = t.TempDir()

	library, provider, err := onnxruntime.New(cfg).Provision(t.Context())
	if err != nil {
		t.Fatalf("Provision() = %v", err)
	}
	if provider != "cpu" {
		t.Errorf("provider = %q, want cpu", provider)
	}
	if !strings.HasPrefix(library, cfg.Dir) || filepath.Base(library) != "libonnxruntime.so" {
		t.Errorf("library = %q, want libonnxruntime.so under %s", library, cfg.Dir)
	}
	if got, err := os.ReadFile(library); err != nil || !bytes.Equal(got, libraryBytes) {
		t.Errorf("library holds %q, %v", got, err)
	}
	if hits.Load() != 1 {
		t.Errorf("the server was asked %d times, want 1", hits.Load())
	}
}

func TestProvisionUnpacksAZipArchive(t *testing.T) {
	archive := buildZip(t, "onnxruntime.dll", libraryBytes)
	server, _ := serve(t, archive)
	cfg := newConfig(server, archive)
	cfg.Dir = t.TempDir()
	cfg.Release.Address = server.URL + "/onnxruntime.zip"
	cfg.Release.Library = "onnxruntime.dll"

	library, _, err := onnxruntime.New(cfg).Provision(t.Context())
	if err != nil {
		t.Fatalf("Provision() = %v", err)
	}
	if got, err := os.ReadFile(library); err != nil || !bytes.Equal(got, libraryBytes) {
		t.Errorf("library holds %q, %v", got, err)
	}
}

func TestProvisionRefusesAnArchiveThatIsNotThePinnedOne(t *testing.T) {
	archive := buildTgz(t, "libonnxruntime.so.1.23.0", libraryBytes)
	server, _ := serve(t, archive)
	cfg := newConfig(server, archive)
	cfg.Dir = t.TempDir()
	cfg.Release.Sum = sum([]byte("another archive"))

	_, _, err := onnxruntime.New(cfg).Provision(t.Context())
	if err == nil || !strings.Contains(err.Error(), "is pinned") {
		t.Fatalf("Provision() = %v, want a pin mismatch", err)
	}
	assertNoFile(t, cfg.Dir)
}

func TestProvisionRefusesALibraryThatIsNotThePinnedOne(t *testing.T) {
	archive := buildTgz(t, "libonnxruntime.so.1.23.0", []byte("another library"))
	server, _ := serve(t, archive)
	cfg := newConfig(server, archive)
	cfg.Dir = t.TempDir()

	_, _, err := onnxruntime.New(cfg).Provision(t.Context())
	if err == nil || !strings.Contains(err.Error(), "is pinned") {
		t.Fatalf("Provision() = %v, want a pin mismatch", err)
	}
	if matches, _ := filepath.Glob(filepath.Join(cfg.Dir, "*", "*", "libonnxruntime.so")); len(matches) != 0 {
		t.Errorf("a library that is not pinned was left at %v", matches)
	}
}

func TestProvisionRefusesAnArchiveWithoutTheLibrary(t *testing.T) {
	archive := buildTgz(t, "libother.so", libraryBytes)
	server, _ := serve(t, archive)
	cfg := newConfig(server, archive)
	cfg.Dir = t.TempDir()

	if _, _, err := onnxruntime.New(cfg).Provision(t.Context()); err == nil || !strings.Contains(err.Error(), "holds no libonnxruntime.so") {
		t.Fatalf("Provision() = %v, want an archive without the library", err)
	}
}

func TestProvisionOfflineFetchesNothing(t *testing.T) {
	archive := buildTgz(t, "libonnxruntime.so.1.23.0", libraryBytes)
	server, hits := serve(t, archive)
	cfg := newConfig(server, archive)
	cfg.Dir = t.TempDir()
	cfg.Offline = true

	_, _, err := onnxruntime.New(cfg).Provision(t.Context())
	if err == nil || !strings.Contains(err.Error(), onnxruntime.EnvOffline) {
		t.Fatalf("Provision() = %v, want the offline switch named", err)
	}
	if hits.Load() != 0 {
		t.Errorf("an offline host asked the server %d times", hits.Load())
	}
}

func TestProvisionOfflineUsesTheLibraryAlreadyThere(t *testing.T) {
	archive := buildTgz(t, "libonnxruntime.so.1.23.0", libraryBytes)
	server, hits := serve(t, archive)
	cfg := newConfig(server, archive)
	cfg.Dir = t.TempDir()
	first, _, err := onnxruntime.New(cfg).Provision(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	cfg.Offline = true
	again, _, err := onnxruntime.New(cfg).Provision(t.Context())
	if err != nil || again != first {
		t.Fatalf("Provision() = %q, %v, want %q", again, err, first)
	}
	if hits.Load() != 1 {
		t.Errorf("the server was asked %d times, want 1", hits.Load())
	}
}

func TestProvisionOfflineUnpacksAnArchiveAlreadyThere(t *testing.T) {
	archive := buildTgz(t, "libonnxruntime.so.1.23.0", libraryBytes)
	server, hits := serve(t, archive)
	cfg := newConfig(server, archive)
	cfg.Dir = t.TempDir()
	library, _, err := onnxruntime.New(cfg).Provision(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(library); err != nil {
		t.Fatal(err)
	}

	cfg.Offline = true
	if _, _, err := onnxruntime.New(cfg).Provision(t.Context()); err != nil {
		t.Fatalf("Provision() = %v", err)
	}
	if hits.Load() != 1 {
		t.Errorf("the server was asked %d times, want 1", hits.Load())
	}
}

func TestProvisionOfflineDropsAnArchiveThatIsNotPinned(t *testing.T) {
	archive := buildTgz(t, "libonnxruntime.so.1.23.0", libraryBytes)
	server, _ := serve(t, archive)
	cfg := newConfig(server, archive)
	cfg.Dir = t.TempDir()
	dir := filepath.Join(cfg.Dir, "onnxruntime", cfg.Release.Sum[:12])
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(dir, "onnxruntime.tgz")
	if err := os.WriteFile(stale, []byte("truncated"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg.Offline = true

	if _, _, err := onnxruntime.New(cfg).Provision(t.Context()); err == nil {
		t.Fatal("Provision() accepted an archive that is not the pinned one")
	}
	if _, err := os.Stat(stale); err == nil {
		t.Error("an archive that is not the pinned one was kept")
	}
}

func TestProvisionRunsOnceForConcurrentFirstCalls(t *testing.T) {
	archive := buildTgz(t, "libonnxruntime.so.1.23.0", libraryBytes)
	server, hits := serve(t, archive)
	cfg := newConfig(server, archive)
	cfg.Dir = t.TempDir()
	provisioner := onnxruntime.New(cfg)

	const callers = 16
	libraries := make([]string, callers)
	errs := make([]error, callers)
	var wg sync.WaitGroup
	for i := range callers {
		wg.Go(func() {
			libraries[i], _, errs[i] = provisioner.Provision(t.Context())
		})
	}
	wg.Wait()

	for i := range callers {
		if errs[i] != nil || libraries[i] != libraries[0] {
			t.Fatalf("caller %d got %q, %v; caller 0 got %q", i, libraries[i], errs[i], libraries[0])
		}
	}
	if hits.Load() != 1 {
		t.Errorf("the server was asked %d times, want 1", hits.Load())
	}
}

func TestProvisionAfterAFailureTriesAgain(t *testing.T) {
	archive := buildTgz(t, "libonnxruntime.so.1.23.0", libraryBytes)
	var failing atomic.Bool
	failing.Store(true)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if failing.Load() {
			http.Error(w, "not yet", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write(archive)
	}))
	t.Cleanup(server.Close)
	cfg := newConfig(server, archive)
	cfg.Dir = t.TempDir()
	provisioner := onnxruntime.New(cfg)

	if _, _, err := provisioner.Provision(t.Context()); err == nil {
		t.Fatal("Provision() succeeded against a failing server")
	}
	failing.Store(false)
	if _, _, err := provisioner.Provision(t.Context()); err != nil {
		t.Fatalf("Provision() after the server recovered = %v", err)
	}
}

func TestProvisionWaitsAgainstItsOwnContext(t *testing.T) {
	release := make(chan struct{})
	archive := buildTgz(t, "libonnxruntime.so.1.23.0", libraryBytes)
	entered := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		_, _ = w.Write(archive)
	}))
	t.Cleanup(server.Close)
	cfg := newConfig(server, archive)
	cfg.Dir = t.TempDir()
	provisioner := onnxruntime.New(cfg)

	done := make(chan error, 1)
	go func() {
		_, _, err := provisioner.Provision(context.WithoutCancel(t.Context()))
		done <- err
	}()
	<-entered

	waiter, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := provisioner.Provision(waiter); err == nil {
		t.Error("a caller whose context ended was held to the provisioning of another")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("the first caller = %v", err)
	}
}

func TestProvisionRefusesAnAddressThatIsNotHTTPS(t *testing.T) {
	cfg := onnxruntime.Config{
		Dir: t.TempDir(),
		Release: onnxruntime.Release{
			Address:    "http://example.invalid/onnxruntime.tgz",
			Sum:        sum([]byte("a")),
			Library:    "libonnxruntime.so",
			LibrarySum: sum(libraryBytes),
		},
	}
	if _, _, err := onnxruntime.New(cfg).Provision(t.Context()); err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("Provision() = %v, want an https refusal", err)
	}
}

func TestProvisionRefusesASumThatIsNotASha256(t *testing.T) {
	cfg := onnxruntime.Config{
		Dir: t.TempDir(),
		Release: onnxruntime.Release{
			Address:    "https://example.invalid/onnxruntime.tgz",
			Sum:        "abc",
			Library:    "libonnxruntime.so",
			LibrarySum: sum(libraryBytes),
		},
	}
	if _, _, err := onnxruntime.New(cfg).Provision(t.Context()); err == nil || !strings.Contains(err.Error(), "sha256") {
		t.Fatalf("Provision() = %v, want a sha256 refusal", err)
	}
}

func TestProvisionRefusesALibraryNameThatIsAPath(t *testing.T) {
	cfg := onnxruntime.Config{
		Dir: t.TempDir(),
		Release: onnxruntime.Release{
			Address:    "https://example.invalid/onnxruntime.tgz",
			Sum:        sum([]byte("a")),
			Library:    "../libonnxruntime.so",
			LibrarySum: sum(libraryBytes),
		},
	}
	if _, _, err := onnxruntime.New(cfg).Provision(t.Context()); err == nil {
		t.Fatal("Provision() accepted a library name that leaves its folder")
	}
}

func TestProvisionRefusesAnUnknownProvider(t *testing.T) {
	cfg := onnxruntime.Config{Dir: t.TempDir(), Provider: "tensorrt"}
	if _, _, err := onnxruntime.New(cfg).Provision(t.Context()); err == nil || !strings.Contains(err.Error(), "tensorrt") {
		t.Fatalf("Provision() = %v, want the provider named", err)
	}
}

func TestProvisionRefusesCoreMLWhereThereIsNone(t *testing.T) {
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		t.Skip("this platform has coreml")
	}
	cfg := onnxruntime.Config{Dir: t.TempDir(), Provider: "coreml"}
	if _, _, err := onnxruntime.New(cfg).Provision(t.Context()); err == nil || !strings.Contains(err.Error(), "coreml") {
		t.Fatalf("Provision() = %v, want the provider refused", err)
	}
}

func TestProvisionRefusesCUDAWhileOnlyTheCPUBuildIsPinned(t *testing.T) {
	cfg := onnxruntime.Config{Dir: t.TempDir(), Provider: "cuda"}
	if _, _, err := onnxruntime.New(cfg).Provision(t.Context()); err == nil || !strings.Contains(err.Error(), "GPU build") {
		t.Fatalf("Provision() = %v, want the GPU build named", err)
	}
}

func TestProvisionAnswersTheProviderTheConfigurationNames(t *testing.T) {
	archive := buildTgz(t, "libonnxruntime.so.1.23.0", libraryBytes)
	server, _ := serve(t, archive)
	cfg := newConfig(server, archive)
	cfg.Dir = t.TempDir()
	cfg.Provider = " CPU "

	_, provider, err := onnxruntime.New(cfg).Provision(t.Context())
	if err != nil || provider != "cpu" {
		t.Fatalf("Provision() provider = %q, %v", provider, err)
	}
}

func TestProvisionKeepsTheLibraryInTheModelsHome(t *testing.T) {
	archive := buildTgz(t, "libonnxruntime.so.1.23.0", libraryBytes)
	server, _ := serve(t, archive)
	cfg := newConfig(server, archive)
	home := t.TempDir()
	t.Setenv(onnxruntime.EnvDir, home)

	library, _, err := onnxruntime.New(cfg).Provision(t.Context())
	if err != nil || !strings.HasPrefix(library, home) {
		t.Fatalf("Provision() = %q, %v, want a library under %s", library, err, home)
	}
}

func TestFromEnvironmentReadsTheProviderAndTheOfflineSwitch(t *testing.T) {
	t.Setenv(onnxruntime.EnvProvider, "coreml")
	t.Setenv(onnxruntime.EnvOffline, "1")
	cfg := onnxruntime.FromEnvironment()
	if cfg.Provider != "coreml" || !cfg.Offline {
		t.Errorf("FromEnvironment() = %+v", cfg)
	}

	t.Setenv(onnxruntime.EnvProvider, "")
	t.Setenv(onnxruntime.EnvOffline, "no")
	cfg = onnxruntime.FromEnvironment()
	if cfg.Provider != "" || cfg.Offline {
		t.Errorf("FromEnvironment() = %+v", cfg)
	}
}

func TestGetHomeIsTheSharedModelsFolder(t *testing.T) {
	t.Setenv(onnxruntime.EnvDir, "/srv/models")
	if got, err := onnxruntime.GetHome(); err != nil || got != "/srv/models" {
		t.Fatalf("GetHome() = %q, %v", got, err)
	}
	t.Setenv(onnxruntime.EnvDir, "")
	t.Setenv("XDG_CONFIG_HOME", "/srv/config")
	t.Setenv("HOME", "/srv/home")
	t.Setenv("AppData", "/srv/config")
	got, err := onnxruntime.GetHome()
	if err != nil || !strings.HasSuffix(filepath.ToSlash(got), "numen/models") {
		t.Fatalf("GetHome() = %q, %v", got, err)
	}
}

// assertNoFile fails when a file was left under dir.
func assertNoFile(t *testing.T, dir string) {
	t.Helper()
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			t.Errorf("%s was left behind", path)
		}
		return nil
	})
}
