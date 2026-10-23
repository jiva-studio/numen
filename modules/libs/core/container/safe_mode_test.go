package container

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/completionport"
	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
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
		"capabilities": ["document", "recognition", "transcription", "proofreading", "embedding"],
		"targets": ["desktop.editor"],
		"permissions": ["vault:read"],
		"backend": {"type": "process", "entrypoint": "bin/doc", "serviceName": "numen.v1.DocumentService"}
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		SafeMode:   true,
		PluginHost: NewPluginHost("desktop.editor", tmp),
	}

	ctx := t.Context()

	t.Run("document extractor returns degraded fallback", func(t *testing.T) {
		extractor := cfg.TextExtractor()
		_, err := extractor.Read(ctx, []byte("fake"))
		if !isDisabledInSafeMode(err, bridge.CapabilityDocument) {
			t.Fatalf("got %v, want the document plugin disabled", err)
		}
	})

	t.Run("book extractor returns degraded fallback", func(t *testing.T) {
		_, err := cfg.TextExtractor().ReadBook(ctx, []byte("fake"))
		if !isDisabledInSafeMode(err, bridge.CapabilityDocument) {
			t.Fatalf("got %v, want the document plugin disabled", err)
		}
	})

	t.Run("page renderer returns degraded fallback", func(t *testing.T) {
		renderer := cfg.PageRenderer()
		_, err := renderer.OpenDocument(ctx, nil)
		if !isDisabledInSafeMode(err, bridge.CapabilityDocument) {
			t.Fatalf("got %v, want the document plugin disabled", err)
		}
	})

	t.Run("transcriber reports not ready", func(t *testing.T) {
		if cfg.TranscriberReady() {
			t.Fatal("expected TranscriberReady to report false in safe mode")
		}
		_, _, err := cfg.Transcriber(ctx)
		if !isDisabledInSafeMode(err, bridge.CapabilityTranscription) {
			t.Fatalf("got %v, want the transcription plugin disabled", err)
		}
	})

	t.Run("recogniser reports not ready", func(t *testing.T) {
		if cfg.RecogniserReady() {
			t.Fatal("expected RecogniserReady to report false in safe mode")
		}
		_, _, err := cfg.Recogniser(ctx)
		if !isDisabledInSafeMode(err, bridge.CapabilityRecognition) {
			t.Fatalf("got %v, want the recognition plugin disabled", err)
		}
	})

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
		_ = closer()

		embedder, closer, err := cfg.Embedder(ctx)
		if err != nil {
			t.Fatalf("expected nil error from Embedder in safe mode, got %v", err)
		}
		if embedder != nil {
			t.Fatal("expected nil embedder from Embedder in safe mode")
		}
		_ = closer()
	})

	t.Run("proofreader reports not ready", func(t *testing.T) {
		if cfg.ProofreaderReady() {
			t.Fatal("expected ProofreaderReady to report false in safe mode")
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

func TestTheTextsAnInstallationBindsAreTheTextExtractor(t *testing.T) {
	cfg := Config{SafeMode: true, Texts: bookReader{}}

	got, err := cfg.TextExtractor().ReadBook(t.Context(), []byte("fake"))
	if err != nil || got.Text != "bound" {
		t.Fatalf("ReadBook = %+v, %v, want the text of the extractor bound", got, err)
	}
}

// bookReader is an extractor an installation binds.
type bookReader struct{ unavailableDocument }

func (bookReader) ReadBook(context.Context, []byte) (domain.BookText, error) {
	return domain.BookText{Text: "bound"}, nil
}

// isDisabledInSafeMode is whether the error says the capability's plugin is
// disabled by safe mode, and is ErrUnavailable.
func isDisabledInSafeMode(err error, capability string) bool {
	var missing *pluginport.UnavailableError
	return errors.As(err, &missing) && missing.Reason == pluginport.PluginDisabled &&
		missing.Capability == capability && errors.Is(err, errSafeMode) && errors.Is(err, completionport.ErrUnavailable)
}

// An installation without a plugin host has no plugin installed, and its
// plugins are set and stopped without one.
func TestAnInstallationWithoutAPluginHostHasNoPluginInstalled(t *testing.T) {
	cfg := Config{}
	_, _, err := cfg.Recogniser(t.Context())
	var missing *pluginport.UnavailableError
	if !errors.As(err, &missing) || missing.Reason != pluginport.PluginNotInstalled || missing.Capability != bridge.CapabilityRecognition {
		t.Fatalf("Recogniser = %v, want the recognition plugin not installed", err)
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
	manifest := `{"id": "md.numen.engines.ocr", "name": "OCR", "version": "1.0.0", "capabilities": ["recognition"],
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
	if !cfg.RecogniserReady() {
		t.Fatal("an installed plugin is not ready")
	}
	safe := cfg.SetSafeMode(true)
	cfg.SafeMode = false
	if cfg.RecogniserReady() {
		t.Error("the host started a plugin in safe mode")
	}
	if !safe.SafeMode {
		t.Error("safe mode was not set")
	}
	if !cfg.SetSafeMode(false).RecogniserReady() {
		t.Error("the host stayed in safe mode once it was left")
	}
}
