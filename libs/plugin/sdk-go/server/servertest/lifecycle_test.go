package servertest

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

const lakeToken = "lake-token"

func newLinkedPlugin(t *testing.T, mount func(*server.Server) error) *httptest.Server {
	t.Helper()
	handler, err := NewHandler(map[string]string{hostcall.PluginTokenEnv: lakeToken}, mount)
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	t.Cleanup(web.Close)
	return web
}

func TestLifecycleClient_CallsThePluginOverTheWire(t *testing.T) {
	var configured map[string]string
	web := newLinkedPlugin(t, func(s *server.Server) error {
		s.SetUpdateConfig(func(values map[string]string) { configured = values })
		return nil
	})
	client := NewLifecycleClient(web.Client(), web.URL, lakeToken)

	status, protocol, err := client.Ping(t.Context())
	if err != nil || status != "ok" || protocol != server.ProtocolVersion {
		t.Fatalf("Ping = %q, %d, %v", status, protocol, err)
	}
	accepted, err := client.Configure(t.Context(), map[string]string{"model": "small"})
	if err != nil || !accepted || configured["model"] != "small" {
		t.Fatalf("Configure = %v, %v, configured %v", accepted, err, configured)
	}
}

func TestLifecycleClient_ReturnsTheErrorsOfThePlugin(t *testing.T) {
	web := newLinkedPlugin(t, func(s *server.Server) error {
		s.SetConfigure(func(context.Context, map[string]string) error { return errors.New("configure failed") })
		return nil
	})
	client := NewLifecycleClient(web.Client(), web.URL, lakeToken)

	if ok, err := client.Configure(t.Context(), map[string]string{"a": "b"}); err == nil || ok {
		t.Fatalf("Configure = %v, %v, want an error", ok, err)
	}
	web.Close()
	if _, _, err := client.Ping(t.Context()); err == nil {
		t.Fatal("Ping of a stopped plugin succeeded")
	}
}
