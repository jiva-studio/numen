package container_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// A container nobody configured builds no embedder, and the machine's own
// settings are not read here.
func TestAContainerNobodyGaveSettingsEmbedsWithNothing(t *testing.T) {
	cfg := container.Config{}
	if cfg.EmbedderReady() {
		t.Error("embedder is ready when no plugin host is configured")
	}

	embedder, closer, why := cfg.Embedder(t.Context())
	if why != nil {
		t.Fatalf("want no reason, got %v", why)
	}
	if embedder != nil {
		t.Errorf("got an embedder: %v", embedder.GetModel())
	}
	if closer == nil {
		t.Error("expected non-nil no-op closer")
	} else if err := closer(); err != nil {
		t.Errorf("closer failed: %v", err)
	}

	indexing, asking, closer, why := cfg.Embedders(t.Context(), run.NewTasks())
	if why != nil {
		t.Fatalf("want no reason from Embedders, got %v", why)
	}
	if indexing != nil || asking != nil {
		t.Errorf("expected nil embedders when plugin absent, got indexing=%v asking=%v", indexing, asking)
	}
	if closer == nil {
		t.Error("expected non-nil closer")
	}
}

func TestAnEmbeddingPluginIsReadyWherePluginIsAvailable(t *testing.T) {
	cfg := container.Config{}
	if cfg.EmbedderReady() {
		t.Error("embedder is ready when no plugin host is configured")
	}

	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "embeddings")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"id": "md.numen.embeddings",
		"name": "Vector Embeddings",
		"version": "1.0.0",
		"capabilities": ["embedding"],
		"targets": ["desktop.editor"]
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	host := plugin.NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatal(err)
	}
	cfg.PluginHost = host

	if !cfg.EmbedderReady() {
		t.Error("embedder is not ready when embedding plugin is present")
	}
}

// When the embedding plugin is absent, search queries seamlessly fall back to
// lexical full-text search without throwing errors.
func TestSearchDegradesGracefullyWithoutPlugin(t *testing.T) {
	db, err := container.Config{IndexPath: filepath.Join(t.TempDir(), "index.db")}.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	cfg := container.Config{}
	var lastErr error
	s := cfg.NewSearch(db, nil, func(err error) {
		lastErr = err
	})

	results, err := s.Execute(t.Context(), domain.Vault{ID: "v1"}, "query", search.Parameters{Limit: 5})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if lastErr != nil {
		t.Fatalf("unexpected search error: %v", lastErr)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results on empty index, got %d", len(results))
	}
}
