package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

func TestPingResult_RoundTripsThroughTheWire(t *testing.T) {
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
	if err != nil || ping.Status != "ok" || ping.Protocol != ProtocolVersion {
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
