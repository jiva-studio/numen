package plugin_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func TestHostSafeMode(t *testing.T) {
	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "sample")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"id": "org.example.sample",
		"name": "Sample",
		"version": "1.0.0",
		"capabilities": ["embedding"],
		"targets": ["desktop.editor"],
		"contributes": {
			"handles": [{"purpose": "text-provider", "contentTypes": ["application/pdf"]}]
		},
		"permissions": ["vault:read"],
		"backend": {"entrypoint": "bin/sample"}
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	host := plugin.NewHost(testsupport.GetMachine(), "desktop.editor", tmp)
	host.Teardown.SetSafeMode(true)

	if !host.Mode.IsSafeMode() {
		t.Fatal("expected IsSafeMode to report true")
	}

	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if plugins := host.Registry.GetPlugins(); len(plugins) != 0 {
		t.Fatalf("expected 0 plugins in safe mode, got %d", len(plugins))
	}

	if _, ok := host.Registry.GetPlugin("org.example.sample"); ok {
		t.Fatal("expected GetPlugin to report false in safe mode")
	}

	if _, ok := host.Registry.FindPluginByCapability("embedding"); ok {
		t.Fatal("expected FindPluginByCapability to report false in safe mode")
	}

	ctx := t.Context()
	if _, err := host.Activation.EnsureActive(ctx, "org.example.sample"); !errors.Is(err, plugin.ErrSafeModeActive) {
		t.Fatalf("expected errSafeModeActive, got %v", err)
	}

	activated, err := host.Events.TriggerEvent(ctx, "onFile:test.pdf")
	if err != nil {
		t.Fatalf("TriggerEvent error: %v", err)
	}
	if len(activated) != 0 {
		t.Fatalf("expected 0 activations in safe mode, got %d", len(activated))
	}
}
