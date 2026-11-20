package provision

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()
	files := newTestProvisioner(t, false)
	files.cfg.Client = server.Client()
	registry := NewRegistry(files)
	var downloaded atomic.Int64
	registry.SetObserver(func(_ manifest.DependencyRequirement, status plugin.DependencyStatus) {
		downloaded.Add(status.BytesDownloaded)
	})
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
