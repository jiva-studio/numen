package provision

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

func TestACallerJoiningARunEveryCallerLeftStartsAFreshRun(t *testing.T) {
	var group flightGroup
	release := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	started := make(chan struct{})
	var runs atomic.Int32
	run := func(ctx context.Context, _ func(pluginport.DependencyStatus)) (pluginport.ResolvedDependency, error) {
		if runs.Add(1) == 1 {
			close(started)
			<-ctx.Done()
			<-release
			return pluginport.ResolvedDependency{}, ctx.Err()
		}
		return pluginport.ResolvedDependency{ID: "fresh"}, nil
	}

	first, leave := context.WithCancel(t.Context())
	exit := make(chan struct{})
	go func() {
		if _, err := group.do(first, "k", nil, run); !errors.Is(err, context.Canceled) {
			t.Errorf("a run whose context was cancelled gave %v, want context.Canceled", err)
		}
		close(exit)
	}()
	<-started
	leave()
	<-exit

	type outcome struct {
		resolved pluginport.ResolvedDependency
		err      error
	}
	got := make(chan outcome, 1)
	go func() {
		resolved, err := group.do(t.Context(), "k", nil, run)
		got <- outcome{resolved, err}
	}()
	var result outcome
	select {
	case result = <-got:
	case <-time.After(300 * time.Millisecond):
		releaseOnce.Do(func() { close(release) })
		result = <-got
	}
	if result.err != nil || result.resolved.ID != "fresh" {
		t.Errorf("the caller joined the abandoned run: %+v, %v", result.resolved, result.err)
	}
}

func TestAFileNameCannotCollideWithTheFilesTheFolderKeepsForItself(t *testing.T) {
	data := []byte("model-bytes")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(data); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	for _, name := range []string{".ready", ".ready.tmp", ".tmp.abc"} {
		t.Run(name, func(t *testing.T) {
			provisioner := newTestProvisioner(t, false)
			provisioner.cfg.Client = server.Client()
			req := newModelRequirement(server.URL+"/w.bin", data)
			req.Metadata = map[string]string{"fileName": name}

			resolved, err := provisioner.Provision(t.Context(), req, nil)

			if err == nil {
				t.Errorf("the file name %q was accepted: %+v", name, resolved)
			}
		})
	}
}

func TestADiscoveredNameCannotLeaveTheSearchedFolders(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "secret"), []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"../secret", "se*", "/etc/passwd"} {
		t.Run(name, func(t *testing.T) {
			provisioner := newTestProvisioner(t, true)
			provisioner.cfg.Discovery = discovery{dirs: []string{bin}}
			req := pluginport.DependencyRequirement{ID: "tool", Kind: pluginport.DependencyKindBinary, Metadata: map[string]string{"binary": name}}

			resolved, err := provisioner.Provision(t.Context(), req, nil)

			if err == nil {
				t.Errorf("the name %q found %s", name, resolved.PrimaryPath)
			}
		})
	}
}

func TestACachedDependencyIsServedOnlyForTheChecksumItWasFetchedWith(t *testing.T) {
	data := []byte("model-bytes")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(data); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = server.Client()
	if _, err := provisioner.Provision(t.Context(), newModelRequirement(server.URL+"/w.bin", data), nil); err != nil {
		t.Fatal(err)
	}

	pinned := newModelRequirement(server.URL+"/w.bin", bytes.Repeat([]byte("other"), 3))
	_, err := provisioner.Provision(t.Context(), pinned, nil)

	if !errors.Is(err, ErrChecksum) {
		t.Errorf("a file fetched with another checksum was served: %v", err)
	}
}

func TestAMachineFileThatIsNotThePinnedBytesIsNotTheDependency(t *testing.T) {
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "ffmpeg"), []byte("not the pinned bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	provisioner := newTestProvisioner(t, true)
	provisioner.cfg.Discovery = discovery{dirs: []string{bin}}
	req := pluginport.DependencyRequirement{
		ID: "ffmpeg", Kind: pluginport.DependencyKindBinary, SourceURL: "https://example.invalid/ffmpeg", ChecksumSHA256: sumOf([]byte("pinned")),
	}

	resolved, err := provisioner.Provision(t.Context(), req, nil)

	if !errors.Is(err, ErrOffline) {
		t.Errorf("the machine's file was taken for the pinned one: %+v, %v", resolved, err)
	}
}
