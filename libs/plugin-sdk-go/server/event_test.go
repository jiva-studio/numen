package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

const eventToken = "event-token"

// serveEvents starts a server built with opts and returns its base URL.
func serveEvents(t *testing.T, opts ...Option) string {
	t.Helper()
	s := New(append([]Option{WithPort(0), WithToken(eventToken), WithStdout(io.Discard), WithStdin(nil)}, opts...)...)
	port, err := s.Listen()
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	serveErrCh := make(chan error, 1)
	go func() { serveErrCh <- s.Serve() }()
	t.Cleanup(func() {
		s.Lifecycle().TriggerShutdown()
		if err := <-serveErrCh; err != nil {
			t.Errorf("Serve returned error: %v", err)
		}
	})
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}

// newEventClient returns a client of the event service at baseURL that sends token.
func newEventClient(baseURL, token string) numenv1connect.PluginEventServiceClient {
	return numenv1connect.NewPluginEventServiceClient(
		http.DefaultClient,
		baseURL,
		connect.WithInterceptors(&bearerAuthInterceptor{token: token}),
	)
}

func TestTheEventHandlerReceivesADeliveredEvent(t *testing.T) {
	var mu sync.Mutex
	var received []string
	client := newEventClient(serveEvents(t, WithEventHandler(func(_ context.Context, event string) error {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, event)
		return nil
	})), eventToken)

	resp, err := client.DeliverEvent(t.Context(), connect.NewRequest(&numenv1.DeliverEventRequest{Event: "onFile:pdf"}))
	if err != nil {
		t.Fatalf("DeliverEvent failed: %v", err)
	}
	if code := resp.Msg.GetError(); code != numenv1.PluginEventErrorCode_PLUGIN_EVENT_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("error = %v, want no error", code)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 || received[0] != "onFile:pdf" {
		t.Fatalf("received %v, want [onFile:pdf]", received)
	}
}

func TestEvent_HandlerRejectionReturnsError(t *testing.T) {
	cases := map[string]error{
		"a plain error":           errors.New("the index is closed"),
		"an Unimplemented status": connect.NewError(connect.CodeUnimplemented, errors.New("the index is closed")),
	}
	for name, caseErr := range cases {
		t.Run(name, func(t *testing.T) {
			client := newEventClient(serveEvents(t, WithEventHandler(func(_ context.Context, _ string) error {
				return caseErr
			})), eventToken)

			resp, err := client.DeliverEvent(t.Context(), connect.NewRequest(&numenv1.DeliverEventRequest{Event: "onFile:pdf"}))
			if err != nil {
				t.Fatalf("DeliverEvent() = %v, want the call served", err)
			}
			if code := resp.Msg.GetError(); code != numenv1.PluginEventErrorCode_PLUGIN_EVENT_ERROR_CODE_REFUSED {
				t.Fatalf("error = %v, want %v", code, numenv1.PluginEventErrorCode_PLUGIN_EVENT_ERROR_CODE_REFUSED)
			}
			if !strings.Contains(resp.Msg.GetReason(), "the index is closed") {
				t.Fatalf("reason = %q, want the handler's error", resp.Msg.GetReason())
			}
		})
	}
}

func TestAServerWithoutAnEventHandlerDoesNotTakeEvents(t *testing.T) {
	client := newEventClient(serveEvents(t), eventToken)

	_, err := client.DeliverEvent(t.Context(), connect.NewRequest(&numenv1.DeliverEventRequest{Event: "onFile:pdf"}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("DeliverEvent() = %v, want %v", err, connect.CodeUnimplemented)
	}
}

func TestAnEventWithoutTheTokenIsRefused(t *testing.T) {
	var calls atomic.Int32
	baseURL := serveEvents(t, WithEventHandler(func(_ context.Context, _ string) error {
		calls.Add(1)
		return nil
	}))

	_, err := newEventClient(baseURL, "wrong-token").DeliverEvent(t.Context(), connect.NewRequest(&numenv1.DeliverEventRequest{Event: "onFile:pdf"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("DeliverEvent() = %v, want %v", err, connect.CodeUnauthenticated)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("the handler ran %d times, want 0", got)
	}
}
