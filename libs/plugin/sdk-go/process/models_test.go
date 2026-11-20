package process_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/process"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

type modelReports struct {
	numenv1connect.UnimplementedPluginServiceHandler
	updates  chan []string
	failNext atomic.Bool
}

func (m *modelReports) ReportModelProviders(_ context.Context, req *connect.Request[v1.ReportModelProvidersRequest]) (*connect.Response[v1.ReportModelProvidersResponse], error) {
	if m.failNext.Swap(false) {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("temporarily unavailable"))
	}
	rows := make([]string, 0, len(req.Msg.GetProviders()))
	for _, row := range req.Msg.GetProviders() {
		rows = append(rows, row.GetStage()+"="+row.GetProvider())
	}
	m.updates <- rows
	return connect.NewResponse(&v1.ReportModelProvidersResponse{}), nil
}

func openTestModels(t *testing.T, service *modelReports) (*process.ModelSessions, *server.Server) {
	t.Helper()
	_, handler := numenv1connect.NewPluginServiceHandler(service)
	host := httptest.NewServer(handler)
	t.Cleanup(host.Close)
	address, err := url.Parse(host.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(hostcall.HostPortEnv, address.Port())
	t.Setenv(hostcall.PluginTokenEnv, "current-run")
	plugin := server.New()
	t.Cleanup(func() {
		if err := plugin.Close(); err != nil {
			t.Error(err)
		}
	})
	return process.NewModelSessions(plugin, func(what string, err error) { t.Logf("%s: %v", what, err) }), plugin
}

func TestModelSessionsRetainEveryLiveProviderUntilItsLastSessionCloses(t *testing.T) {
	service := &modelReports{updates: make(chan []string, 4)}
	runtime, _ := openTestModels(t, service)
	check := func(want ...string) {
		t.Helper()
		select {
		case got := <-service.updates:
			if !slices.Equal(got, want) {
				t.Fatalf("providers = %v, want %v", got, want)
			}
		default:
			t.Fatal("no update")
		}
	}
	cpu := map[string]string{"Embeddings": "cpu"}
	closeFirst := runtime.Track(t.Context(), cpu)
	check("Embeddings=cpu")
	closeSecond := runtime.Track(t.Context(), cpu)
	closeGPU := runtime.Track(t.Context(), map[string]string{"Embeddings": "cuda"})
	check("Embeddings=cpu", "Embeddings=cuda")
	closeFirst()
	closeFirst()
	closeSecond()
	check("Embeddings=cuda")
	closeGPU()
	check()
}

func TestModelReportsRetryTheCurrentSnapshotAndStopOnShutdown(t *testing.T) {
	service := &modelReports{updates: make(chan []string, 4)}
	models, plugin := openTestModels(t, service)
	service.failNext.Store(true)
	closeCPU := models.Track(t.Context(), map[string]string{"Layout": "cpu"})
	select {
	case got := <-service.updates:
		if !slices.Equal(got, []string{"Layout=cpu"}) {
			t.Fatalf("retried snapshot = %v", got)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("opening was not retried")
	}
	service.failNext.Store(true)
	closeCPU()
	select {
	case got := <-service.updates:
		if len(got) != 0 {
			t.Fatalf("closed snapshot = %v", got)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("closing was not retried")
	}
	service.failNext.Store(true)
	models.Track(t.Context(), map[string]string{"Layout": "cuda"})
	if err := plugin.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-service.updates:
		t.Fatalf("received snapshot after shutdown: %v", got)
	case <-time.After(1500 * time.Millisecond):
	}
}
