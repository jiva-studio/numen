package host_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

type dependencyHost struct {
	numenv1connect.UnimplementedHostDependencyServiceHandler
}

func (dependencyHost) FetchModelFile(_ context.Context, req *connect.Request[v1.FetchModelFileRequest]) (*connect.Response[v1.FetchModelFileResponse], error) {
	if req.Header().Get("Authorization") != "Bearer test" || req.Msg.GetUrl() != "https://models.invalid/graph" {
		return connect.NewResponse(&v1.FetchModelFileResponse{Error: v1.ErrorCode_ERROR_CODE_INTERNAL, Detail: "unexpected request"}), nil
	}
	if req.Msg.GetSha256() != "pin" {
		return connect.NewResponse(&v1.FetchModelFileResponse{Error: v1.ErrorCode_ERROR_CODE_INTERNAL, Detail: "checksum mismatch"}), nil
	}
	return connect.NewResponse(&v1.FetchModelFileResponse{Path: "local-model"}), nil
}

func TestDependencyClientReturnsModelPathsAndFetchErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostDependencyServiceHandler(dependencyHost{}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := host.NewDependencyClient(hostcall.Endpoint{BaseURL: server.URL, Token: "test"})
	const address = "https://models.invalid/graph"
	if file, err := client.FetchModelFile(t.Context(), address, "pin"); err != nil || file != "local-model" {
		t.Fatalf("file = %q, %v", file, err)
	}
	if file, err := client.FetchModelFile(t.Context(), address, "bad"); err == nil || err.Error() != "checksum mismatch" || file != "" {
		t.Fatalf("bad pin = %q, %v", file, err)
	}
}
