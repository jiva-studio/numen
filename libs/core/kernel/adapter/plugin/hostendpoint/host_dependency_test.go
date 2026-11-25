package hostendpoint

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/provision"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugindependency"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

func TestDependencyClientRequiresAnAuthenticatedRequest(t *testing.T) {
	registry := provision.NewObservedRegistry(nil, provision.NewPlaceholderProvisioner(t.TempDir()))
	callers := newCallers().asCallers()
	callers.Dependencies = plugindependency.NewFetch(func() pluginport.DependencyRegistry { return registry })
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostDependencyServiceHandler(&dependencyEndpoint{endpoint: &Endpoint{callers: callers}}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	const address = "https://models.invalid/selected.onnx"
	client := numenv1connect.NewHostDependencyServiceClient(server.Client(), server.URL)
	call := func(token, sum string) (*connect.Response[v1.FetchModelFileResponse], error) {
		req := connect.NewRequest(&v1.FetchModelFileRequest{Url: address, Sha256: sum})
		req.Header().Set("Authorization", "Bearer "+token)
		return client.FetchModelFile(t.Context(), req)
	}
	if _, err := call("unrecognized", ""); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("unauthenticated request = %v", err)
	}
	invalid, err := call("token-a", "bogus")
	if err != nil || invalid.Msg.GetError() == v1.ErrorCode_ERROR_CODE_UNSPECIFIED || invalid.Msg.GetDetail() == "" || invalid.Msg.GetPath() != "" {
		t.Fatalf("invalid pin response = %v, %v", invalid, err)
	}
	answer, err := call("token-a", strings.Repeat("0", 64))
	if err != nil {
		t.Fatal(err)
	}
	file := answer.Msg.GetPath()
	if _, err := os.Stat(file); err != nil {
		t.Fatal(err)
	}
}
