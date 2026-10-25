package integration

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/agentbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/documentbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/proofreaderbridge"
	"github.com/jiva-studio/numen/libs/core/port/agentport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

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

type stubTranscriptionServer struct {
	numenv1connect.UnimplementedTranscriptionServiceHandler
}

func (stubTranscriptionServer) GetTranscriptionModel(
	context.Context, *connect.Request[v1.GetTranscriptionModelRequest],
) (*connect.Response[v1.GetTranscriptionModelResponse], error) {
	return connect.NewResponse(&v1.GetTranscriptionModelResponse{}), nil
}

func newLeaseHost(t *testing.T) *plugintest.LeaseHost {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(&plugintest.MockAgentServer{}))
	mux.Handle(numenv1connect.NewDocumentServiceHandler(stubDocumentServer{}))
	mux.Handle(numenv1connect.NewEmbeddingServiceHandler(stubEmbeddingServer{}))
	mux.Handle(numenv1connect.NewTranscriptionServiceHandler(stubTranscriptionServer{}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &plugintest.LeaseHost{Endpoint: pluginport.Endpoint{Host: "127.0.0.1", Port: server.Listener.Addr().(*net.TCPAddr).Port}}
}

func TestEveryProofreadingProducesItsReferenceBack(t *testing.T) {
	host := newLeaseHost(t)
	bridge, err := proofreaderbridge.Open(host)
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		_, _ = bridge.ProofreadSpeech(t.Context(), nil, 0)
	}
	_ = bridge.Close()
	if host.Acquired.Load() != 3 || host.Released.Load() != 3 {
		t.Errorf("acquired %d, released %d, want 3 and 3", host.Acquired.Load(), host.Released.Load())
	}
}

func TestEveryRunOfAnAgentContainsAndProducesBackItsOwnReference(t *testing.T) {
	host := newLeaseHost(t)
	bridge, err := agentbridge.OpenForPlugin(t.Context(), host, "md.numen.stub")
	if err != nil {
		t.Fatal(err)
	}

	const runs = 3
	for range runs {
		run, err := bridge.StartRun(t.Context(), agentport.Task{Question: "q"})
		if err != nil {
			t.Fatal(err)
		}
		for range run.GetSteps() {
		}
		_ = run.Stop()
		_ = run.Stop()
	}
	if got := host.Acquired.Load(); got != 1+runs {
		t.Errorf("acquired %d times, want %d", got, 1+runs)
	}
	if got := host.Released.Load(); got != runs {
		t.Errorf("gave back %d before Close, want %d", got, runs)
	}

	_ = bridge.Close()
	_ = bridge.Close()
	if got := host.Released.Load(); got != 1+runs {
		t.Errorf("gave back %d after Close, want %d", got, 1+runs)
	}
}

type seekCloser struct {
	*strings.Reader
}

func (seekCloser) Close() error { return nil }

func TestEveryOpenDocumentContainsAndProducesBackItsOwnReference(t *testing.T) {
	host := newLeaseHost(t)
	bridge, err := documentbridge.Open(t.Context(), host)
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

	if host.Acquired.Load() != 1+documents || host.Released.Load() != 1+documents {
		t.Errorf("acquired %d, released %d, want %d and %d",
			host.Acquired.Load(), host.Released.Load(), 1+documents, 1+documents)
	}
}
