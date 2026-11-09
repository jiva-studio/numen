package plugin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

const configuredPluginID = "md.numen.agents.numen"

// configured records the Configure calls a plugin receives.
type configured struct {
	testsupport.PluginLifecycle
	mu        sync.Mutex
	calls     []*v1.ConfigureRequest
	shutdowns int
}

func (c *configured) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.calls)
}

func (c *configured) Configure(
	_ context.Context, req *connect.Request[v1.ConfigureRequest],
) (*connect.Response[v1.ConfigureResponse], error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, req.Msg)
	return connect.NewResponse(&v1.ConfigureResponse{Success: true}), nil
}

// Shutdown counts the shutdown requests the plugin receives.
func (c *configured) Shutdown(
	_ context.Context, _ *connect.Request[v1.ShutdownRequest],
) (*connect.Response[v1.ShutdownResponse], error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.shutdowns++
	return connect.NewResponse(&v1.ShutdownResponse{Success: true}), nil
}

func (c *configured) last(t *testing.T) *v1.ConfigureRequest {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.calls) == 0 {
		t.Fatal("the plugin was not configured")
	}
	return c.calls[len(c.calls)-1]
}

func newConfiguredHost(t *testing.T) (*Host, *configured) {
	t.Helper()
	probe := &configured{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(probe))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	host := NewHost(testsupport.GetMachine(), "desktop.editor")
	t.Cleanup(func() {
		if err := host.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})
	host.plugins[configuredPluginID] = manifest.Discovered{Manifest: pluginport.PluginManifest{
		ID: configuredPluginID,
		Settings: []manifest.Setting{
			{Key: "model", Type: "string", Title: "Model", Default: "sonnet"},
		},
	}}
	sup := lifecycle.New(testsupport.GetMachine())
	sup.ActivateInProcess(testsupport.GetMachine().Now())
	host.supervisors[configuredPluginID] = sup
	host.endpoints[configuredPluginID] = transport.Endpoint{
		Host: "127.0.0.1", Port: plugintest.GetPort(t, server), Token: "launch-name",
	}
	return host, probe
}
