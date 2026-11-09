package plugin

import (
	"context"
	"errors"
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
	mu            sync.Mutex
	calls         []*v1.ConfigureRequest
	unimplemented bool
	shutdowns     int
	failure       error
	hold          chan struct{}
	entered       chan struct{}
	delay         time.Duration
	during        func()
}

func (c *configured) setFailure(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failure = err
}

func (c *configured) setHold(t *testing.T) (entered <-chan struct{}, release func()) {
	t.Helper()
	hold, arrivals := make(chan struct{}), make(chan struct{}, 8)
	c.mu.Lock()
	c.hold, c.entered = hold, arrivals
	c.mu.Unlock()
	var once sync.Once
	release = func() { once.Do(func() { close(hold) }) }
	t.Cleanup(release)
	return arrivals, release
}

func (c *configured) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.calls)
}

func (c *configured) Configure(
	ctx context.Context, req *connect.Request[v1.ConfigureRequest],
) (*connect.Response[v1.ConfigureResponse], error) {
	c.mu.Lock()
	hold, entered, delay, during := c.hold, c.entered, c.delay, c.during
	c.mu.Unlock()
	if during != nil {
		during()
	}
	if entered != nil {
		select {
		case entered <- struct{}{}:
		default:
		}
	}
	if hold != nil {
		select {
		case <-hold:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, req.Msg)
	if c.failure != nil {
		return nil, c.failure
	}
	if c.unimplemented {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("no configure"))
	}
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
