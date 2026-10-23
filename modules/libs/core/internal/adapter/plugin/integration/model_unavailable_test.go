package integration

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin/downloaderbridge"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin/embedderbridge"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin/recogniserbridge"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin/transcriberbridge"
	"github.com/jiva-studio/numen/modules/libs/core/port/downloadport"
	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
)

// stubModelHost resolves every capability to one plugin listening at the endpoint given.
type stubModelHost struct {
	endpoint pluginport.Endpoint
	acquired atomic.Int32
	released atomic.Int32
}

func (h *stubModelHost) ResolveCapability(string) (pluginport.DiscoveredPlugin, error) {
	return pluginport.DiscoveredPlugin{Manifest: pluginport.PluginManifest{ID: "md.numen.stub"}}, nil
}

func (h *stubModelHost) ResolveCapabilities(string) []pluginport.DiscoveredPlugin { return nil }
func (h *stubModelHost) ResolveService(string) (pluginport.DiscoveredPlugin, bool) {
	return pluginport.DiscoveredPlugin{}, false
}

func (h *stubModelHost) GetPlugin(string) (pluginport.DiscoveredPlugin, bool) {
	return pluginport.DiscoveredPlugin{Manifest: pluginport.PluginManifest{ID: "md.numen.stub"}}, true
}

func (h *stubModelHost) EnsureActive(context.Context, string) (pluginport.Endpoint, error) {
	h.acquired.Add(1)
	return h.endpoint, nil
}

func (h *stubModelHost) Acquire(string)                          { h.acquired.Add(1) }
func (h *stubModelHost) Release(string)                          { h.released.Add(1) }
func (h *stubModelHost) CheckPluginPermission(_, _ string) error { return nil }
func (h *stubModelHost) GetVaultPath() string                    { return "" }

func (h *stubModelHost) ValidatePluginPath(_, target string) (string, error) { return target, nil }

// newDownModelHost is a host whose plugin process is not listening.
func newDownModelHost(t *testing.T) *stubModelHost {
	t.Helper()
	server := httptest.NewServer(http.NotFoundHandler())
	serverPort := server.Listener.Addr().(*net.TCPAddr).Port
	server.Close()
	return &stubModelHost{endpoint: pluginport.Endpoint{Host: "127.0.0.1", Port: serverPort}}
}

// newSilentModelHost is a host whose plugin accepts a request and never answers it.
func newSilentModelHost(t *testing.T) *stubModelHost {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)
	return &stubModelHost{endpoint: pluginport.Endpoint{Host: "127.0.0.1", Port: server.Listener.Addr().(*net.TCPAddr).Port}}
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
		{"recogniser", func(ctx context.Context, h *stubModelHost) error {
			b, err := recogniserbridge.Open(ctx, h)
			if b != nil {
				return errors.New("a bridge was returned with the error")
			}
			return err
		}},
		{"transcriber", func(ctx context.Context, h *stubModelHost) error {
			b, err := transcriberbridge.Open(ctx, h)
			if b != nil {
				return errors.New("a bridge was returned with the error")
			}
			return err
		}},
	}
}

func TestABridgeIsNotOpenedOnAPluginThatIsDown(t *testing.T) {
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

func TestABridgeIsNotOpenedOnAPluginThatDoesNotAnswerInTime(t *testing.T) {
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

func TestADownloaderThatIsDownNamesNoModel(t *testing.T) {
	host := newDownModelHost(t)
	bridge, err := downloaderbridge.Open(t.Context(), host)
	if err != nil {
		t.Fatal(err)
	}
	model, err := bridge.GetDownloadModel(t.Context(), domain.URL("https://example.com/video"))
	if err == nil {
		t.Fatal("a download model was named by a plugin that is down")
	}
	if model != (downloadport.DownloadModel{}) {
		t.Fatalf("a model was returned with the error: %+v", model)
	}
}

func (h *stubModelHost) ResolveNoteProfile(domain.NoteType) (domain.NoteProfile, bool) {
	return domain.NoteProfile{}, false
}
