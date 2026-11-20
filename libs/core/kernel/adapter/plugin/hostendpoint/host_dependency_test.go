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
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

func TestDependencyClientRequiresAnAuthenticatedRequest(t *testing.T) {
	registry := provision.NewRegistry(provision.NewPlaceholderProvisioner(t.TempDir()))
	callers := newCallers().asCallers()
	callers.Dependencies = plugindependency.NewFetch(func() pluginport.DependencyRegistry { return registry })
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostDependencyServiceHandler(&dependencyEndpoint{endpoint: &Endpoint{callers: callers}}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	const address = "https://models.invalid/selected.onnx"
	client := host.NewDependencyClient(hostcall.Endpoint{BaseURL: server.URL, Token: "unrecognized"})
	if _, err := client.FetchModelFile(t.Context(), address, ""); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("unauthenticated request = %v", err)
	}
	client = host.NewDependencyClient(hostcall.Endpoint{BaseURL: server.URL, Token: "token-a"})
	if _, err := client.FetchModelFile(t.Context(), address, "bogus"); err == nil {
		t.Fatal("invalid pin succeeded")
	}
	file, err := client.FetchModelFile(t.Context(), address, strings.Repeat("0", 64))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatal(err)
	}
}
