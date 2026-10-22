package provision

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
)

func sumOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func newTestProvisioner(t *testing.T, offline bool) *FileProvisioner {
	t.Helper()
	return NewFileProvisioner(Config{
		Dir: t.TempDir(), Offline: offline, Discovery: discovery{dirs: []string{}},
		Backoff: backoff{attempts: 3, first: time.Millisecond, sleep: func(context.Context, time.Duration) error { return nil }},
	})
}

func newModelRequirement(address string, data []byte) pluginport.DependencyRequirement {
	return pluginport.DependencyRequirement{ID: "whisper", Kind: pluginport.DependencyKindModel, Version: "1", SourceURL: address, ChecksumSHA256: sumOf(data)}
}

type fakeProvisioner struct {
	id    string
	kind  pluginport.DependencyKind
	calls atomic.Int32
	gate  chan struct{}
}

func (f *fakeProvisioner) CanProvision(req pluginport.DependencyRequirement) bool {
	return req.Kind == f.kind && (f.id == "" || req.ID == f.id)
}

func (f *fakeProvisioner) Provision(ctx context.Context, req pluginport.DependencyRequirement, onProgress func(pluginport.DependencyStatus)) (pluginport.ResolvedDependency, error) {
	f.calls.Add(1)
	onProgress(pluginport.DependencyStatus{BytesDownloaded: 1, BytesTotal: 2})
	if f.gate != nil {
		select {
		case <-f.gate:
		case <-ctx.Done():
			return pluginport.ResolvedDependency{}, ctx.Err()
		}
	}
	onProgress(pluginport.DependencyStatus{BytesDownloaded: 2, BytesTotal: 2})
	return pluginport.ResolvedDependency{ID: req.ID, PrimaryPath: "/p/" + req.ID}, nil
}

func (f *fakeProvisioner) Status(context.Context, pluginport.DependencyRequirement) (pluginport.DependencyStatus, error) {
	return pluginport.DependencyStatus{State: pluginport.DependencyStateMissing}, nil
}

func TestTheRegistryRoutesARequirementToTheProvisionerThatHandlesIt(t *testing.T) {
	onnx := &fakeProvisioner{id: "onnxruntime", kind: pluginport.DependencyKindRuntime}
	models := &fakeProvisioner{kind: pluginport.DependencyKindModel}
	registry := NewRegistry(onnx, models)

	if _, err := registry.Provision(t.Context(), pluginport.DependencyRequirement{ID: "onnxruntime", Kind: pluginport.DependencyKindRuntime}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Provision(t.Context(), pluginport.DependencyRequirement{ID: "m", Kind: pluginport.DependencyKindModel}, nil); err != nil {
		t.Fatal(err)
	}
	if onnx.calls.Load() != 1 || models.calls.Load() != 1 {
		t.Errorf("onnx ran %d times and models %d, want one each", onnx.calls.Load(), models.calls.Load())
	}
	_, err := registry.Provision(t.Context(), pluginport.DependencyRequirement{ID: "d", Kind: pluginport.DependencyKindDataset}, nil)
	if !errors.Is(err, pluginport.ErrNoProvisioner) {
		t.Errorf("err = %v, want ErrNoProvisioner", err)
	}
	if registry.CanProvision(pluginport.DependencyRequirement{ID: "d", Kind: pluginport.DependencyKindDataset}) {
		t.Error("the registry claims to provision a dataset nothing handles")
	}
}

func TestConcurrentIdenticalRequestsShareOneRunAndAllHearItsProgress(t *testing.T) {
	provisioner := &fakeProvisioner{kind: pluginport.DependencyKindModel, gate: make(chan struct{})}
	registry := NewRegistry(provisioner)
	req := pluginport.DependencyRequirement{ID: "m", Kind: pluginport.DependencyKindModel, Version: "1"}

	const callers = 5
	var heard [callers]atomic.Int32
	var started, finished sync.WaitGroup
	started.Add(callers)
	finished.Add(callers)
	for i := range callers {
		go func() {
			defer finished.Done()
			started.Done()
			_, err := registry.Provision(t.Context(), req, func(pluginport.DependencyStatus) { heard[i].Add(1) })
			if err != nil {
				t.Error(err)
			}
		}()
	}
	started.Wait()
	for !registry.flights.isRunning(getFlightKey(req)) {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	close(provisioner.gate)
	finished.Wait()

	if got := provisioner.calls.Load(); got != 1 {
		t.Errorf("the provisioner ran %d times, want 1", got)
	}
	for i := range heard {
		if heard[i].Load() == 0 {
			t.Errorf("caller %d heard no progress", i)
		}
	}
}

func TestARunContinuesWhileACallerRemainsAndStopsWhenAllLeave(t *testing.T) {
	provisioner := &fakeProvisioner{kind: pluginport.DependencyKindModel, gate: make(chan struct{})}
	registry := NewRegistry(provisioner)
	req := pluginport.DependencyRequirement{ID: "m", Kind: pluginport.DependencyKindModel}
	ctx, cancel := context.WithCancel(t.Context())
	errs := make(chan error, 1)
	go func() {
		_, err := registry.Provision(ctx, req, nil)
		errs <- err
	}()
	for !registry.flights.isRunning(getFlightKey(req)) {
		time.Sleep(time.Millisecond)
	}
	cancel()
	if err := <-errs; !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want canceled", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for registry.flights.isRunning(getFlightKey(req)) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if registry.flights.isRunning(getFlightKey(req)) {
		t.Error("the run kept going after every caller left")
	}
}

func TestADownloadIsVerifiedAndPlacedUnderTheCacheLayout(t *testing.T) {
	data := bytes.Repeat([]byte("model-bytes"), 5000)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "w.bin", time.Time{}, bytes.NewReader(data))
	}))
	defer server.Close()
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = server.Client()
	req := newModelRequirement(server.URL+"/w.bin", data)

	var progressed atomic.Int32
	resolved, err := provisioner.Provision(t.Context(), req, func(pluginport.DependencyStatus) { progressed.Add(1) })
	if err != nil {
		t.Fatal(err)
	}
	wantDir := filepath.Join(provisioner.cfg.Dir, "model", "whisper")
	if !strings.HasPrefix(resolved.RootPath, wantDir) || filepath.Base(resolved.PrimaryPath) != "w.bin" {
		t.Errorf("resolved %+v, want under %s", resolved, wantDir)
	}
	got, err := os.ReadFile(resolved.PrimaryPath)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("the file holds the wrong bytes: %v", err)
	}
	if !isReady(resolved.RootPath) {
		t.Error("the .ready sentinel is missing")
	}
	if progressed.Load() == 0 {
		t.Error("no progress was reported")
	}
	if resolved.EnvVars["NUMEN_DEPENDENCY_WHISPER"] != resolved.PrimaryPath {
		t.Errorf("env = %v", resolved.EnvVars)
	}
	left, _ := filepath.Glob(filepath.Join(resolved.RootPath, tempPrefix+"*"))
	if len(left) != 0 {
		t.Errorf("temporary files remain: %v", left)
	}

	// A second provisioning is served from the folder with the server gone.
	server.Close()
	again, err := provisioner.Provision(t.Context(), req, nil)
	if err != nil || again.PrimaryPath != resolved.PrimaryPath {
		t.Errorf("second provision = %+v, %v", again, err)
	}
	status, err := provisioner.Status(t.Context(), req)
	if err != nil || status.State != pluginport.DependencyStateReady {
		t.Errorf("status = %+v, %v", status, err)
	}
}

func TestADownloadResumesFromAPartialFile(t *testing.T) {
	data := bytes.Repeat([]byte("0123456789"), 4000)
	var ranges []string
	var mu sync.Mutex
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		ranges = append(ranges, r.Header.Get("Range"))
		mu.Unlock()
		http.ServeContent(w, r, "w.bin", time.Time{}, bytes.NewReader(data))
	}))
	defer server.Close()
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = server.Client()
	req := newModelRequirement(server.URL+"/w.bin", data)
	dir, _ := provisioner.getDir(req)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, tempPrefix+"left-over"), data[:15000], 0o644); err != nil {
		t.Fatal(err)
	}

	resolved, err := provisioner.Provision(t.Context(), req, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(resolved.PrimaryPath)
	if !bytes.Equal(got, data) {
		t.Error("the resumed file holds the wrong bytes")
	}
	if len(ranges) != 1 || ranges[0] != "bytes=15000-" {
		t.Errorf("Range headers = %q, want bytes=15000-", ranges)
	}
}

func TestADownloadThatFailsTheChecksumLeavesNothingInPlace(t *testing.T) {
	data := []byte("the real bytes")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("tampered bytes"))
	}))
	defer server.Close()
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = server.Client()
	req := newModelRequirement(server.URL+"/w.bin", data)

	_, err := provisioner.Provision(t.Context(), req, nil)
	if !errors.Is(err, ErrChecksum) {
		t.Fatalf("err = %v, want ErrChecksum", err)
	}
	dir, _ := provisioner.getDir(req)
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		t.Errorf("%s was left in the folder", entry.Name())
	}
	if status, _ := provisioner.Status(t.Context(), req); status.State != pluginport.DependencyStateMissing {
		t.Errorf("state = %s, want missing", status.State)
	}
}

func TestAnOfflineMachineFetchesNothing(t *testing.T) {
	provisioner := newTestProvisioner(t, true)
	_, err := provisioner.Provision(t.Context(), newModelRequirement("https://example.invalid/w.bin", []byte("x")), nil)
	if !errors.Is(err, ErrOffline) {
		t.Errorf("err = %v, want ErrOffline", err)
	}
}

func TestTheMachineIsSearchedBeforeTheNetwork(t *testing.T) {
	bin := t.TempDir()
	program := filepath.Join(bin, "ffmpeg")
	if err := os.WriteFile(program, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	provisioner := newTestProvisioner(t, true)
	provisioner.cfg.Discovery = discovery{dirs: []string{bin}}

	resolved, err := provisioner.Provision(t.Context(), pluginport.DependencyRequirement{ID: "ffmpeg", Kind: pluginport.DependencyKindBinary}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.PrimaryPath != program {
		t.Errorf("PrimaryPath = %s, want %s", resolved.PrimaryPath, program)
	}
}

func TestTheLoaderCacheNamesALibrary(t *testing.T) {
	cache := []byte("2 libs found in cache\n\tlibfoo.so.1 (libc6,x86-64) => /usr/lib/libfoo.so.1\n\tlibbar.so (libc6) => /usr/lib/libbar.so\n")
	if got := findInLoaderCache(cache, "foo"); got != "/usr/lib/libfoo.so.1" {
		t.Errorf("found %q", got)
	}
	if got := findInLoaderCache(cache, "baz"); got != "" {
		t.Errorf("found %q for a library the cache lacks", got)
	}
}

func TestTheCacheLockExcludesAnotherHolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.lock")
	release, err := lockFile(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 250*time.Millisecond)
	defer cancel()
	if _, err := lockFile(ctx, path); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second lock err = %v, want a wait that ended", err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	again, err := lockFile(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	_ = again()
}

func TestARenameIsRetriedWhileAccessIsDenied(t *testing.T) {
	var attempts int
	policy := backoff{attempts: 5, first: time.Millisecond, sleep: func(context.Context, time.Duration) error { return nil }}
	err := retry(t.Context(), policy, func(error) bool { return true }, func() error {
		attempts++
		if attempts < 3 {
			return os.ErrPermission
		}
		return nil
	})
	if err != nil || attempts != 3 {
		t.Errorf("err = %v after %d attempts, want success on the third", err, attempts)
	}
	attempts = 0
	err = retry(t.Context(), policy, func(error) bool { return false }, func() error { attempts++; return os.ErrPermission })
	if err == nil || attempts != 1 {
		t.Errorf("an error that is not retried ran %d times", attempts)
	}
}

func TestTheCacheFolderNamesKindIdVersionAndPlatform(t *testing.T) {
	dir := getDependencyDir("/root", "model", "../x/y", "1.0")
	if !strings.HasPrefix(dir, filepath.Join("/root", "model", ".._x_y", "1.0-")) {
		t.Errorf("dir = %s", dir)
	}
	if getDependencyDir("/r", "model", "..", "") == filepath.Join("/r", "model", "..") {
		t.Error("a parent folder id escaped the root")
	}
}
