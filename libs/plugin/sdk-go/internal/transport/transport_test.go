package transport_test

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/internal/transport"
)

type fakeStreamConn struct {
	connect.StreamingClientConn
	header http.Header
}

func (f *fakeStreamConn) RequestHeader() http.Header { return f.header }

func TestTheTokenIsPresentedOnAStreamingCall(t *testing.T) {
	conn := &fakeStreamConn{header: http.Header{}}
	wrapped := transport.Bearer("s").WrapStreamingClient(func(context.Context, connect.Spec) connect.StreamingClientConn { return conn })

	gotConn := wrapped(t.Context(), connect.Spec{})

	if gotConn != conn {
		t.Fatalf("got conn %v, want %v", gotConn, conn)
	}
	if got := conn.header.Get("Authorization"); got != "Bearer s" {
		t.Errorf("got %q", got)
	}
}

func TestNoTokenIsPresentedAsNone(t *testing.T) {
	conn := &fakeStreamConn{header: http.Header{}}

	transport.Bearer("").WrapStreamingClient(func(context.Context, connect.Spec) connect.StreamingClientConn { return conn })(t.Context(), connect.Spec{})

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

	if _, err := transport.Bearer("u").WrapUnary(next)(t.Context(), connect.NewRequest(&struct{}{})); err != nil {
		t.Fatal(err)
	}

	if observed != "Bearer u" {
		t.Errorf("got %q", observed)
	}
}

func TestAStreamingHandlerIsPassedThrough(t *testing.T) {
	called := false
	next := func(context.Context, connect.StreamingHandlerConn) error { called = true; return nil }

	if err := transport.Bearer("s").WrapStreamingHandler(next)(t.Context(), nil); err != nil {
		t.Fatal(err)
	}

	if !called {
		t.Error("handler not called")
	}
}

func TestAClientOfTheHostSendsWithTheTokenOnEveryCall(t *testing.T) {
	options := transport.Options(hostcall.Endpoint{BaseURL: "http://127.0.0.1:1", Token: "t"})

	if len(options) != 1 || transport.HTTPClient() != http.DefaultClient {
		t.Error("the client is not made as the plugin makes it")
	}
}
