package theme_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/theme"
)

// TestAStreamWhoseClientWentAwayEnds. The window roots every request at the
// process, so the folder watch is never told the page it was opened from is
// gone: the editor reloads on a vault swap and leaves one behind every time. A
// write that fails is the one report there is, so the stream makes one.
func TestAStreamWhoseClientWentAwayEnds(t *testing.T) {
	service := dressed(t, theme.Appearance{}).service

	entered, returned := make(chan struct{}, 1), make(chan struct{}, 1)
	route, handler := numenv1connect.NewThemeServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(route, rooted(handler, entered, returned))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := numenv1connect.NewThemeServiceClient(server.Client(), server.URL)
	if _, err := client.Changed(t.Context(), connect.NewRequest(&v1.ChangedRequest{})); err != nil {
		t.Fatal(err)
	}

	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		t.Fatal("the stream never reached the handler")
	}

	// The page is gone, and nothing cancelled the context the handler holds.
	server.CloseClientConnections()

	select {
	case <-returned:
	case <-time.After(10 * time.Second):
		t.Fatal("the client went away and the stream is still standing")
	}
}

// rooted serves the handler the way the window does: the request context is the
// process's own, and ends when the application ends and at no other moment.
func rooted(handler http.Handler, entered, returned chan<- struct{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		defer func() { returned <- struct{}{} }()
		handler.ServeHTTP(w, r.WithContext(context.Background()))
	})
}
