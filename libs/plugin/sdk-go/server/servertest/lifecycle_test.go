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

	protocol, err := client.Ping(t.Context())
	if err != nil || protocol != server.ProtocolVersion {
		t.Fatalf("Ping = %d, %v", protocol, err)
	}
	err = client.Configure(t.Context(), map[string]string{"model": "small"})
	if err != nil || configured["model"] != "small" {
		t.Fatalf("Configure = %v, configured %v", err, configured)
	}
}

func TestLifecycleClient_ReturnsTheErrorsOfThePlugin(t *testing.T) {
	web := newLinkedPlugin(t, func(s *server.Server) error {
		s.SetConfigure(func(context.Context, map[string]string) error { return errors.New("configure failed") })
		return nil
	})
	client := NewLifecycleClient(web.Client(), web.URL, lakeToken)

	if err := client.Configure(t.Context(), map[string]string{"a": "b"}); err == nil {
		t.Fatal("Configure succeeded, want an error")
	}
	web.Close()
	if _, err := client.Ping(t.Context()); err == nil {
		t.Fatal("Ping of a stopped plugin succeeded")
	}
}
