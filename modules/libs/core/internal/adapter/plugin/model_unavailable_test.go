package plugin

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// stubModelHost resolves every capability to one plugin listening at the endpoint given.
type stubModelHost struct {
	endpoint port.Endpoint
	acquired atomic.Int32
	released atomic.Int32
}

func (h *stubModelHost) ResolveCapability(string) (port.DiscoveredPlugin, error) {
	return port.DiscoveredPlugin{Manifest: port.PluginManifest{ID: "md.numen.stub"}}, nil
}

func (h *stubModelHost) ResolveCapabilities(string) []port.DiscoveredPlugin { return nil }
func (h *stubModelHost) ResolveService(string) (port.DiscoveredPlugin, bool) {
	return port.DiscoveredPlugin{}, false
}

func (h *stubModelHost) GetPlugin(string) (port.DiscoveredPlugin, bool) {
	return port.DiscoveredPlugin{Manifest: port.PluginManifest{ID: "md.numen.stub"}}, true
}

func (h *stubModelHost) EnsureActive(context.Context, string) (port.Endpoint, error) {
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
	listening := server.Listener.Addr().(*net.TCPAddr).Port
	server.Close()
	return &stubModelHost{endpoint: port.Endpoint{Host: "127.0.0.1", Port: listening}}
}

// newSilentModelHost is a host whose plugin accepts a request and never answers it.
func newSilentModelHost(t *testing.T) *stubModelHost {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)
	return &stubModelHost{endpoint: port.Endpoint{Host: "127.0.0.1", Port: server.Listener.Addr().(*net.TCPAddr).Port}}
}

type modelOpener struct {
	name string
	open func(ctx context.Context, h *stubModelHost) error
}

func modelOpeners() []modelOpener {
	return []modelOpener{
		{"embedder", func(ctx context.Context, h *stubModelHost) error {
			b, err := OpenEmbedderBridge(ctx, h)
			if b != nil {
				return errors.New("a bridge was returned with the error")
			}
			return err
		}},
		{"recogniser", func(ctx context.Context, h *stubModelHost) error {
			b, err := OpenRecogniserBridge(ctx, h)
			if b != nil {
				return errors.New("a bridge was returned with the error")
			}
			return err
		}},
		{"transcriber", func(ctx context.Context, h *stubModelHost) error {
			b, err := OpenTranscriberBridge(ctx, h)
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
	bridge, err := OpenDownloaderBridge(t.Context(), host)
	if err != nil {
		t.Fatal(err)
	}
	model, err := bridge.GetDownloadModel(t.Context(), domain.URL("https://example.com/video"))
	if err == nil {
		t.Fatal("a download model was named by a plugin that is down")
	}
	if model != (port.DownloadModel{}) {
		t.Fatalf("a model was returned with the error: %+v", model)
	}
}

func (h *stubModelHost) ResolveNoteProfile(domain.NoteType) (domain.NoteProfile, bool) {
	return domain.NoteProfile{}, false
}
