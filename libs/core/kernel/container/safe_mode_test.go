package container

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
)

func TestContainerSafeModeDegradation(t *testing.T) {
	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "doc")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"id": "md.numen.readers.pdf",
		"name": "PDF",
		"version": "1.0.0",
		"capabilities": ["embedding"],
		"targets": ["desktop.editor"],
		"permissions": ["vault:read"],
		"services": ["md.numen.readers.pdf.SampleService"], "backend": {"entrypoint": "bin/doc"}
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := NewConfig(Config{
		SafeMode:   true,
		PluginHost: newPluginHost(t, "desktop.editor", tmp),
	})

	ctx := t.Context()

	t.Run("embedder reports not ready", func(t *testing.T) {
		if embedderReady(searchOf(cfg)) {
			t.Fatal("expected EmbedderReady to report false in safe mode")
		}
		indexing, asking, closer, err := searchEmbedders(searchOf(cfg), ctx, nil)
		if err != nil {
			t.Fatalf("expected nil error from Embedders in safe mode, got %v", err)
		}
		if indexing != nil || asking != nil {
			t.Fatal("expected nil embedders from Embedders in safe mode")
		}
		if err := closer(); err != nil {
			t.Fatal(err)
		}

		embedder, closer, err := searchEmbedder(searchOf(cfg), ctx)
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
		for _, info := range cfg.PluginHost.Runtime.GetPluginInfos() {
			if info.IsActive {
				t.Fatalf("expected plugin %s to remain inactive in safe mode", info.ID)
			}
		}
	})
}

// An installation without a plugin host has no plugin installed, and its
// plugins are set and stopped without one.
func TestAnInstallationWithoutAPluginHostHasNoPluginInstalled(t *testing.T) {
	cfg := NewConfig(Config{})
	_, err := plugins(cfg).RunningHost.Runtime.ResolveCapability(bridge.CapabilityEmbedding)
	var missing *plugindomain.UnavailableError
	if !errors.As(err, &missing) || missing.Reason != plugindomain.NotInstalled || missing.Capability != bridge.CapabilityEmbedding {
		t.Fatalf("resolvePlugin = %v, want the embedding plugin not installed", err)
	}
	setPluginVaultPath(plugins(cfg), "vault")
	if err := cfg.StopPlugins(t.Context()); err != nil {
		t.Fatalf("StopPlugins = %v, want nothing to stop", err)
	}
	safe := NewConfig(Config{SafeMode: true})
	_, err = plugins(safe).RunningHost.Runtime.ResolveCapability(bridge.CapabilityEmbedding)
	if !errors.Is(err, errSafeMode) || !errors.As(err, &missing) || missing.Reason != plugindomain.Disabled {
		t.Fatalf("resolvePlugin in safe mode = %v, want the embedding plugin disabled by safe mode", err)
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
		"targets": ["desktop.editor"], "backend": {"entrypoint": "bin/engines.ocr"}}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	host := pluginhost.Open(t, "desktop.editor", tmp)
	cfg := NewConfig(Config{PluginHost: AdaptPluginHost((*PluginRoles)(host))})

	setPluginVaultPath(plugins(cfg), "vault")
	if got := host.Mode.GetVaultPath(); got != "vault" {
		t.Errorf("the host holds the vault %q", got)
	}
	if !embedderReady(searchOf(cfg)) {
		t.Fatal("an installed plugin is not ready")
	}
	safe := NewConfig(Config{PluginHost: cfg.PluginHost, SafeMode: true})
	if embedderReady(searchOf(safe)) {
		t.Error("the host started a plugin in safe mode")
	}
	if !embedderReady(searchOf(NewConfig(Config{PluginHost: cfg.PluginHost}))) {
		t.Error("the host stayed in safe mode once it was left")
	}
}
