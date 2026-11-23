// Package processtest is a host that counts the releases of a plugin's process, for the tests of a plugin.
package processtest

import (
	"context"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/process"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

const pluginToken = "plugin-token"

// Host serves the release of a process on a loopback port for as long as the test runs.
type Host struct {
	port    string
	service *service
}

type service struct {
	numenv1connect.UnimplementedHostProcessServiceHandler

	mu       sync.Mutex
	released int
}

func (s *service) ReleaseProcess(
	context.Context, *connect.Request[v1.ReleaseProcessRequest],
) (*connect.Response[v1.ReleaseProcessResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.released++
	return connect.NewResponse(&v1.ReleaseProcessResponse{}), nil
}

// NewHost starts the host and stops it when the test ends.
func NewHost(t *testing.T) *Host {
	t.Helper()
	host := &Host{service: &service{}}
	_, handler := numenv1connect.NewHostProcessServiceHandler(host.service)
	served := httptest.NewServer(server.RequireToken(pluginToken, handler))
	t.Cleanup(served.Close)
	address, err := url.Parse(served.URL)
	if err != nil {
		t.Fatal(err)
	}
	host.port = address.Port()
	return host
}

// GetLaunchVars returns the host port and token to pass to a linked plugin.
func (h *Host) GetLaunchVars() map[string]string {
	return map[string]string{hostcall.HostPortEnv: h.port, hostcall.PluginTokenEnv: pluginToken}
}

// NewClient returns a client of the plugin that releases its process on this host.
func (h *Host) NewClient(t *testing.T) *process.Client {
	t.Helper()
	endpoint, err := hostcall.ReadVars(h.GetLaunchVars())
	if err != nil {
		t.Fatal(err)
	}
	return process.NewClient(endpoint)
}

// CountReleases returns how many times the plugin released its process.
func (h *Host) CountReleases() int {
	h.service.mu.Lock()
	defer h.service.mu.Unlock()
	return h.service.released
}
