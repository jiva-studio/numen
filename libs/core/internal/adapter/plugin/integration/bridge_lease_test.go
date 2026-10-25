package integration

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/agentbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
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

func newLeaseHost(t *testing.T) *plugintest.LeaseHost {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(&plugintest.MockAgentServer{}))
	mux.Handle(numenv1connect.NewDocumentServiceHandler(stubDocumentServer{}))
	mux.Handle(numenv1connect.NewEmbeddingServiceHandler(stubEmbeddingServer{}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &plugintest.LeaseHost{Endpoint: pluginport.Endpoint{Host: "127.0.0.1", Port: server.Listener.Addr().(*net.TCPAddr).Port}}
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
