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

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

// A write to a response after its handler returned panics inside net/http and ends the process, so a
// stream that clients open and cancel over and over, as a reloading window does, must leave no
// keepalive running behind the handler.
func TestAWatchEndsItsKeepaliveBeforeTheHandlerReturns(t *testing.T) {
	service, _, _ := createService(t, themedomain.Appearance{})
	service.SetKeepalive(time.Millisecond)

	route, handler := numenv1connect.NewThemeServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(route, handler)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	client := numenv1connect.NewThemeServiceClient(server.Client(), server.URL)

	for range 100 {
		ctx, cancel := context.WithCancel(t.Context())
		stream, err := client.WatchThemes(ctx, connect.NewRequest(&v1.WatchThemesRequest{}))
		if err != nil {
			t.Fatal(err)
		}
		if !stream.Receive() {
			t.Fatalf("the stream ended before a keepalive: %v", stream.Err())
		}
		cancel()
	}

	testsupport.WaitFor(t, func() bool { return !isWatchRunning() })
}

func isWatchRunning() bool {
	var stacks strings.Builder
	if err := pprof.Lookup("goroutine").WriteTo(&stacks, 2); err != nil {
		return false
	}
	return strings.Contains(stacks.String(), "theme.(*Service).WatchThemes")
}
