package container

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var _ port.PluginManager = (*plugin.Host)(nil)

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
		if !errors.Is(err, errDocumentUnavailable) {
			t.Fatalf("expected errDocumentUnavailable, got %v", err)
		}
	})

	t.Run("book extractor returns degraded fallback", func(t *testing.T) {
		_, err := cfg.TextExtractor().ReadBook(ctx, []byte("fake"))
		if !errors.Is(err, errDocumentUnavailable) {
			t.Fatalf("expected errDocumentUnavailable, got %v", err)
		}
	})

	t.Run("page renderer returns degraded fallback", func(t *testing.T) {
		renderer := cfg.PageRenderer()
		_, err := renderer.OpenDocument(ctx, nil)
		if !errors.Is(err, errDocumentUnavailable) {
			t.Fatalf("expected errDocumentUnavailable, got %v", err)
		}
	})

	t.Run("transcriber reports not ready", func(t *testing.T) {
		if cfg.TranscriberReady() {
			t.Fatal("expected TranscriberReady to report false in safe mode")
		}
		_, _, err := cfg.Transcriber(ctx)
		if !errors.Is(err, errTranscriptionUnavailable) {
			t.Fatalf("expected errTranscriptionUnavailable, got %v", err)
		}
	})

	t.Run("recogniser reports not ready", func(t *testing.T) {
		if cfg.RecogniserReady() {
			t.Fatal("expected RecogniserReady to report false in safe mode")
		}
		_, _, err := cfg.Recogniser(ctx)
		if !errors.Is(err, errRecognitionUnavailable) {
			t.Fatalf("expected errRecognitionUnavailable, got %v", err)
		}
		if cfg.isRecognitionPrepared() {
			t.Fatal("expected isRecognitionPrepared to report false in safe mode")
		}
		_, _, err = cfg.openRecognitionRuntime(ctx, nil)
		if !errors.Is(err, errRecognitionUnavailable) {
			t.Fatalf("expected openRecognitionRuntime to return errRecognitionUnavailable, got %v", err)
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
