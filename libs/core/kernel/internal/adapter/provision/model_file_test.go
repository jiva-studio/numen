package provision

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugindependency"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestModelFileRequestsShareVerifiedContentAcrossAddresses(t *testing.T) {
	const body = "shared model file"
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		if _, err := w.Write([]byte(body)); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	files := newTestProvisioner(t, false)
	files.cfg.Client = server.Client()
	var downloaded atomic.Int64
	registry := NewObservedRegistry(func(_ manifest.DependencyRequirement, status plugin.DependencyStatus) {
		downloaded.Add(status.BytesDownloaded)
	}, files)
	fetch := plugindependency.NewFetch(func() pluginport.DependencyRegistry { return registry })
	digest := newModelRequirement(server.URL, []byte(body)).ChecksumSHA256
	saved, err := fetch.Execute(t.Context(), server.URL+"/first.onnx", digest)
	if err != nil {
		t.Fatal(err)
	}
	other, err := fetch.Execute(t.Context(), server.URL+"/alias.onnx", digest)
	if err != nil || saved != other || calls.Load() != 1 || downloaded.Load() == 0 {
		t.Fatalf("second fetch = %q, %v, requests %d, reported bytes %d", other, err, calls.Load(), downloaded.Load())
	}
	got, err := os.ReadFile(saved)
	if err != nil || string(got) != body {
		t.Fatalf("file = %q, %v", got, err)
	}

	wrong := strings.Repeat("0", 64)
	if _, err := fetch.Execute(t.Context(), server.URL+"/bad", wrong); !errors.Is(err, ErrChecksum) {
		t.Fatalf("wrong digest = %v", err)
	}
	if err := filepath.WalkDir(files.cfg.Dir, func(path string, entry os.DirEntry, err error) error {
		if err == nil && strings.HasPrefix(entry.Name(), tempPrefix) {
			t.Errorf("failed download left %s", path)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

func TestProfileWithoutPinsCachesTheObservedBytesByDigest(t *testing.T) {
	const body = "unversioned model"
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		if _, err := w.Write([]byte(body)); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	files := newTestProvisioner(t, false)
	files.cfg.Client = server.Client()
	registry := NewRegistry(files)
	fetch := plugindependency.NewFetch(func() pluginport.DependencyRegistry { return registry })
	saved, err := fetch.Execute(t.Context(), server.URL+"/plain", "")
	if err != nil {
		t.Fatal(err)
	}
	alias, err := fetch.Execute(t.Context(), server.URL+"/alias", "")
	if err != nil || saved != alias {
		t.Fatalf("alias = %q, %v; want %q", alias, err, saved)
	}
	files.cfg.Offline = true
	for _, version := range []struct{ url, digest string }{{server.URL + "/plain", ""}, {server.URL + "/elsewhere", sumOf([]byte(body))}} {
		got, err := fetch.Execute(t.Context(), version.url, version.digest)
		if err != nil || got != saved {
			t.Fatalf("cached = %q, %v; want %q", got, err, saved)
		}
	}
	if calls.Load() != 2 {
		t.Errorf("requests = %d, want 2", calls.Load())
	}
}

func TestModelDownloadStatusNamesTheModel(t *testing.T) {
	const body = "named model"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte(body)); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	files := newTestProvisioner(t, false)
	files.cfg.Client = server.Client()
	var titles []string
	registry := NewObservedRegistry(func(req manifest.DependencyRequirement, status plugin.DependencyStatus) {
		if status.State == plugin.DependencyStateProvisioning && status.Message != "" {
			titles = append(titles, req.GetTitle()+"|"+status.Message)
		}
	}, files)
	fetch := plugindependency.NewFetch(func() pluginport.DependencyRegistry { return registry })
	digest := newModelRequirement(server.URL, []byte(body)).ChecksumSHA256
	if _, err := fetch.Execute(t.Context(), server.URL+"/rec/latin_rec.onnx", digest); err != nil {
		t.Fatal(err)
	}
	want := "latin_rec model|Downloading latin_rec model"
	if !slices.Contains(titles, want) {
		t.Fatalf("statuses = %v, want %q", titles, want)
	}
}
