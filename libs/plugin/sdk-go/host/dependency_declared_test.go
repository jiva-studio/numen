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

type declaredDependencyHost struct {
	numenv1connect.UnimplementedHostDependencyServiceHandler
}

func (declaredDependencyHost) FetchDependency(_ context.Context, req *connect.Request[v1.FetchDependencyRequest]) (*connect.Response[v1.FetchDependencyResponse], error) {
	if req.Header().Get("Authorization") != "Bearer test" {
		return connect.NewResponse(&v1.FetchDependencyResponse{Error: v1.ErrorCode_ERROR_CODE_INTERNAL, Detail: "unexpected request"}), nil
	}
	if req.Msg.GetId() != "speech-encoder" {
		return connect.NewResponse(&v1.FetchDependencyResponse{Error: v1.ErrorCode_ERROR_CODE_NOT_FOUND, Detail: "the plugin declares no model " + req.Msg.GetId()}), nil
	}
	return connect.NewResponse(&v1.FetchDependencyResponse{Path: "/models/speech-encoder.onnx"}), nil
}

func TestDependencyClientFetchesADeclaredModelByItsID(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostDependencyServiceHandler(declaredDependencyHost{}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := host.NewDependencyClient(hostcall.Endpoint{BaseURL: server.URL, Token: "test"})

	if file, err := client.FetchDependency(t.Context(), "speech-encoder"); err != nil || file != "/models/speech-encoder.onnx" {
		t.Errorf("speech-encoder = %q, %v", file, err)
	}
	file, err := client.FetchDependency(t.Context(), "absent-model")
	if err == nil || err.Error() != "the plugin declares no model absent-model" || file != "" {
		t.Errorf("absent-model = %q, %v, want the detail of the host", file, err)
	}
}
