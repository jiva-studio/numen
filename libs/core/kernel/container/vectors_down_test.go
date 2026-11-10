package container_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
)

// openDownEmbeddingHost lends a host whose embedding plugin is installed and cannot start: the binary
// its manifest names is missing.
func openDownEmbeddingHost(t *testing.T) *container.PluginHost {
	t.Helper()
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "embeddings")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	text := `{
		"id": "md.numen.embeddings", "name": "Embeddings", "version": "1.0.0",
		"capabilities": ["embedding"], "targets": ["desktop.editor"], "permissions": [],
		"backend": {"entrypoint": "bin/embed"}
	}`
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return container.AdaptPluginHost(pluginhost.Open("desktop.editor", dir))
}

// A plugin that does not answer names no model, and a vault read without a
// model leaves the vectors of the index where they are.
func TestAnEmbeddingPluginThatIsDownPreservesTheVectorsOfTheIndex(t *testing.T) {
	db, v := makeSwitchedIndex(t)
	host := openDownEmbeddingHost(t)
	target := container.NewConfig(container.Config{
		PluginHost:         host,
		ShouldRebuildIndex: true,
		Paths:              container.Paths{ServiceDir: ".numen"},
	})

	indexing, asking, _, err := target.Embedders(t.Context(), nil)
	if err == nil {
		t.Fatal("an embedding plugin that is down was opened")
	}
	if indexing != nil || asking != nil {
		t.Fatalf("an embedder was made without a model: %v, %v", indexing, asking)
	}

	whole, err := container.ReadWholeVault(target, t.Context(), db, indexing, v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := whole.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	if got := unembeddedTo(t, db, v, unit{"first"}); got != 0 {
		t.Errorf("%d chunks lost the vector of the first model", got)
	}
}
