package plugin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/plugin"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

type processes struct {
	lists chan plugindomain.ProcessList
}

func (p processes) Execute(ctx context.Context) <-chan plugindomain.ProcessList {
	out := make(chan plugindomain.ProcessList)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case list := <-p.lists:
				select {
				case out <- list:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}

func TestAWindowGetsThePluginsThatRunAndEachChange(t *testing.T) {
	lists := make(chan plugindomain.ProcessList, 2)
	start := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	lists <- plugindomain.ProcessList{Time: start.Add(time.Minute), Processes: []plugindomain.ProcessStatus{{
		Process:    plugindomain.Process{PluginID: "org.example.maps", StartTime: start, LastUseTime: start, LeaseCount: 1},
		PluginName: "Maps", Health: plugindomain.HealthOK,
	}}}
	path, served, _ := plugin.NewRoute(plugin.NewHandler(processes{lists: lists}, &releases{}), connect.WithInterceptors())
	mux := http.NewServeMux()
	mux.Handle(path, served)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := numenv1connect.NewPluginServiceClient(server.Client(), server.URL)

	stream, err := client.WatchPluginProcesses(t.Context(), connect.NewRequest(&v1.WatchPluginProcessesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := stream.Close(); err != nil {
			t.Error(err)
		}
	})
	if !stream.Receive() || len(stream.Msg().GetProcesses()) != 1 || stream.Msg().GetProcesses()[0].GetPluginName() != "Maps" ||
		stream.Msg().GetTimeUnixMs() != start.Add(time.Minute).UnixMilli() {
		t.Fatalf("first list = %v, %v, want maps", stream.Msg(), stream.Err())
	}

	lists <- plugindomain.ProcessList{Time: start.Add(2 * time.Minute)}
	if !stream.Receive() || len(stream.Msg().GetProcesses()) != 0 {
		t.Fatalf("second list = %v, %v, want none", stream.Msg(), stream.Err())
	}
}

type releases struct {
	mu        sync.Mutex
	pluginIDs []string
}

func (r *releases) Execute(_ context.Context, pluginID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pluginIDs = append(r.pluginIDs, pluginID)
}

func (r *releases) list() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.pluginIDs)
}

func newReleaseClient(t *testing.T, released *releases) numenv1connect.PluginServiceClient {
	t.Helper()
	path, served, _ := plugin.NewRoute(plugin.NewHandler(processes{lists: nil}, released), connect.WithInterceptors())
	mux := http.NewServeMux()
	mux.Handle(path, served)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewPluginServiceClient(server.Client(), server.URL)
}

func TestAWindowReleasesTheProcessOfThePluginItNames(t *testing.T) {
	released := &releases{}
	client := newReleaseClient(t, released)

	if _, err := client.ReleaseProcess(t.Context(), connect.NewRequest(&v1.ReleaseProcessRequest{PluginId: "org.example.maps"})); err != nil {
		t.Fatal(err)
	}

	if got := released.list(); !slices.Equal(got, []string{"org.example.maps"}) {
		t.Errorf("released %v, want org.example.maps", got)
	}
}

func TestAReleaseThatNamesNoPluginIsRefused(t *testing.T) {
	released := &releases{}
	client := newReleaseClient(t, released)

	_, err := client.ReleaseProcess(t.Context(), connect.NewRequest(&v1.ReleaseProcessRequest{}))

	if connect.CodeOf(err) != connect.CodeInvalidArgument || len(released.list()) != 0 {
		t.Errorf("error = %v, released %v; want InvalidArgument and none", err, released.list())
	}
}
