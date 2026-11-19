package container

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

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
		"capabilities": ["search.source"],
		"targets": ["desktop.editor"],
		"permissions": ["vault:read"],
		"services": ["md.numen.readers.pdf.SampleService"], "backend": {"entrypoint": "bin/doc"}
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := NewTestInstallation(Config{SafeMode: true}, newPluginHost(t, "desktop.editor", tmp))

	ctx := t.Context()

	t.Run("embedder reports not ready", func(t *testing.T) {
		if embedderReady(sourceOf(cfg)) {
			t.Fatal("expected EmbedderReady to report false in safe mode")
		}
		indexing, asking, closer, err := cfg.Embedders(ctx)
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
		for _, info := range cfg.Execution.Runtime.GetPluginInfos() {
			if info.IsActive {
				t.Fatalf("expected plugin %s to remain inactive in safe mode", info.ID)
			}
		}
	})
}

func TestAnInstallationWithoutAPluginHostHasNoPluginInstalled(t *testing.T) {
	cfg := NewTestInstallation(Config{}, nil)
	_, err := getRunningExecution(cfg.Execution, cfg.SafeMode).Runtime.ResolveCapability(bridge.CapabilitySearchSource)
	var missing *plugindomain.UnavailableError
	if !errors.As(err, &missing) || missing.Reason != plugindomain.NotInstalled || missing.Capability != bridge.CapabilitySearchSource {
		t.Fatalf("resolvePlugin = %v, want the embedding plugin not installed", err)
	}
	cfg.Execution.Runtime.PluginBinding.SetVaultPath("vault")
	if err := StopPlugins(cfg.Execution.Watch, t.Context()); err != nil {
		t.Fatalf("StopPlugins = %v, want nothing to stop", err)
	}
	safe := NewTestInstallation(Config{SafeMode: true}, nil)
	_, err = getRunningExecution(safe.Execution, safe.SafeMode).Runtime.ResolveCapability(bridge.CapabilitySearchSource)
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
	manifest := `{"id": "md.numen.engines.ocr", "name": "OCR", "version": "1.0.0", "capabilities": ["search.source"],
		"targets": ["desktop.editor"], "backend": {"entrypoint": "bin/engines.ocr"}}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	host := openPluginRoles(t, "desktop.editor", tmp)
	cfg := NewTestInstallation(Config{}, AdaptTestPlugins(host.Management, host.Supervision))

	cfg.Execution.Runtime.PluginBinding.SetVaultPath("vault")
	if got := host.Management.Mode.GetVaultPath(); got != "vault" {
		t.Errorf("the host holds the vault %q", got)
	}
	if !embedderReady(sourceOf(cfg)) {
		t.Fatal("an installed plugin is not ready")
	}
	safe := NewTestInstallation(Config{SafeMode: true}, cfg.TestPlugins)
	if embedderReady(sourceOf(safe)) {
		t.Error("the host started a plugin in safe mode")
	}
	if !embedderReady(sourceOf(NewTestInstallation(Config{}, cfg.TestPlugins))) {
		t.Error("the host stayed in safe mode once it was left")
	}
}
