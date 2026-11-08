package providerbridge_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin"
)

const cancelWait = 5 * time.Second

// blockingProvider answers Complete when its context ends, and tells the test which way it ended.
type blockingProvider struct {
	numenv1connect.UnimplementedProviderServiceHandler
	started chan struct{}
	ended   chan error
}

func (p *blockingProvider) Complete(
	ctx context.Context, _ *connect.Request[v1.ProviderCompleteRequest],
) (*connect.Response[v1.ProviderCompleteResponse], error) {
	close(p.started)
	<-ctx.Done()
	p.ended <- ctx.Err()
	return nil, ctx.Err()
}

func newBlockingProvider() *blockingProvider {
	return &blockingProvider{started: make(chan struct{}), ended: make(chan error, 1)}
}

func cancelRunningCall(t *testing.T, client numenv1connect.ProviderServiceClient, provider *blockingProvider) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	failed := make(chan error, 1)
	go func() {
		_, err := client.Complete(ctx, connect.NewRequest(&v1.ProviderCompleteRequest{Call: &v1.ProviderCall{Model: "m"}}))
		failed <- err
	}()
	select {
	case <-provider.started:
	case <-time.After(cancelWait):
		t.Fatal("the provider never received the call")
	}
	cancel()
	select {
	case <-provider.ended:
	case <-time.After(cancelWait):
		t.Fatal("the context of the provider's handler did not end after the caller cancelled")
	}
	select {
	case err := <-failed:
		if connect.CodeOf(err) != connect.CodeCanceled {
			t.Fatalf("the caller got code %v, want canceled: %v", connect.CodeOf(err), err)
		}
	case <-time.After(cancelWait):
		t.Fatal("the caller's call did not return after it was cancelled")
	}
}

func TestACancelledCallReachesTheProviderOverTheInProcessPipe(t *testing.T) {
	provider := newBlockingProvider()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewProviderServiceHandler(provider))
	client := numenv1connect.NewProviderServiceClient(
		&http.Client{Transport: plugin.NewInProcessTransport(mux)}, "http://inprocess")
	cancelRunningCall(t, client, provider)
}

func TestACancelledCallReachesTheProviderOverTheLoopback(t *testing.T) {
	provider := newBlockingProvider()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewProviderServiceHandler(provider))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := numenv1connect.NewProviderServiceClient(server.Client(), server.URL)
	cancelRunningCall(t, client, provider)
}
