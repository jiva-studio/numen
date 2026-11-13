package container_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// A container with no configuration builds no embedder, and the machine's own
// settings are not read here.
func TestAContainerWithoutSettingsEmbedsWithNothing(t *testing.T) {
	cfg := container.NewConfig(container.Config{})
	if cfg.EmbedderReady() {
		t.Error("embedder is ready when no plugin host is configured")
	}

	embedder, closer, why := cfg.Embedder(t.Context())
	if why != nil {
		t.Fatalf("want no reason, got %v", why)
	}
	if embedder != nil {
		t.Errorf("got an embedder: %v", embedder.Describe())
	}
	if closer == nil {
		t.Error("expected non-nil no-op closer")
	} else if err := closer(); err != nil {
		t.Errorf("closer failed: %v", err)
	}

	indexing, asking, closer, why := cfg.Embedders(t.Context())
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
	cfg := container.NewConfig(container.Config{})
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
		"capabilities": ["search.source"],
		"targets": ["desktop.editor"]
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	cfg.PluginHost = container.AdaptPluginHost(host)

	if !cfg.EmbedderReady() {
		t.Error("embedder is not ready when embedding plugin is present")
	}
}

// When the embedding plugin is absent, search queries seamlessly fall back to
// lexical full-text search without throwing errors.
func TestSearchDegradesGracefullyWithoutPlugin(t *testing.T) {
	db, err := container.OpenIndex(container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: filepath.Join(t.TempDir(), "index.db")},
	}), t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, db)

	cfg := container.NewConfig(container.Config{})
	var lastErr error
	s := container.NewSearch(cfg, db, nil, func(err error) {
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

func writeEmbeddingPlugin(t *testing.T, manifest string) *container.PluginRoles {
	t.Helper()
	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "embeddings")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	return host
}

// The provider is the plugin that declares the capability. A plugin that only
// names the embedding service is not queried for vectors.
func TestAPluginWithoutTheEmbeddingCapabilityIsNotTheProvider(t *testing.T) {
	host := writeEmbeddingPlugin(t, `{
		"id": "test.vectors",
		"name": "Vectors",
		"version": "1.0.0",
		"targets": ["desktop.editor"],
		"services": ["numen.v1.SearchSourceService"], "backend": {"entrypoint": "bin/vectors"}
	}`)
	cfg := container.NewConfig(container.Config{PluginHost: container.AdaptPluginHost(host)})

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
		"capabilities": ["search.source"],
		"targets": ["desktop.editor"],
		"backend": {"entrypoint": "bin/vectors"}
	}`)
	host.Settings.SetSettings(map[string]map[string]any{"test.vectors": {"enabled": false}})
	cfg, vault := makeScannedVault(t)
	cfg.PluginHost = container.AdaptPluginHost(host)

	embedder, closer, why := cfg.Embedder(t.Context())
	if why != nil || embedder != nil {
		t.Fatalf("got embedder %v, error %v; want neither", embedder, why)
	}
	defer func() {
		if err := closer(); err != nil {
			t.Error(err)
		}
	}()
	db := read(t, cfg)
	if _, err := container.ReadWholeVault(cfg, t.Context(), db, embedder, vault); err != nil {
		t.Fatal(err)
	}
	found, err := container.NewSearch(cfg, db, embedder, nil).Execute(t.Context(), vault, "disorder", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Source != "Entropy.md" {
		t.Errorf("the words answered with %+v", found)
	}
}
