package hostendpoint

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

type releasedProcesses struct {
	mu        sync.Mutex
	pluginIDs []string
}

func (r *releasedProcesses) ReleaseProcess(_ context.Context, pluginID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pluginIDs = append(r.pluginIDs, pluginID)
}

func (r *releasedProcesses) list() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.pluginIDs)
}

func newProcessClient(t *testing.T, released *releasedProcesses, token string) numenv1connect.PluginServiceClient {
	t.Helper()
	callers := newCallers().asCallers()
	callers.Processes = released
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginServiceHandler(&processEndpoint{endpoint: &Endpoint{callers: callers}}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewPluginServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.AuthInterceptor(token)))
}

// A plugin releases its own process: the token names the plugin.
func TestAPluginReleasesItsOwnProcess(t *testing.T) {
	released := &releasedProcesses{}
	client := newProcessClient(t, released, "token-a")

	if _, err := client.ReleaseProcess(t.Context(), connect.NewRequest(&v1.ReleaseProcessRequest{})); err != nil {
		t.Fatal(err)
	}

	if got := released.list(); !slices.Equal(got, []string{"md.numen.a"}) {
		t.Errorf("released %v, want md.numen.a", got)
	}
}

func TestAnUnknownTokenReleasesNothing(t *testing.T) {
	released := &releasedProcesses{}
	client := newProcessClient(t, released, "token-of-nobody")

	_, err := client.ReleaseProcess(t.Context(), connect.NewRequest(&v1.ReleaseProcessRequest{}))

	if connect.CodeOf(err) != connect.CodeUnauthenticated || len(released.list()) != 0 {
		t.Errorf("release = %v with %v released, want it unauthenticated", err, released.list())
	}
}
