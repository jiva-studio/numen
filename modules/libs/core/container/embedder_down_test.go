package container_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// downEmbeddingHost resolves an embedding plugin whose process is not listening.
type downEmbeddingHost struct {
	port.PluginManager
	endpoint port.Endpoint
	released atomic.Int32
	acquired atomic.Int32
}

func (h *downEmbeddingHost) FindPluginByCapability(string) (port.DiscoveredPlugin, bool) {
	return port.DiscoveredPlugin{Manifest: port.PluginManifest{ID: "md.numen.embeddings"}}, true
}

func (h *downEmbeddingHost) ResolveCapability(string) (port.DiscoveredPlugin, error) {
	return port.DiscoveredPlugin{Manifest: port.PluginManifest{ID: "md.numen.embeddings"}}, nil
}

func (h *downEmbeddingHost) EnsureActive(context.Context, string) (port.Endpoint, error) {
	h.acquired.Add(1)
	return h.endpoint, nil
}

func (h *downEmbeddingHost) Release(string) { h.released.Add(1) }

func newDownEmbeddingHost(t *testing.T) *downEmbeddingHost {
	t.Helper()
	server := httptest.NewServer(http.NotFoundHandler())
	listening := server.Listener.Addr().(*net.TCPAddr).Port
	server.Close()
	return &downEmbeddingHost{endpoint: port.Endpoint{Host: "127.0.0.1", Port: listening}}
}

// A plugin that does not answer names no model, and a vault read without a
// model leaves the vectors of the index where they are.
func TestAnEmbeddingPluginThatIsDownLeavesTheVectorsOfTheIndex(t *testing.T) {
	db, v := makeSwitchedIndex(t)
	host := newDownEmbeddingHost(t)
	held := container.Config{
		Embedding:          container.DefaultEmbedding(),
		ServiceDir:         ".numen",
		PluginHost:         host,
		ShouldRebuildIndex: true,
	}

	indexing, asking, _, err := held.Embedders(t.Context(), nil)
	if err == nil {
		t.Fatal("an embedding plugin that is down was opened")
	}
	if indexing != nil || asking != nil {
		t.Fatalf("an embedder was made without a model: %v, %v", indexing, asking)
	}
	if host.acquired.Load() != host.released.Load() {
		t.Errorf("acquired %d times, released %d times", host.acquired.Load(), host.released.Load())
	}

	whole, err := held.ReadWholeVault(t.Context(), db, indexing, v)
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
