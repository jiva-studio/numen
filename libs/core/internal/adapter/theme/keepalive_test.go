package theme_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/theme"
)

// TestAStreamWhoseClientWentAwayEnds. The window roots every request at the
// process, so the folder watch is not notified when its page disconnects: the
// editor reloads on a vault swap and leaves one watch behind each time. A failed
// write is the only disconnect signal, so the stream writes.
func TestAStreamWhoseClientWentAwayEnds(t *testing.T) {
	service := createFakeAppearances(t, theme.Appearance{}).service

	entered, returned := make(chan struct{}, 1), make(chan struct{}, 1)
	route, handler := numenv1connect.NewThemeServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(route, wrapWithProcessContext(handler, entered, returned))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := numenv1connect.NewThemeServiceClient(server.Client(), server.URL)
	if _, err := client.WatchThemes(t.Context(), connect.NewRequest(&v1.WatchThemesRequest{})); err != nil {
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

	deadline := time.Now().Add(10 * time.Second)
	for isWatchingFolder() {
		if time.Now().After(deadline) {
			t.Fatal("the stream ended and the folder watch is still running")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// isWatchingFolder reports whether a folder watch goroutine is running, which
// means the folder registration is still held.
func isWatchingFolder() bool {
	var stacks strings.Builder
	if err := pprof.Lookup("goroutine").WriteTo(&stacks, 2); err != nil {
		return false
	}
	return strings.Contains(stacks.String(), "theme.Catalogue.debounce")
}

// wrapWithProcessContext serves the handler the way the window does: the request context is the
// process's own, and ends when the application ends and at no other moment.
func wrapWithProcessContext(handler http.Handler, entered, returned chan<- struct{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		defer func() { returned <- struct{}{} }()
		handler.ServeHTTP(w, r.WithContext(context.Background()))
	})
}
