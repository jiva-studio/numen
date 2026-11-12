// Package healthtest is a host that keeps the health items a plugin reports, for the tests of a plugin.
// It keeps one list per test and checks nothing the real host checks.
package healthtest

import (
	"context"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// Host serves the health service of a host on a loopback port for as long as the test runs.
type Host struct {
	port    string
	service *service
}

// service serves the health service and keeps the items it is told.
type service struct {
	numenv1connect.UnimplementedPluginHealthServiceHandler

	mu    sync.Mutex
	items map[string]health.Item
	order []string
}

// NewHost starts the host and stops it when the test ends.
func NewHost(t *testing.T) *Host {
	t.Helper()
	host := &Host{service: &service{items: map[string]health.Item{}}}
	_, handler := numenv1connect.NewPluginHealthServiceHandler(host.service)
	served := httptest.NewServer(handler)
	t.Cleanup(served.Close)
	address, err := url.Parse(served.URL)
	if err != nil {
		t.Fatal(err)
	}
	host.port = address.Port()
	return host
}

// Vars are the variables a host gives a plugin it links in, naming this host and the token.
func (h *Host) Vars(token string) map[string]string {
	return map[string]string{hostcall.HostPortEnv: h.port, hostcall.PluginTokenEnv: token}
}

// List returns the items reported and not cleared, in the order their keys were first reported.
func (h *Host) List() []health.Item {
	return h.service.list()
}

func (h *service) list() []health.Item {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]health.Item, 0, len(h.order))
	for _, key := range h.order {
		out = append(out, h.items[key])
	}
	return out
}

// ReportHealth keeps the item under its key.
func (h *service) ReportHealth(
	_ context.Context, req *connect.Request[v1.ReportHealthRequest],
) (*connect.Response[v1.ReportHealthResponse], error) {
	in := req.Msg.GetItem()
	item := health.Item{Key: in.GetKey(), Severity: readSeverity(in.GetSeverity()), Text: in.GetText(), Detail: in.GetDetail()}
	if action := in.GetAction(); action != nil {
		item.Action = &health.Action{Command: action.GetCommand(), Values: action.GetValues(), Title: action.GetTitle()}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, isKept := h.items[item.Key]; !isKept {
		h.order = append(h.order, item.Key)
	}
	h.items[item.Key] = item
	return connect.NewResponse(&v1.ReportHealthResponse{}), nil
}

// ClearHealth removes the item under the key.
func (h *service) ClearHealth(
	_ context.Context, req *connect.Request[v1.ClearHealthRequest],
) (*connect.Response[v1.ClearHealthResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	key := req.Msg.GetKey()
	delete(h.items, key)
	h.order = slices.DeleteFunc(h.order, func(one string) bool { return one == key })
	return connect.NewResponse(&v1.ClearHealthResponse{}), nil
}

func readSeverity(severity v1.HealthSeverity) health.Severity {
	switch severity {
	case v1.HealthSeverity_HEALTH_SEVERITY_WARNING:
		return health.Warning
	case v1.HealthSeverity_HEALTH_SEVERITY_ERROR:
		return health.Error
	case v1.HealthSeverity_HEALTH_SEVERITY_OK, v1.HealthSeverity_HEALTH_SEVERITY_UNSPECIFIED:
		return 0
	}
	return 0
}
