package provision

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func buildTgz(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	zipped := gzip.NewWriter(&buffer)
	archive := tar.NewWriter(zipped)
	for name, data := range files {
		if err := archive.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := archive.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zipped.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func buildZip(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	file, err := archive.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func serveFiles(t *testing.T, files map[string][]byte) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		data, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if _, err := w.Write(data); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

func getPlatform() string { return runtime.GOOS + "/" + runtime.GOARCH }

func newRuntimeRequirement(sources []manifest.DependencySource) manifest.DependencyRequirement {
	return manifest.DependencyRequirement{
		ID: "sample-runtime", Kind: manifest.DependencyKindRuntime, Version: "1.0.0", Sources: sources,
		Metadata: map[string]string{"envVar": "SAMPLE_RUNTIME_LIB"},
	}
}

func TestTheSourceOfThisPlatformIsFetched(t *testing.T) {
	here, there := []byte("library of this platform"), []byte("library of another platform")
	server, _ := serveFiles(t, map[string][]byte{"/here.so": here, "/there.so": there})
	req := newRuntimeRequirement([]manifest.DependencySource{
		{Platform: getPlatform(), SourceURL: server.URL + "/here.so", ChecksumSHA256: sumOf(here)},
		{Platform: "plan9/mips64", SourceURL: server.URL + "/there.so", ChecksumSHA256: sumOf(there)},
	})
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = server.Client()

	resolved, err := provisioner.Provision(t.Context(), req, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(resolved.PrimaryPath)
	if err != nil || !bytes.Equal(got, here) {
		t.Errorf("the library holds %q, %v; want the one of this platform", got, err)
	}
	if resolved.EnvVars["SAMPLE_RUNTIME_LIB"] != resolved.PrimaryPath {
		t.Errorf("variables = %v", resolved.EnvVars)
	}
}

func TestADependencyWithNoSourceForThisPlatformIsRefused(t *testing.T) {
	server, requests := serveFiles(t, map[string][]byte{"/there.so": []byte("x")})
	req := newRuntimeRequirement([]manifest.DependencySource{
		{Platform: "plan9/mips64", SourceURL: server.URL + "/there.so", ChecksumSHA256: sumOf([]byte("x"))},
	})
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = server.Client()

	_, err := provisioner.Provision(t.Context(), req, nil)
	if err == nil || !strings.Contains(err.Error(), getPlatform()) {
		t.Errorf("error = %v, want one that names %s", err, getPlatform())
	}
	if requests.Load() != 0 {
		t.Errorf("%d requests for a platform the sources do not name", requests.Load())
	}
}

func TestTheArchiveMemberIsUnpackedAndTheArchiveRemoved(t *testing.T) {
	library := []byte("the library")
	archive := buildTgz(t, map[string][]byte{"runtime-1.0.0/README": []byte("read me"), "runtime-1.0.0/lib/libsample.so.1.0.0": library})
	server, _ := serveFiles(t, map[string][]byte{"/runtime.tgz": archive})
	req := newRuntimeRequirement([]manifest.DependencySource{{Platform: getPlatform(),
		SourceURL: server.URL + "/runtime.tgz", ChecksumSHA256: sumOf(archive),
		ArchiveMember: "libsample.so.*", MemberSHA256: sumOf(library),
	}})
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = server.Client()

	resolved, err := provisioner.Provision(t.Context(), req, nil)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(resolved.PrimaryPath) != "libsample.so.1.0.0" {
		t.Errorf("the dependency is %s, want the member", resolved.PrimaryPath)
	}
	if got, err := os.ReadFile(resolved.PrimaryPath); err != nil || !bytes.Equal(got, library) {
		t.Errorf("the member holds %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(resolved.PrimaryPath), "runtime.tgz")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the archive is still there: %v", err)
	}
	again, err := provisioner.Provision(t.Context(), req, nil)
	if err != nil || again.PrimaryPath != resolved.PrimaryPath {
		t.Errorf("the second provisioning gave %s, %v; want the unpacked member", again.PrimaryPath, err)
	}
}

func TestAZipMemberIsMatchedByItsPath(t *testing.T) {
	library := []byte("the windows library")
	archive := buildZip(t, "runtime-1.0.0/lib/sample.dll", library)
	server, _ := serveFiles(t, map[string][]byte{"/runtime.zip": archive})
	req := newRuntimeRequirement([]manifest.DependencySource{{Platform: getPlatform(),
		SourceURL: server.URL + "/runtime.zip", ChecksumSHA256: sumOf(archive),
		ArchiveMember: "*/lib/sample.dll", MemberSHA256: sumOf(library),
	}})
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = server.Client()

	resolved, err := provisioner.Provision(t.Context(), req, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(resolved.PrimaryPath); err != nil || !bytes.Equal(got, library) {
		t.Errorf("the member holds %q, %v", got, err)
	}
}

func TestAnArchiveMemberWithAnotherChecksumIsRefused(t *testing.T) {
	library := []byte("a library that is not the pinned one")
	archive := buildTgz(t, map[string][]byte{"runtime-1.0.0/lib/libsample.so.1.0.0": library})
	server, _ := serveFiles(t, map[string][]byte{"/runtime.tgz": archive})
	req := newRuntimeRequirement([]manifest.DependencySource{{Platform: getPlatform(),
		SourceURL: server.URL + "/runtime.tgz", ChecksumSHA256: sumOf(archive),
		ArchiveMember: "libsample.so.*", MemberSHA256: sumOf([]byte("the pinned library")),
	}})
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = server.Client()

	if _, err := provisioner.Provision(t.Context(), req, nil); !errors.Is(err, ErrChecksum) {
		t.Fatalf("error = %v, want ErrChecksum", err)
	}
	dir := getDependencyDir(provisioner.cfg.Dir, string(req.Kind), req.ID, req.Version)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), tempPrefix) {
			t.Errorf("%s is left in the folder of a refused member", entry.Name())
		}
	}
}

func TestASearchPatternFindsTheDependencyOnThisMachine(t *testing.T) {
	root := t.TempDir()
	library := filepath.Join(root, "lib", "libsample.so.1.2.0")
	if err := os.MkdirAll(filepath.Dir(library), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(library, []byte("installed"), 0o644); err != nil {
		t.Fatal(err)
	}
	server, requests := serveFiles(t, nil)
	req := newRuntimeRequirement([]manifest.DependencySource{
		{Platform: getPlatform(), SourceURL: server.URL + "/runtime.so", ChecksumSHA256: sumOf([]byte("downloaded"))},
	})
	req.Search = []string{"relative/lib/libsample.so*", "$SAMPLE_UNSET/lib/libsample.so*", "$SAMPLE_ROOT/lib/libsample.so*"}
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = server.Client()
	provisioner.cfg.GetEnv = func(name string) string {
		if name == "SAMPLE_ROOT" {
			return root
		}
		return ""
	}

	resolved, err := provisioner.Provision(t.Context(), req, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.PrimaryPath != library || resolved.EnvVars["SAMPLE_RUNTIME_LIB"] != library {
		t.Errorf("resolved %+v, want %s", resolved, library)
	}
	if requests.Load() != 0 {
		t.Errorf("%d requests for a dependency the machine holds", requests.Load())
	}
}

func TestAFallbackSourceURLIsTriedWhenAnEarlierSourceFails(t *testing.T) {
	here := []byte("library from mirror")
	server, _ := serveFiles(t, map[string][]byte{
		"/tampered.so": []byte("wrong bytes"),
		"/mirror.so":   here,
	})
	req := newRuntimeRequirement([]manifest.DependencySource{{
		Platform: getPlatform(),
		SourceURLs: []string{
			server.URL + "/missing.so",
			server.URL + "/tampered.so",
			server.URL + "/mirror.so",
		},
		ChecksumSHA256: sumOf(here),
	}})
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = server.Client()

	resolved, err := provisioner.Provision(t.Context(), req, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(resolved.PrimaryPath)
	if err != nil || !bytes.Equal(got, here) {
		t.Errorf("the library holds %q, %v; want %q", got, err, here)
	}
}

func TestEverySourceURLFailingReportsEveryError(t *testing.T) {
	server, _ := serveFiles(t, nil)
	req := manifest.DependencyRequirement{
		ID:             "whisper",
		Kind:           manifest.DependencyKindModel,
		Version:        "1",
		SourceURLs:     []string{server.URL + "/primary.bin", server.URL + "/mirror.bin"},
		ChecksumSHA256: sumOf([]byte("model")),
	}
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = server.Client()

	_, err := provisioner.Provision(t.Context(), req, nil)
	if err == nil || !strings.Contains(err.Error(), "/primary.bin") || !strings.Contains(err.Error(), "/mirror.bin") {
		t.Errorf("error = %v, want both failed URLs", err)
	}
}

func TestAFetchedBinaryIsAProgramThatCanRun(t *testing.T) {
	program := []byte("program of this platform")
	server, _ := serveFiles(t, map[string][]byte{"/tool-platform": program})
	req := manifest.DependencyRequirement{
		ID: "tool", Kind: manifest.DependencyKindBinary, Version: "1.0.0",
		Metadata: map[string]string{"fileName": "tool"},
		Sources:  []manifest.DependencySource{{Platform: getPlatform(), SourceURL: server.URL + "/tool-platform", ChecksumSHA256: sumOf(program)}},
	}
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = server.Client()

	resolved, err := provisioner.Provision(t.Context(), req, nil)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(resolved.PrimaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != windowsOS && info.Mode().Perm()&0o100 == 0 {
		t.Errorf("mode = %v, want an executable file", info.Mode())
	}
	want := "tool"
	if runtime.GOOS == windowsOS {
		want = "tool.exe"
	}
	if got := filepath.Base(resolved.PrimaryPath); got != want {
		t.Errorf("file = %s, want %s", got, want)
	}
}
