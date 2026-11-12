package servertest

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

const lakeToken = "lake-token"

func newLinkedPlugin(t *testing.T, mount func(*server.Server) error) *httptest.Server {
	t.Helper()
	handler, err := server.NewLinkedHandler(map[string]string{hostcall.PluginTokenEnv: lakeToken}, mount)
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	t.Cleanup(web.Close)
	return web
}

func TestLifecycleClient_CallsThePluginOverTheWire(t *testing.T) {
	var configured map[string]string
	closed := false
	web := newLinkedPlugin(t, func(s *server.Server) error {
		s.SetUpdateConfig(func(values map[string]string) { configured = values })
		s.OnClose(func() error { closed = true; return nil })
		return nil
	})
	client := NewLifecycleClient(web.Client(), web.URL, hostcall.WithBearer(lakeToken))

	ping, err := client.Ping(t.Context())
	if err != nil || ping.Status != "ok" || ping.Protocol != server.ProtocolVersion {
		t.Fatalf("Ping = %+v, %v", ping, err)
	}
	accepted, err := client.Configure(t.Context(), map[string]string{"model": "small"})
	if err != nil || !accepted || configured["model"] != "small" {
		t.Fatalf("Configure = %v, %v, configured %v", accepted, err, configured)
	}
	accepted, err = client.Shutdown(t.Context())
	if err != nil || !accepted || !closed {
		t.Fatalf("Shutdown = %v, %v, closed %v", accepted, err, closed)
	}
}

func TestLifecycleClient_ReturnsTheErrorsOfThePlugin(t *testing.T) {
	web := newLinkedPlugin(t, func(s *server.Server) error {
		s.SetConfigure(func(context.Context, map[string]string) error { return errors.New("configure failed") })
		s.OnClose(func() error { return errors.New("shutdown failed") })
		return nil
	})
	client := NewLifecycleClient(web.Client(), web.URL, hostcall.WithBearer(lakeToken))

	if ok, err := client.Configure(t.Context(), map[string]string{"a": "b"}); err == nil || ok {
		t.Fatalf("Configure = %v, %v, want an error", ok, err)
	}
	if ok, err := client.Shutdown(t.Context()); err == nil || ok || connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("Shutdown = %v, %v, want an internal error", ok, err)
	}
	web.Close()
	if _, err := client.Ping(t.Context()); err == nil {
		t.Fatal("Ping of a stopped plugin succeeded")
	}
}
