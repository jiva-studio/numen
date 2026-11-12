package plugin_test

import (
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/sourcebridge"
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
	bridge, err := sourcebridge.Open(ctx, pluginhost.Join(host))
	if err != nil {
		t.Fatalf("OpenEmbedderBridge with real binary failed: %v", err)
	}
	defer func() {
		testsupport.Close(t, bridge)
		if err := host.Teardown.TeardownAll(ctx, 1*time.Second); err != nil {
			t.Error(err)
		}
	}()

	model := bridge.Describe()
	if model.Name == "" || model.Width <= 0 {
		t.Fatalf("expected valid model from real embedding plugin, got %+v", model)
	}

	vectors, err := bridge.IndexTexts(ctx, searchsource.Place{}, []searchsource.Text{{ChunkID: "1", Text: "sample search query"}, {ChunkID: "2", Text: "another sentence"}})
	if err != nil {
		t.Fatalf("IndexTexts failed: %v", err)
	}
	if len(vectors) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vectors))
	}
	if len(vectors[0]) != model.Width {
		t.Fatalf("expected vector length %d, got %d", model.Width, len(vectors[0]))
	}
}
