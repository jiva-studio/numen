package container_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

// downEmbeddingHost resolves an embedding plugin whose process is not listening.
type downEmbeddingHost struct {
	endpoint transport.Endpoint
	released atomic.Int32
	acquired atomic.Int32
}

func (h *downEmbeddingHost) FindPluginByCapability(string) (manifest.DiscoveredPlugin, bool) {
	return manifest.DiscoveredPlugin{Manifest: manifest.Manifest{ID: "md.numen.embeddings"}}, true
}

func (h *downEmbeddingHost) ResolveCapability(string) (manifest.DiscoveredPlugin, error) {
	return manifest.DiscoveredPlugin{Manifest: manifest.Manifest{ID: "md.numen.embeddings"}}, nil
}

func (h *downEmbeddingHost) GetPlugin(string) (manifest.DiscoveredPlugin, bool) {
	return manifest.DiscoveredPlugin{}, false
}

func (h *downEmbeddingHost) ResolveService(string) (manifest.DiscoveredPlugin, bool) {
	return manifest.DiscoveredPlugin{}, false
}

func (h *downEmbeddingHost) EnsureActive(context.Context, string, plugindomain.Holder) (transport.Endpoint, error) {
	h.acquired.Add(1)
	return h.endpoint, nil
}

func (h *downEmbeddingHost) Release(string, string, plugindomain.Holder) { h.released.Add(1) }

func (h *downEmbeddingHost) EnsureActiveForQuestion(ctx context.Context, pluginID string, holder plugindomain.Holder) (transport.Endpoint, error) {
	return h.EnsureActive(ctx, pluginID, holder)
}

func (h *downEmbeddingHost) ReleaseQuestion(pluginID, token string, holder plugindomain.Holder) {
	h.Release(pluginID, token, holder)
}

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

func (h *downEmbeddingHost) lend() *container.TestPlugins {
	execution, _, _ := container.PreparePluginHost(nil, nil, nil, false)
	execution.Runtime.Resolver = h
	execution.Runtime.Activator = h
	execution.Runtime.ConfigWatcher = h
	return &container.TestPlugins{Execution: execution}
}

// A plugin that does not answer names no model: the window opens its source undescribed, and a vault read
// without a model leaves the vectors of the index where they are.
func TestDownEmbeddingPluginPreservesIndexVectors(t *testing.T) {
	db, v := makeSwitchedIndex(t)
	host := newDownEmbeddingHost(t)
	target := container.NewTestInstallation(container.Config{
		ShouldRebuildIndex: true,
		Paths:              container.Paths{ServiceDir: ".numen"},
	}, host.lend())

	indexed, _, closeSource, err := target.Embedders(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if indexed == nil || indexed.Describe().IsDescribed() {
		t.Fatalf("the source of a plugin that is down is %v, want one with no description", indexed)
	}

	whole, err := container.ReadWholeVault(target, t.Context(), db, indexed, v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := whole.Execute(t.Context(), v, indexing.ReadCallbacks{}); err != nil {
		t.Fatal(err)
	}
	if got := unembeddedTo(t, db, v, unit{"first"}); got != 0 {
		t.Errorf("%d chunks lost the vector of the first model", got)
	}
	if err := closeSource(); err != nil {
		t.Fatal(err)
	}
	if host.acquired.Load() != host.released.Load() {
		t.Errorf("acquired %d times, released %d times", host.acquired.Load(), host.released.Load())
	}
}

// WatchPluginConfig never calls back: the plugin is not configured while the test runs.
func (h *downEmbeddingHost) WatchPluginConfig(string, func()) (stop func()) { return func() {} }

func (h *downEmbeddingHost) ResolveNoteProfile(domain.NoteType) (domain.NoteProfile, bool) {
	return domain.NoteProfile{}, false
}
