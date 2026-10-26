package container_test

import (
	"context"
	"github.com/jiva-studio/numen/libs/notes/domain"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/core/container"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// downEmbeddingHost resolves an embedding plugin whose process is not listening.
type downEmbeddingHost struct {
	pluginport.PluginManager
	endpoint pluginport.Endpoint
	released atomic.Int32
	acquired atomic.Int32
}

func (h *downEmbeddingHost) FindPluginByCapability(string) (pluginport.DiscoveredPlugin, bool) {
	return pluginport.DiscoveredPlugin{Manifest: pluginport.PluginManifest{ID: "md.numen.embeddings"}}, true
}

func (h *downEmbeddingHost) ResolveCapability(string) (pluginport.DiscoveredPlugin, error) {
	return pluginport.DiscoveredPlugin{Manifest: pluginport.PluginManifest{ID: "md.numen.embeddings"}}, nil
}

func (h *downEmbeddingHost) ResolveService(string) (pluginport.DiscoveredPlugin, bool) {
	return pluginport.DiscoveredPlugin{}, false
}

func (h *downEmbeddingHost) EnsureActive(context.Context, string) (pluginport.Endpoint, error) {
	h.acquired.Add(1)
	return h.endpoint, nil
}

func (h *downEmbeddingHost) Release(string) { h.released.Add(1) }

func newDownEmbeddingHost(t *testing.T) *downEmbeddingHost {
	t.Helper()
	server := httptest.NewServer(http.NotFoundHandler())
	serverPort := server.Listener.Addr().(*net.TCPAddr).Port
	server.Close()
	return &downEmbeddingHost{endpoint: pluginport.Endpoint{Host: "127.0.0.1", Port: serverPort}}
}

// A plugin that does not answer names no model, and a vault read without a
// model leaves the vectors of the index where they are.
func TestAnEmbeddingPluginThatIsDownPreservesTheVectorsOfTheIndex(t *testing.T) {
	db, v := makeSwitchedIndex(t)
	host := newDownEmbeddingHost(t)
	target := container.Config{
		Embedding:          container.DefaultEmbedding(),
		ServiceDir:         ".numen",
		PluginHost:         host,
		ShouldRebuildIndex: true,
	}

	indexing, asking, _, err := target.Embedders(t.Context(), nil)
	if err == nil {
		t.Fatal("an embedding plugin that is down was opened")
	}
	if indexing != nil || asking != nil {
		t.Fatalf("an embedder was made without a model: %v, %v", indexing, asking)
	}
	if host.acquired.Load() != host.released.Load() {
		t.Errorf("acquired %d times, released %d times", host.acquired.Load(), host.released.Load())
	}

	whole, err := target.ReadWholeVault(t.Context(), db, indexing, v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := whole.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	if got := owingTo(t, db, v, unit{"first"}); got != 0 {
		t.Errorf("%d chunks lost the vector of the first model", got)
	}
}

func (h *downEmbeddingHost) ResolveNoteProfile(domain.NoteType) (domain.NoteProfile, bool) {
	return domain.NoteProfile{}, false
}
