package plugin

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// leaseHost resolves every capability to one stub plugin and counts the references taken and given back.
type leaseHost struct {
	endpoint port.Endpoint
	acquired atomic.Int32
	released atomic.Int32
}

func (h *leaseHost) FindPluginByCapability(string) (port.DiscoveredPlugin, bool) {
	return port.DiscoveredPlugin{Manifest: port.PluginManifest{ID: "md.numen.stub"}}, true
}

func (h *leaseHost) FindPluginByFileExtension(string) (port.DiscoveredPlugin, bool) {
	return port.DiscoveredPlugin{Manifest: port.PluginManifest{ID: "md.numen.stub"}}, true
}

func (h *leaseHost) GetPlugin(string) (port.DiscoveredPlugin, bool) {
	return port.DiscoveredPlugin{Manifest: port.PluginManifest{ID: "md.numen.stub"}}, true
}

func (h *leaseHost) EnsureActive(context.Context, string) (port.Endpoint, error) {
	h.acquired.Add(1)
	return h.endpoint, nil
}

func (h *leaseHost) Acquire(string) { h.acquired.Add(1) }
func (h *leaseHost) Release(string) { h.released.Add(1) }

func (h *leaseHost) CheckPluginPermission(_, _ string) error { return nil }
func (h *leaseHost) GetVaultPath() string                    { return "" }

func (h *leaseHost) ValidatePluginPath(_, target string) (string, error) { return target, nil }

type stubDocumentServer struct {
	numenv1connect.UnimplementedDocumentServiceHandler
}

func (stubDocumentServer) GetDocument(
	context.Context, *connect.Request[v1.GetDocumentRequest],
) (*connect.Response[v1.GetDocumentResponse], error) {
	return connect.NewResponse(&v1.GetDocumentResponse{}), nil
}

type stubEmbeddingServer struct {
	numenv1connect.UnimplementedEmbeddingServiceHandler
}

func (stubEmbeddingServer) GetEmbeddingModel(
	context.Context, *connect.Request[v1.GetEmbeddingModelRequest],
) (*connect.Response[v1.GetEmbeddingModelResponse], error) {
	return connect.NewResponse(&v1.GetEmbeddingModelResponse{}), nil
}

type stubRecognitionServer struct {
	numenv1connect.UnimplementedRecognitionServiceHandler
}

func (stubRecognitionServer) GetRecognitionModel(
	context.Context, *connect.Request[v1.GetRecognitionModelRequest],
) (*connect.Response[v1.GetRecognitionModelResponse], error) {
	return connect.NewResponse(&v1.GetRecognitionModelResponse{}), nil
}

type stubTranscriptionServer struct {
	numenv1connect.UnimplementedTranscriptionServiceHandler
}

func (stubTranscriptionServer) GetTranscriptionModel(
	context.Context, *connect.Request[v1.GetTranscriptionModelRequest],
) (*connect.Response[v1.GetTranscriptionModelResponse], error) {
	return connect.NewResponse(&v1.GetTranscriptionModelResponse{}), nil
}

func newLeaseHost(t *testing.T) *leaseHost {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(&mockAgentServer{}))
	mux.Handle(numenv1connect.NewDocumentServiceHandler(stubDocumentServer{}))
	mux.Handle(numenv1connect.NewEmbeddingServiceHandler(stubEmbeddingServer{}))
	mux.Handle(numenv1connect.NewRecognitionServiceHandler(stubRecognitionServer{}))
	mux.Handle(numenv1connect.NewTranscriptionServiceHandler(stubTranscriptionServer{}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &leaseHost{endpoint: port.Endpoint{Host: "127.0.0.1", Port: server.Listener.Addr().(*net.TCPAddr).Port}}
}

func TestADoubleCloseGivesTheReferenceBackOnce(t *testing.T) {
	bridges := []struct {
		name string
		open func(ctx context.Context, h *leaseHost) (io.Closer, error)
	}{
		{"embedder", func(ctx context.Context, h *leaseHost) (io.Closer, error) { return OpenEmbedderBridge(ctx, h) }},
		{"recogniser", func(ctx context.Context, h *leaseHost) (io.Closer, error) { return OpenRecogniserBridge(ctx, h) }},
		{"transcriber", func(ctx context.Context, h *leaseHost) (io.Closer, error) { return OpenTranscriberBridge(ctx, h) }},
		{"downloader", func(ctx context.Context, h *leaseHost) (io.Closer, error) { return OpenDownloaderBridge(ctx, h) }},
		{"document", func(ctx context.Context, h *leaseHost) (io.Closer, error) { return OpenDocumentBridge(ctx, h) }},
		{"book", func(ctx context.Context, h *leaseHost) (io.Closer, error) { return OpenBookBridge(ctx, h) }},
		{"agent", func(ctx context.Context, h *leaseHost) (io.Closer, error) { return OpenAgentBridge(ctx, h) }},
		{"proofreader", func(_ context.Context, h *leaseHost) (io.Closer, error) { return OpenProofreaderBridge(h) }},
	}
	for _, tc := range bridges {
		t.Run(tc.name, func(t *testing.T) {
			host := newLeaseHost(t)
			bridge, err := tc.open(t.Context(), host)
			if err != nil {
				t.Fatal(err)
			}
			held := host.acquired.Load()
			if got := host.released.Load(); got != 0 {
				t.Fatalf("%d references were given back before Close", got)
			}

			for range 2 {
				if err := bridge.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if got := host.released.Load(); got != held {
				t.Errorf("held %d references, gave back %d", held, got)
			}
		})
	}
}

func TestEveryProofreadGivesItsReferenceBack(t *testing.T) {
	host := newLeaseHost(t)
	bridge, err := OpenProofreaderBridge(host)
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		_, _ = bridge.Proofread(t.Context(), domain.ProofreadRequest{
			Kind:   domain.ProofreadScan,
			Groups: []domain.ProofreadGroup{{Number: 1, Lines: []domain.ProofreadLine{{Number: 1, Text: "p"}}}},
		})
	}
	_ = bridge.Close()
	if host.acquired.Load() != 3 || host.released.Load() != 3 {
		t.Errorf("acquired %d, released %d, want 3 and 3", host.acquired.Load(), host.released.Load())
	}
}

func TestEveryRunOfAnAgentHoldsAndGivesBackItsOwnReference(t *testing.T) {
	host := newLeaseHost(t)
	bridge, err := OpenAgentBridge(t.Context(), host)
	if err != nil {
		t.Fatal(err)
	}

	const runs = 3
	for range runs {
		run, err := bridge.StartRun(t.Context(), port.Task{Question: "q"})
		if err != nil {
			t.Fatal(err)
		}
		for range run.GetSteps() {
		}
		_ = run.Stop()
		_ = run.Stop()
	}
	if got := host.acquired.Load(); got != 1+runs {
		t.Errorf("acquired %d times, want %d", got, 1+runs)
	}
	if got := host.released.Load(); got != runs {
		t.Errorf("gave back %d before Close, want %d", got, runs)
	}

	_ = bridge.Close()
	_ = bridge.Close()
	if got := host.released.Load(); got != 1+runs {
		t.Errorf("gave back %d after Close, want %d", got, 1+runs)
	}
}

type seekCloser struct {
	*strings.Reader
}

func (seekCloser) Close() error { return nil }

func TestEveryOpenDocumentHoldsAndGivesBackItsOwnReference(t *testing.T) {
	host := newLeaseHost(t)
	bridge, err := OpenDocumentBridge(t.Context(), host)
	if err != nil {
		t.Fatal(err)
	}

	const documents = 2
	for range documents {
		doc, err := bridge.OpenDocument(t.Context(), seekCloser{strings.NewReader("%PDF")})
		if err != nil {
			t.Fatal(err)
		}
		doc.Close()
		doc.Close()
	}
	_ = bridge.Close()
	_ = bridge.Close()

	if host.acquired.Load() != 1+documents || host.released.Load() != 1+documents {
		t.Errorf("acquired %d, released %d, want %d and %d",
			host.acquired.Load(), host.released.Load(), 1+documents, 1+documents)
	}
}
