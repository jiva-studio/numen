package container

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// countingDocumentHost is one document plugin at an endpoint, counting the
// references taken on it and given back.
type countingDocumentHost struct {
	port.PluginManager
	endpoint port.Endpoint
	acquired atomic.Int32
	released atomic.Int32
}

func (h *countingDocumentHost) ResolveCapability(string) (port.DiscoveredPlugin, error) {
	return port.DiscoveredPlugin{Manifest: port.PluginManifest{ID: "md.numen.pdf"}}, nil
}

func (h *countingDocumentHost) ResolveCapabilities(string) []port.DiscoveredPlugin { return nil }

func (h *countingDocumentHost) EnsureActive(context.Context, string) (port.Endpoint, error) {
	h.acquired.Add(1)
	return h.endpoint, nil
}

func (h *countingDocumentHost) Acquire(string)                          { h.acquired.Add(1) }
func (h *countingDocumentHost) Release(string)                          { h.released.Add(1) }
func (h *countingDocumentHost) CheckPluginPermission(_, _ string) error { return nil }
func (h *countingDocumentHost) GetVaultPath() string                    { return "" }

type pageCountingServer struct {
	numenv1connect.UnimplementedDocumentServiceHandler
}

func (pageCountingServer) GetDocument(
	context.Context, *connect.Request[v1.GetDocumentRequest],
) (*connect.Response[v1.GetDocumentResponse], error) {
	return connect.NewResponse(&v1.GetDocumentResponse{Pages: []*v1.Page{{Width: 1, Height: 1}}}), nil
}

type readSeekNopCloser struct{ *strings.Reader }

func (readSeekNopCloser) Close() error { return nil }

// A document opened and closed gives back every reference taken for it.
func TestAnOpenedDocumentGivesBackEveryReference(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewDocumentServiceHandler(pageCountingServer{}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	host := &countingDocumentHost{endpoint: port.Endpoint{Host: "127.0.0.1", Port: server.Listener.Addr().(*net.TCPAddr).Port}}
	cfg := Config{PluginHost: host}

	doc, err := cfg.PageRenderer().OpenDocument(t.Context(), readSeekNopCloser{strings.NewReader("%PDF")})
	if err != nil {
		t.Fatal(err)
	}
	if doc.GetPageCount() != 1 {
		t.Fatalf("the document has %d pages", doc.GetPageCount())
	}
	if host.released.Load() >= host.acquired.Load() {
		t.Errorf("an open document holds no reference: acquired %d, released %d", host.acquired.Load(), host.released.Load())
	}
	doc.Close()
	if host.acquired.Load() != host.released.Load() {
		t.Errorf("acquired %d, released %d", host.acquired.Load(), host.released.Load())
	}
}
