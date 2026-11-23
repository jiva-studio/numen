// Package logstest is a host that keeps the log entries a plugin reports, for the tests of a plugin.
package logstest

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/logs"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// Host serves LogService on a loopback port for as long as the test runs.
type Host struct {
	port    string
	service *service
}

type service struct {
	numenv1connect.UnimplementedLogServiceHandler

	mu      sync.Mutex
	entries []logs.Entry
}

// NewHost starts the host and stops it when the test ends.
func NewHost(t *testing.T) *Host {
	t.Helper()
	host := &Host{service: &service{}}
	_, handler := numenv1connect.NewLogServiceHandler(host.service)
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

// List returns the entries reported so far in arrival order.
func (h *Host) List() []logs.Entry {
	return h.service.list()
}

func (s *service) list() []logs.Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.entries)
}

// WriteLog appends the entry to the host's in-memory list.
func (s *service) WriteLog(
	_ context.Context, req *connect.Request[v1.WriteLogRequest],
) (*connect.Response[v1.WriteLogResponse], error) {
	var props map[string]any
	if len(req.Msg.GetPropertiesJson()) > 0 {
		props = make(map[string]any, len(req.Msg.GetPropertiesJson()))
		for k, raw := range req.Msg.GetPropertiesJson() {
			var decoded any
			if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
				props[k] = decoded
			} else {
				props[k] = raw
			}
		}
	}
	entry := logs.Entry{
		Level:      readLevel(req.Msg.GetLevel()),
		Message:    req.Msg.GetMessage(),
		Properties: props,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, entry)
	return connect.NewResponse(&v1.WriteLogResponse{}), nil
}

func readLevel(level v1.LogLevel) logs.Level {
	switch level {
	case v1.LogLevel_LOG_LEVEL_DEBUG:
		return logs.Debug
	case v1.LogLevel_LOG_LEVEL_INFO, v1.LogLevel_LOG_LEVEL_UNSPECIFIED:
		return logs.Info
	case v1.LogLevel_LOG_LEVEL_WARNING:
		return logs.Warning
	case v1.LogLevel_LOG_LEVEL_ERROR:
		return logs.Error
	}
	return logs.Info
}
