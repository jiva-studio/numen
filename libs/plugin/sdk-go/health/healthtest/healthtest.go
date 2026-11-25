// Package healthtest is a host that keeps the health items a plugin reports, for the tests of a plugin.
// It keeps one list per test and checks the key of every item as the real host does
// (libs/core/kernel/domain/plugin/health.go, CheckHealthKey) and the limits of the text and the detail, and nothing else.
package healthtest

import (
	"context"
	"fmt"
	"net/http/httptest"
	"net/url"
	"regexp"
	"slices"
	"sync"
	"testing"
	"unicode/utf8"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// Host serves the health service of a host on a loopback port for as long as the test runs.
type Host struct {
	port    string
	service *service
}

// service serves the health service and keeps the items it receives.
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

// BuildVars returns the variables a host gives a plugin it links in, naming this host and the token.
func (h *Host) BuildVars(token string) map[string]string {
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
	if err := checkKey(in.GetKey()); err != nil {
		return nil, err
	}
	if utf8.RuneCountInString(in.GetText()) > health.MaxTextLength || utf8.RuneCountInString(in.GetDetail()) > health.MaxDetailLength {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf(
			"invalid health item: the text of the item %q is longer than %d characters or its detail longer than %d",
			in.GetKey(), health.MaxTextLength, health.MaxDetailLength))
	}
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
	key := req.Msg.GetKey()
	if err := checkKey(key); err != nil {
		return nil, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
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

const maxKeyLength = 32

var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func checkKey(key string) error {
	if len(key) > maxKeyLength || !keyPattern.MatchString(key) {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid health item: the key %q is not accepted by the host", key))
	}
	return nil
}
