package container_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"

	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// downEmbeddingHost resolves an embedding plugin whose process is not listening.
type downEmbeddingHost struct {
	pluginport.PluginResolver
	endpoint transport.Endpoint
	released atomic.Int32
	acquired atomic.Int32
}

func (h *downEmbeddingHost) FindPluginByCapability(string) (manifest.Discovered, bool) {
	return manifest.Discovered{Manifest: manifest.Manifest{ID: "md.numen.embeddings"}}, true
}

func (h *downEmbeddingHost) ResolveCapability(string) (manifest.Discovered, error) {
	return manifest.Discovered{Manifest: manifest.Manifest{ID: "md.numen.embeddings"}}, nil
}

func (h *downEmbeddingHost) ResolveService(string) (manifest.Discovered, bool) {
	return manifest.Discovered{}, false
}

func (h *downEmbeddingHost) EnsureActive(context.Context, string) (transport.Endpoint, error) {
	h.acquired.Add(1)
	return h.endpoint, nil
}

func (h *downEmbeddingHost) Release(string) { h.released.Add(1) }

func newDownEmbeddingHost(t *testing.T) *downEmbeddingHost {
	t.Helper()
	server := httptest.NewServer(http.NotFoundHandler())
	addr, isTCP := server.Listener.Addr().(*net.TCPAddr)
	if !isTCP {
		t.Fatalf("expected *net.TCPAddr, got %T", server.Listener.Addr())
	}
	serverPort := addr.Port
	server.Close()
	return &downEmbeddingHost{endpoint: transport.Endpoint{Host: "127.0.0.1", Port: serverPort}}
}

func (h *downEmbeddingHost) lend() *container.PluginHost {
	return &container.PluginHost{Runtime: container.PluginRuntime{PluginResolver: h, Activator: h, PluginBinding: container.AbsentBinding()}}
}

// A plugin that does not answer names no model, and a vault read without a
// model leaves the vectors of the index where they are.
func TestDownEmbeddingPluginPreservesIndexVectors(t *testing.T) {
	db, v := makeSwitchedIndex(t)
	host := newDownEmbeddingHost(t)
	target := container.NewConfig(container.Config{
		PluginHost:         host.lend(),
		ShouldRebuildIndex: true,
		Paths:              container.Paths{ServiceDir: ".numen"},
	})

	indexed, asking, _, err := target.Embedders(t.Context(), nil)
	if err == nil {
		t.Fatal("an embedding plugin that is down was opened")
	}
	if indexed != nil || asking != nil {
		t.Fatalf("an embedder was made without a model: %v, %v", indexed, asking)
	}
	if host.acquired.Load() != host.released.Load() {
		t.Errorf("acquired %d times, released %d times", host.acquired.Load(), host.released.Load())
	}

	whole, err := container.ReadWholeVault(target, t.Context(), db, indexed, v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := whole.Execute(t.Context(), v, indexing.Reporting{}); err != nil {
		t.Fatal(err)
	}
	if got := unembeddedTo(t, db, v, unit{"first"}); got != 0 {
		t.Errorf("%d chunks lost the vector of the first model", got)
	}
}

func (h *downEmbeddingHost) ResolveNoteProfile(domain.NoteType) (domain.NoteProfile, bool) {
	return domain.NoteProfile{}, false
}
