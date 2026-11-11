package plugin_test

import (
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/embedderbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func TestRealEmbeddingPluginProcessE2E(t *testing.T) {
	pluginsDir, err := filepath.Abs("../../../../plugins")
	if err != nil {
		t.Fatalf("resolve plugins dir: %v", err)
	}

	embeddingsBin := filepath.Join(pluginsDir, "embeddings", "bin", "embeddings")
	if _, err := os.Stat(embeddingsBin); os.IsNotExist(err) {
		t.Skipf("Embeddings binary not found at %s; skipping real process E2E test", embeddingsBin)
	}

	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", pluginsDir)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	plugins := host.Registry.GetPlugins()
	if len(plugins) == 0 {
		t.Fatal("expected to discover embedding plugin in real plugins/")
	}

	ctx := t.Context()
	bridge, err := embedderbridge.Open(ctx, pluginhost.Join(host))
	if err != nil {
		t.Fatalf("OpenEmbedderBridge with real binary failed: %v", err)
	}
	defer func() {
		testsupport.Close(t, bridge)
		if err := host.Teardown.TeardownAll(ctx, 1*time.Second); err != nil {
			t.Error(err)
		}
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
