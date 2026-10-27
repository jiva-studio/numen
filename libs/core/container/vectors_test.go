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

// writeEmbeddingPlugin installs a plugin with the manifest given and returns
// the host that discovered it.
func writeEmbeddingPlugin(t *testing.T, manifest string) *plugin.Host {
	t.Helper()
	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "embeddings")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	host := plugin.NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatal(err)
	}
	return host
}

// The provider is the plugin that declares the capability. A plugin that only
// names the embedding service is not asked for vectors.
func TestAPluginWithoutTheEmbeddingCapabilityIsNotTheProvider(t *testing.T) {
	host := writeEmbeddingPlugin(t, `{
		"id": "test.vectors",
		"name": "Vectors",
		"version": "1.0.0",
		"targets": ["desktop.editor"],
		"backend": {"type": "process", "entrypoint": "bin/vectors", "serviceName": "numen.v1.EmbeddingService"}
	}`)
	cfg := container.Config{PluginHost: host}

	if cfg.EmbedderReady() {
		t.Error("a plugin without the embedding capability is the provider")
	}
	embedder, _, why := cfg.Embedder(t.Context())
	if why != nil || embedder != nil {
		t.Errorf("got embedder %v, error %v; want neither", embedder, why)
	}
}

// With the provider disabled, the vault is read and searched by its words.
func TestSearchWithDisabledEmbeddingProviderMatchesWords(t *testing.T) {
	host := writeEmbeddingPlugin(t, `{
		"id": "test.vectors",
		"name": "Vectors",
		"version": "1.0.0",
		"capabilities": ["embedding"],
		"targets": ["desktop.editor"],
		"backend": {"type": "process", "entrypoint": "bin/vectors"}
	}`)
	host.SetSettings(map[string]map[string]any{"test.vectors": {"enabled": false}})
	cfg, vault := makeScannedVault(t)
	cfg.PluginHost = host

	embedder, closer, why := cfg.Embedder(t.Context())
	if why != nil || embedder != nil {
		t.Fatalf("got embedder %v, error %v; want neither", embedder, why)
	}
	defer closer()
	db := read(t, cfg)
	if _, err := cfg.ReadWholeVault(t.Context(), db, embedder, vault); err != nil {
		t.Fatal(err)
	}
	found, err := cfg.NewSearch(db, embedder, nil).Execute(t.Context(), vault, "disorder", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Source != "Entropy.md" {
		t.Errorf("the words answered with %+v", found)
	}
}
