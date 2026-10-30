package container

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

var (
	_ pluginport.PluginManager          = (*plugin.Host)(nil)
	_ pluginport.PluginLifecycleManager = (*plugin.Host)(nil)
	_ pluginport.PluginRegistryManager  = (*plugin.Host)(nil)
)

func TestContainerSafeModeDegradation(t *testing.T) {
	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "doc")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"id": "md.numen.pdf",
		"name": "PDF",
		"version": "1.0.0",
		"capabilities": ["embedding"],
		"targets": ["desktop.editor"],
		"permissions": ["vault:read"],
		"backend": {"type": "process", "entrypoint": "bin/doc", "serviceName": "numen.sample.v1.SampleService"}
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		SafeMode:   true,
		PluginHost: NewPluginHost("desktop.editor", tmp),
	}

	ctx := t.Context()

	t.Run("embedder reports not ready", func(t *testing.T) {
		if cfg.EmbedderReady() {
			t.Fatal("expected EmbedderReady to report false in safe mode")
		}
		indexing, asking, closer, err := cfg.Embedders(ctx, nil)
		if err != nil {
			t.Fatalf("expected nil error from Embedders in safe mode, got %v", err)
		}
		if indexing != nil || asking != nil {
			t.Fatal("expected nil embedders from Embedders in safe mode")
		}
		if err := closer(); err != nil {
			t.Fatal(err)
		}

		embedder, closer, err := cfg.Embedder(ctx)
		if err != nil {
			t.Fatalf("expected nil error from Embedder in safe mode, got %v", err)
		}
		if embedder != nil {
			t.Fatal("expected nil embedder from Embedder in safe mode")
		}
		if err := closer(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("plugin processes were never started", func(t *testing.T) {
		for _, info := range cfg.PluginHost.GetPluginInfos() {
			if info.IsActive {
				t.Fatalf("expected plugin %s to remain inactive in safe mode", info.ID)
			}
		}
	})
}

// An installation without a plugin host has no plugin installed, and its
// plugins are set and stopped without one.
func TestAnInstallationWithoutAPluginHostHasNoPluginInstalled(t *testing.T) {
	cfg := Config{}
	_, err := cfg.resolvePlugin(bridge.CapabilityEmbedding)
	var missing *pluginport.UnavailableError
	if !errors.As(err, &missing) || missing.Reason != pluginport.PluginNotInstalled || missing.Capability != bridge.CapabilityEmbedding {
		t.Fatalf("resolvePlugin = %v, want the embedding plugin not installed", err)
	}
	cfg.SetPluginVaultPath("vault")
	if err := cfg.StopPlugins(t.Context()); err != nil {
		t.Fatalf("StopPlugins = %v, want nothing to stop", err)
	}
	if !cfg.SetSafeMode(true).SafeMode {
		t.Fatal("safe mode was not set")
	}
}

// Safe mode and the vault reach the plugin host the configuration holds.
func TestTheConfigurationSetsItsPluginHost(t *testing.T) {
	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "ocr")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"id": "md.numen.engines.ocr", "name": "OCR", "version": "1.0.0", "capabilities": ["embedding"],
		"targets": ["desktop.editor"], "backend": {"type": "process", "entrypoint": "bin/engines.ocr"}}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	host := NewPluginHost("desktop.editor", tmp)
	cfg := Config{PluginHost: host}

	cfg.SetPluginVaultPath("vault")
	if got := host.GetVaultPath(); got != "vault" {
		t.Errorf("the host holds the vault %q", got)
	}
	if !cfg.EmbedderReady() {
		t.Fatal("an installed plugin is not ready")
	}
	safe := cfg.SetSafeMode(true)
	cfg.SafeMode = false
	if cfg.EmbedderReady() {
		t.Error("the host started a plugin in safe mode")
	}
	if !safe.SafeMode {
		t.Error("safe mode was not set")
	}
	if !cfg.SetSafeMode(false).EmbedderReady() {
		t.Error("the host stayed in safe mode once it was left")
	}
}
