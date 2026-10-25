package container

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin"
)

func TestDownloaderWithoutHost(t *testing.T) {
	cfg := Config{}
	if cfg.DownloaderReady() {
		t.Fatalf("expected DownloaderReady to be false")
	}
	dl := cfg.Downloader(t.Context())
	if _, ok := dl.(unavailableDownloader); !ok {
		t.Fatalf("expected unavailableDownloader, got %T", dl)
	}
}

func TestDownloaderWithoutPlugin(t *testing.T) {
	tmp := t.TempDir()
	host := plugin.NewHost("desktop.editor", tmp)
	_ = host.Discover()

	cfg := Config{PluginHost: host}
	if cfg.DownloaderReady() {
		t.Fatalf("expected DownloaderReady to be false")
	}
	dl := cfg.Downloader(t.Context())
	if _, ok := dl.(unavailableDownloader); !ok {
		t.Fatalf("expected unavailableDownloader, got %T", dl)
	}
}

func TestDownloaderWithPlugin(t *testing.T) {
	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "downloader")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	manifest := `{
		"id": "md.numen.downloader",
		"name": "Media Downloader",
		"version": "1.0.0",
		"capabilities": ["downloader"],
		"targets": ["desktop.editor"],
		"backend": {
			"type": "process",
			"entrypoint": "bin/downloader",
			"serviceName": "numen.v1.DownloadService"
		}
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}

	host := plugin.NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("discover failed: %v", err)
	}

	cfg := Config{PluginHost: host}
	if !cfg.DownloaderReady() {
		t.Fatalf("expected DownloaderReady to be true")
	}
	dl := cfg.Downloader(t.Context())
	if _, ok := dl.(hostDownloaderPort); !ok {
		t.Fatalf("expected hostDownloaderPort, got %T", dl)
	}
}
