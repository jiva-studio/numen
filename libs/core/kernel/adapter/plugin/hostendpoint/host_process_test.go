package hostendpoint

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"

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

func TestInvalidProviderReportsPreserveTheRunningStages(t *testing.T) {
	callers := newCallers().asCallers()
	var stages []plugin.ModelProvider
	callers.ModelProviders = func(id, token string, _ uint64, rows []plugin.ModelProvider) bool {
		if id != "md.numen.a" || token != "token-a" {
			return false
		}
		stages = slices.Clone(rows)
		return true
	}
	endpoint := processEndpoint{endpoint: &Endpoint{callers: callers}}
	report := func(rows ...*v1.ModelProvider) error {
		req := connect.NewRequest(&v1.ReportModelProvidersRequest{Providers: rows, Revision: 1})
		req.Header().Set("Authorization", "Bearer token-a")
		_, err := endpoint.ReportModelProviders(t.Context(), req)
		return err
	}
	row := &v1.ModelProvider{Stage: "Layout", Provider: "cpu"}
	if err := report(row); err != nil {
		t.Fatal(err)
	}
	if err := report(row, row); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("duplicate model stage: %v", err)
	}
	if want := []plugin.ModelProvider{{Stage: "Layout", Provider: "cpu"}}; !slices.Equal(stages, want) {
		t.Fatalf("stages = %v", stages)
	}
	if err := report(); err != nil || len(stages) != 0 {
		t.Fatalf("close: %v, stages %v", err, stages)
	}
}
