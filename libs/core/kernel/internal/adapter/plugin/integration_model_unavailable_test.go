package plugin_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/embedderbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/plugintest"
)

// stubModelHost resolves every capability to one plugin listening at the endpoint given.
type stubModelHost struct {
	endpoint transport.Endpoint
	acquired atomic.Int32
	released atomic.Int32
}

func (h *stubModelHost) ResolveCapability(string) (manifest.Discovered, error) {
	return manifest.Discovered{Manifest: manifest.Manifest{ID: "md.numen.stub"}}, nil
}

// WatchPluginConfig registers nothing: the stub is never configured.
func (h *stubModelHost) WatchPluginConfig(string, func()) func() { return func() {} }

func (h *stubModelHost) ResolveService(string) (manifest.Discovered, bool) {
	return manifest.Discovered{}, false
}

func (h *stubModelHost) GetPlugin(string) (manifest.Discovered, bool) {
	return manifest.Discovered{Manifest: manifest.Manifest{ID: "md.numen.stub"}}, true
}

func (h *stubModelHost) EnsureActive(context.Context, string) (transport.Endpoint, error) {
	h.acquired.Add(1)
	return h.endpoint, nil
}

func (h *stubModelHost) Acquire(string)                          { h.acquired.Add(1) }
func (h *stubModelHost) Release(string)                          { h.released.Add(1) }
func (h *stubModelHost) CheckPluginPermission(_, _ string) error { return nil }
func (h *stubModelHost) GetVaultPath() string                    { return "" }

func newDownModelHost(t *testing.T) *stubModelHost {
	t.Helper()
	server := httptest.NewServer(http.NotFoundHandler())
	serverPort := plugintest.GetPort(t, server)
	server.Close()
	return &stubModelHost{endpoint: transport.Endpoint{Host: "127.0.0.1", Port: serverPort}}
}

func newSilentModelHost(t *testing.T) *stubModelHost {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)
	return &stubModelHost{endpoint: transport.Endpoint{Host: "127.0.0.1", Port: plugintest.GetPort(t, server)}}
}

type modelOpener struct {
	name string
	open func(ctx context.Context, h *stubModelHost) error
}

func modelOpeners() []modelOpener {
	return []modelOpener{
		{"embedder", func(ctx context.Context, h *stubModelHost) error {
			b, err := embedderbridge.Open(ctx, h)
			if b != nil {
				return errors.New("a bridge was returned with the error")
			}
			return err
		}},
	}
}

func TestEmbedderOpen_RefusesDownPlugin(t *testing.T) {
	for _, tc := range modelOpeners() {
		t.Run(tc.name, func(t *testing.T) {
			host := newDownModelHost(t)
			if err := tc.open(t.Context(), host); err == nil {
				t.Fatal("a bridge was opened on a plugin that is down")
			} else if err.Error() == "" {
				t.Fatal("the error says nothing")
			}
			if host.acquired.Load() != host.released.Load() {
				t.Errorf("acquired %d times, released %d times", host.acquired.Load(), host.released.Load())
			}
		})
	}
}

func TestEmbedderOpen_RefusesTimedOutPlugin(t *testing.T) {
	for _, tc := range modelOpeners() {
		t.Run(tc.name, func(t *testing.T) {
			host := newSilentModelHost(t)
			ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
			defer cancel()
			if err := tc.open(ctx, host); err == nil {
				t.Fatal("a bridge was opened on a plugin that never answered")
			}
			if host.acquired.Load() != host.released.Load() {
				t.Errorf("acquired %d times, released %d times", host.acquired.Load(), host.released.Load())
			}
		})
	}
}

func (h *stubModelHost) ResolveNoteProfile(domain.NoteType) (domain.NoteProfile, bool) {
	return domain.NoteProfile{}, false
}
