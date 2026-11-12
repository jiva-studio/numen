package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

func TestPingResult_RoundTripsThroughTheWire(t *testing.T) {
	if ProtocolVersion != 1 {
		t.Fatalf("ProtocolVersion = %d, want 1", ProtocolVersion)
	}
	want := PingResult{Status: "degraded", Protocol: 7}

	got := readPing(writePing(want))

	if got != want {
		t.Errorf("ping result %+v, want %+v", got, want)
	}
}

func TestLifecycleClient_CallsTheHandlerOverTheWire(t *testing.T) {
	var configured map[string]string
	shutdown := make(chan struct{})
	life := NewLifecycleServer(shutdown, WithLifecycleUpdateConfig(func(values map[string]string) { configured = values }))
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(lifecycleHandler{state: life}))
	web := httptest.NewServer(mux)
	defer web.Close()
	client := NewLifecycleClient(web.Client(), web.URL)

	ping, err := client.Ping(t.Context())
	if err != nil || ping.Status != "ok" || ping.Protocol != 1 {
		t.Fatalf("Ping = %+v, %v", ping, err)
	}
	accepted, err := client.Configure(t.Context(), map[string]string{"model": "small"})
	if err != nil || !accepted || configured["model"] != "small" {
		t.Fatalf("Configure = %v, %v, configured %v", accepted, err, configured)
	}
	accepted, err = client.Shutdown(t.Context())
	if err != nil || !accepted {
		t.Fatalf("Shutdown = %v, %v", accepted, err)
	}
	select {
	case <-life.ShutdownTrigger():
	default:
		t.Fatal("the shutdown signal was not sent")
	}
}

func TestLifecycleClient_PropagatesErrors(t *testing.T) {
	wantConfErr := errors.New("configure failed")
	life := NewLifecycleServer(
		make(chan struct{}),
		WithLifecycleConfigure(func(context.Context, map[string]string) error {
			return wantConfErr
		}),
		WithLifecycleShutdown(func() error {
			return errors.New("shutdown failed")
		}),
	)
	lh := lifecycleHandler{state: life}
	if _, err := lh.Configure(t.Context(), connect.NewRequest(&v1.ConfigureRequest{})); !errors.Is(err, wantConfErr) {
		t.Fatalf("lh.Configure err = %v, want %v", err, wantConfErr)
	}

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(lh))
	web := httptest.NewServer(mux)
	client := NewLifecycleClient(web.Client(), web.URL)

	if ok, err := client.Configure(t.Context(), map[string]string{"a": "b"}); err == nil || ok {
		t.Fatalf("expected Configure error, got ok=%v err=%v", ok, err)
	}
	if ok, err := client.Shutdown(t.Context()); err == nil || ok || connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("expected Shutdown CodeInternal error, got ok=%v err=%v", ok, err)
	}

	web.Close()
	if _, err := client.Ping(t.Context()); err == nil {
		t.Fatal("expected Ping error after server closed")
	}
}
