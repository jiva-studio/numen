package integration_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/downloaderbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/embedderbridge"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestRealEmbeddingPluginProcessE2E(t *testing.T) {
	pluginsDir, err := filepath.Abs("../../../../../plugins")
	if err != nil {
		t.Fatalf("resolve plugins dir: %v", err)
	}

	embeddingsBin := filepath.Join(pluginsDir, "embeddings", "bin", "embeddings")
	if _, err := os.Stat(embeddingsBin); os.IsNotExist(err) {
		t.Skipf("Embeddings binary not found at %s; skipping real process E2E test", embeddingsBin)
	}

	host := plugin.NewHost("desktop.editor", pluginsDir)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	plugins := host.GetPlugins()
	if len(plugins) == 0 {
		t.Fatal("expected to discover embedding plugin in real modules/plugins")
	}

	ctx := t.Context()
	bridge, err := embedderbridge.Open(ctx, host)
	if err != nil {
		t.Fatalf("OpenEmbedderBridge with real binary failed: %v", err)
	}
	defer func() {
		_ = bridge.Close()
		_ = host.TeardownAll(ctx, 1*time.Second)
	}()

	model := bridge.GetModel()
	if model.Name == "" || model.Dimensions <= 0 {
		t.Fatalf("expected valid model from real embedding plugin, got %+v", model)
	}

	vectors, err := bridge.Embed(ctx, []string{"sample search query", "another sentence"})
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}
	if len(vectors) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vectors))
	}
	if len(vectors[0]) != model.Dimensions {
		t.Fatalf("expected vector length %d, got %d", model.Dimensions, len(vectors[0]))
	}
}

func TestRealDownloaderPluginProcessE2E(t *testing.T) {
	pluginsDir, err := filepath.Abs("../../../../../plugins")
	if err != nil {
		t.Fatalf("resolve plugins dir: %v", err)
	}

	downloaderBin := filepath.Join(pluginsDir, "downloader", "bin", "downloader")
	if _, err := os.Stat(downloaderBin); os.IsNotExist(err) {
		t.Skipf("downloader binary not found at %s; skipping real process E2E test", downloaderBin)
	}

	host := plugin.NewHost("desktop.editor", pluginsDir)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	ctx := t.Context()
	bridge, err := downloaderbridge.Open(ctx, host)
	if err != nil {
		t.Fatalf("OpenDownloaderBridge with real binary failed: %v", err)
	}
	defer func() {
		_ = bridge.Close()
		_ = host.TeardownAll(ctx, 1*time.Second)
	}()

	model, err := bridge.GetDownloadModel(ctx, domain.URL("https://example.com/test"))
	if err != nil {
		t.Fatalf("GetDownloadModel failed: %v", err)
	}
	if model.Producer == "" {
		t.Fatalf("expected valid producer from real downloader plugin, got %+v", model)
	}

	can, err := bridge.CanHandle(ctx, domain.URL("https://example.com/test"))
	if err != nil {
		t.Fatalf("CanHandle failed: %v", err)
	}
	if !can {
		t.Fatalf("expected CanHandle to be true for https://example.com/test")
	}
}
