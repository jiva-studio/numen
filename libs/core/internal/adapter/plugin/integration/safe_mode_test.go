package integration

import (
	"errors"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin"
	"os"
	"path/filepath"
	"testing"
)

func TestHostSafeMode(t *testing.T) {
	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "ocr")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"id": "md.numen.engines.ocr",
		"name": "OCR",
		"version": "1.0.0",
		"capabilities": ["embedding"],
		"targets": ["desktop.editor"],
		"activationEvents": ["onFile:*.pdf"],
		"permissions": ["vault:read"],
		"lifecycle": {"mode": "on-demand", "idleTimeoutSeconds": 10, "keepAliveWhileVisible": false},
		"backend": {"type": "process", "entrypoint": "bin/engines.ocr", "serviceName": "numen.v1.OcrService"}
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	host := plugin.NewHost("desktop.editor", tmp)
	host.SetSafeMode(true)

	if !host.IsSafeMode() {
		t.Fatal("expected IsSafeMode to report true")
	}

	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if plugins := host.GetPlugins(); len(plugins) != 0 {
		t.Fatalf("expected 0 plugins in safe mode, got %d", len(plugins))
	}

	if _, ok := host.GetPlugin("md.numen.engines.ocr"); ok {
		t.Fatal("expected GetPlugin to report false in safe mode")
	}

	if _, ok := host.FindPluginByCapability("embedding"); ok {
		t.Fatal("expected FindPluginByCapability to report false in safe mode")
	}

	ctx := t.Context()
	if _, err := host.EnsureActive(ctx, "md.numen.engines.ocr"); !errors.Is(err, plugin.ErrSafeModeActive) {
		t.Fatalf("expected errSafeModeActive, got %v", err)
	}

	activated, err := host.TriggerEvent(ctx, "onFile:test.pdf")
	if err != nil {
		t.Fatalf("TriggerEvent error: %v", err)
	}
	if len(activated) != 0 {
		t.Fatalf("expected 0 activations in safe mode, got %d", len(activated))
	}
}
