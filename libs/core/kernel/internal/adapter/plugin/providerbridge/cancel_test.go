package providerbridge_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/providerbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
)

const cancelWait = 5 * time.Second

func newBlockedService() *scriptedService {
	return &scriptedService{
		frames: []*v1.ProviderServiceCallResponse{data("partial")}, block: true,
		started: make(chan struct{}), ended: make(chan error, 1),
	}
}

func cancelRunningCall(t *testing.T, provider connectionport.Provider, service *scriptedService) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	outcome := make(chan error, 1)
	var frames []string
	go func() {
		_, err := callCollecting(ctx, provider, &frames)
		outcome <- err
	}()
	select {
	case <-service.started:
	case <-time.After(cancelWait):
		t.Fatal("the provider never received the call")
	}
	cancel()
	select {
	case got := <-service.ended:
		if !errors.Is(got, context.Canceled) {
			t.Fatalf("the handler context ended with %v, want context.Canceled", got)
		}
	case <-time.After(cancelWait):
		t.Fatal("the context of the provider's handler did not end after the caller canceled")
	}
	select {
	case err := <-outcome:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Call = %v, want context.Canceled", err)
		}
	case <-time.After(cancelWait):
		t.Fatal("Call did not return after it was canceled")
	}
}

func TestACanceledCallReachesTheProviderOverTheInProcessPipe(t *testing.T) {
	service := newBlockedService()
	provider, _, _ := newBridge(t, service)
	cancelRunningCall(t, provider, service)
}

func TestACanceledCallReachesTheProviderOverTheLoopback(t *testing.T) {
	service := newBlockedService()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewProviderServiceHandler(service))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	address, isTCP := server.Listener.Addr().(*net.TCPAddr)
	if !isTCP {
		t.Fatal("the test server does not listen on TCP")
	}
	host := &activator{endpoint: transport.Endpoint{Host: address.IP.String(), Port: address.Port, Token: token}}
	provider, _, err := providerbridge.NewSource(host).OpenProvider(t.Context(), pluginID)
	if err != nil {
		t.Fatal(err)
	}
	cancelRunningCall(t, provider, service)
}
