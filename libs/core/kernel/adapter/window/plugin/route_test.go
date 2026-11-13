package plugin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/plugin"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

type processes struct {
	lists chan []plugindomain.ProcessStatus
}

func (p processes) Execute(ctx context.Context) <-chan []plugindomain.ProcessStatus {
	out := make(chan []plugindomain.ProcessStatus)
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
	lists := make(chan []plugindomain.ProcessStatus, 2)
	start := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	lists <- []plugindomain.ProcessStatus{{
		Process:    plugindomain.Process{PluginID: "org.example.maps", StartTime: start, LastUseTime: start, LeaseCount: 1},
		PluginName: "Maps", Health: plugindomain.HealthOK,
	}}
	path, served, _ := plugin.NewRoute(plugin.NewHandler(processes{lists: lists}), connect.WithInterceptors())
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
	if !stream.Receive() || len(stream.Msg().GetProcesses()) != 1 || stream.Msg().GetProcesses()[0].GetPluginName() != "Maps" {
		t.Fatalf("first list = %v, %v, want maps", stream.Msg(), stream.Err())
	}

	lists <- nil
	if !stream.Receive() || len(stream.Msg().GetProcesses()) != 0 {
		t.Fatalf("second list = %v, %v, want none", stream.Msg(), stream.Err())
	}
}
