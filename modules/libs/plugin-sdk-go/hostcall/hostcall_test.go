package hostcall_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/hostcall"
)

type fakeStreamConn struct {
	connect.StreamingClientConn
	header http.Header
}

func (f *fakeStreamConn) RequestHeader() http.Header { return f.header }

func TestTheTokenIsPresentedOnAStreamingCall(t *testing.T) {
	conn := &fakeStreamConn{header: http.Header{}}
	wrapped := hostcall.Bearer("s").WrapStreamingClient(func(context.Context, connect.Spec) connect.StreamingClientConn { return conn })

	wrapped(context.Background(), connect.Spec{})

	if got := conn.header.Get("Authorization"); got != "Bearer s" {
		t.Errorf("got %q", got)
	}
}

func TestNoTokenIsPresentedAsNone(t *testing.T) {
	conn := &fakeStreamConn{header: http.Header{}}

	hostcall.Bearer("").WrapStreamingClient(func(context.Context, connect.Spec) connect.StreamingClientConn { return conn })(context.Background(), connect.Spec{})

	if got := conn.header.Get("Authorization"); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestTheTokenIsPresentedOnAUnaryCall(t *testing.T) {
	var observed string
	next := func(_ context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		observed = req.Header().Get("Authorization")
		return nil, nil
	}

	_, _ = hostcall.Bearer("u").WrapUnary(next)(context.Background(), connect.NewRequest(&struct{}{}))

	if observed != "Bearer u" {
		t.Errorf("got %q", observed)
	}
}

func TestAStreamingHandlerIsPassedThrough(t *testing.T) {
	called := false
	next := func(context.Context, connect.StreamingHandlerConn) error { called = true; return nil }

	_ = hostcall.Bearer("s").WrapStreamingHandler(next)(context.Background(), nil)

	if !called {
		t.Error("handler not called")
	}
}

func TestTheHostIsWhereTheVariablesSayAndTheTokenIsTheirs(t *testing.T) {
	endpoint, err := hostcall.FromVars(map[string]string{hostcall.HostPortEnv: " 4321 ", hostcall.PluginTokenEnv: "tok"})

	if err != nil || endpoint.BaseURL != "http://127.0.0.1:4321" || endpoint.Token != "tok" {
		t.Errorf("got %+v (%v)", endpoint, err)
	}
}

func TestNoPortIsNoHost(t *testing.T) {
	if _, err := hostcall.FromVars(map[string]string{}); !errors.Is(err, hostcall.ErrNoHost) {
		t.Errorf("got %v", err)
	}
	t.Setenv(hostcall.HostPortEnv, "")
	if _, err := hostcall.FromEnv(); !errors.Is(err, hostcall.ErrNoHost) {
		t.Errorf("got %v", err)
	}
}

func TestTheEnvironmentNamesTheHost(t *testing.T) {
	t.Setenv(hostcall.HostPortEnv, "99")
	t.Setenv(hostcall.PluginTokenEnv, "t")

	endpoint, err := hostcall.FromEnv()

	if err != nil || endpoint.BaseURL != "http://127.0.0.1:99" || endpoint.Token != "t" {
		t.Errorf("got %+v (%v)", endpoint, err)
	}
	if len(endpoint.Options()) != 1 || hostcall.HTTPClient() != http.DefaultClient {
		t.Error("the client is not made as the plugin makes it")
	}
}
