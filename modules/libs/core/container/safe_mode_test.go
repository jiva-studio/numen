package container

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
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
		"capabilities": ["document", "recognition", "transcription", "proofreading"],
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
	})

	t.Run("proofreader reports not ready", func(t *testing.T) {
		if cfg.ProofreaderReady() {
			t.Fatal("expected ProofreaderReady to report false in safe mode")
		}
	})
}
