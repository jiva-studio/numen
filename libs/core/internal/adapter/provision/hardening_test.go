package provision

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
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

func TestProvision_RejectsReservedFileNames(t *testing.T) {
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

func TestProvision_CachedDependencyRequiresMatchingChecksum(t *testing.T) {
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

func TestAFailedDownloadHidesTheCredentialsOfItsAddress(t *testing.T) {
	provisioner := newTestProvisioner(t, true)
	address := "https://reader:token-1@models.example.invalid/weights.onnx?signature=secret-2#part"

	_, err := provisioner.Provision(t.Context(), newModelRequirement(address, []byte("weights")), nil)
	if err == nil {
		t.Fatal("the download succeeded offline")
	}
	text := err.Error()
	for _, secret := range []string{"token-1", "secret-2", "reader"} {
		if strings.Contains(text, secret) {
			t.Errorf("the failure reads %q, which holds %q", text, secret)
		}
	}
	if want := "https://models.example.invalid/weights.onnx"; !strings.Contains(text, want) {
		t.Errorf("the failure reads %q, want it to name %q", text, want)
	}
}

func TestAFailedDownloadEscapesControlCharactersInItsAddress(t *testing.T) {
	provisioner := newTestProvisioner(t, true)
	address := "https://models.example.invalid/weights\n.onnx"

	_, err := provisioner.Provision(t.Context(), newModelRequirement(address, []byte("weights")), nil)
	if err == nil {
		t.Fatal("the download succeeded offline")
	}
	if text := err.Error(); strings.ContainsAny(text, "\n\r") {
		t.Errorf("the failure reads %q, want one line", text)
	}
}

func TestAMalformedAddressIsNamedOnceInTheFailure(t *testing.T) {
	provisioner := newTestProvisioner(t, false)
	req := newModelRequirement("https://models.example.invalid/%zz", []byte("weights"))
	req.Metadata = map[string]string{"fileName": "weights.onnx"}

	_, err := provisioner.Provision(t.Context(), req, nil)
	if err == nil {
		t.Fatal("the download of a malformed address succeeded")
	}
	if text := err.Error(); strings.Count(text, "models.example.invalid") != 1 {
		t.Errorf("the failure reads %q, want the address named once", text)
	}
}

func TestAFailedDownloadNamesTheHostItWasRedirectedTo(t *testing.T) {
	dropping := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Error("the response writer does not support hijacking")
			return
		}
		conn, _, err := hijacker.Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}))
	defer dropping.Close()
	redirecting := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, dropping.URL+"/weights.onnx?signature=secret-3", http.StatusFound)
	}))
	defer redirecting.Close()
	pool := x509.NewCertPool()
	pool.AddCert(redirecting.Certificate())
	pool.AddCert(dropping.Certificate())
	provisioner := newTestProvisioner(t, false)
	provisioner.cfg.Client = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}}}

	_, err := provisioner.Provision(t.Context(), newModelRequirement(redirecting.URL+"/weights.onnx", []byte("weights")), nil)
	if err == nil {
		t.Fatal("the download over a dropped connection succeeded")
	}
	text := err.Error()
	if host := strings.TrimPrefix(dropping.URL, "https://"); !strings.Contains(text, host) {
		t.Errorf("the failure reads %q, want it to name the host %s it was redirected to", text, host)
	}
	if strings.Contains(text, "secret-3") {
		t.Errorf("the failure reads %q, which holds the query of the redirect", text)
	}
}

func TestAMalformedAddressIsRedactedInTheFailure(t *testing.T) {
	provisioner := newTestProvisioner(t, true)
	address := "https://reader:token-6@models.example.invalid/%zz?signature=secret-7#part"

	_, err := provisioner.Provision(t.Context(), newModelRequirement(address, []byte("weights")), nil)
	if err == nil {
		t.Fatal("the download succeeded offline")
	}
	text := err.Error()
	for _, secret := range []string{"token-6", "secret-7", "reader", "part"} {
		if strings.Contains(text, secret) {
			t.Errorf("the failure reads %q, which holds %q", text, secret)
		}
	}
	if want := "models.example.invalid/%zz"; !strings.Contains(text, want) {
		t.Errorf("the failure reads %q, want it to name %q", text, want)
	}
}

func TestARefusedRedirectIsRedactedInTheFailure(t *testing.T) {
	redirecting := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://reader:token-8@models.example.invalid/weights.onnx?signature=secret-9", http.StatusFound)
	}))
	defer redirecting.Close()
	provisioner := newTestProvisioner(t, false)
	client := newClient()
	client.Transport = redirecting.Client().Transport
	provisioner.cfg.Client = client

	_, err := provisioner.Provision(t.Context(), newModelRequirement(redirecting.URL+"/weights.onnx?token=secret-10", []byte("weights")), nil)
	if err == nil {
		t.Fatal("the download over a redirect to http succeeded")
	}
	text := err.Error()
	for _, secret := range []string{"token-8", "secret-9", "secret-10", "reader"} {
		if strings.Contains(text, secret) {
			t.Errorf("the failure reads %q, which holds %q", text, secret)
		}
	}
	if want := "http://models.example.invalid/weights.onnx"; !strings.Contains(text, want) {
		t.Errorf("the failure reads %q, want it to name the refused address %q", text, want)
	}
}
