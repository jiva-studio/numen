package plugin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/plugintest"
)

type stubEmbeddingServer struct {
	numenv1connect.UnimplementedSearchSourceServiceHandler
}

func (stubEmbeddingServer) DescribeSearchSource(
	context.Context, *connect.Request[v1.DescribeSearchSourceRequest],
) (*connect.Response[v1.DescribeSearchSourceResponse], error) {
	return connect.NewResponse(&v1.DescribeSearchSourceResponse{}), nil
}

func newLeaseHost(t *testing.T) *plugintest.LeaseHost {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewSearchSourceServiceHandler(stubEmbeddingServer{}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &plugintest.LeaseHost{Endpoint: transport.Endpoint{Host: "127.0.0.1", Port: wiretest.GetPort(t, server)}}
}
