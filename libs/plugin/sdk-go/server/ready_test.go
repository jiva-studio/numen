package server

import (
	"fmt"
	"net/http"
	"strings"
	"syscall"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// pipeOutput is standard output as a plugin process has it on macOS: a pipe whose Sync fails with
// EBADF.
type pipeOutput struct {
	safeBuffer
}

func (*pipeOutput) Sync() error { return syscall.EBADF }

func TestServeKeepsServingWhenOutputCannotSync(t *testing.T) {
	const token = "pipe-token"
	var stdout pipeOutput
	s := New(withPort(0), withToken(token), withStdout(&stdout), withStdin(nil), withSignals())
	port, err := s.Listen()
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	serveErrCh := make(chan error, 1)
	go func() { serveErrCh <- s.Serve() }()
	t.Cleanup(func() {
		s.Lifecycle().TriggerShutdown()
		if err := <-serveErrCh; err != nil {
			t.Errorf("Serve: %v", err)
		}
	})

	deadline := time.After(5 * time.Second)
	for !strings.Contains(stdout.String(), fmt.Sprintf("NUMEN_READY port=%d\n", port)) {
		select {
		case <-deadline:
			t.Fatalf("no ready line, stdout %q", stdout.String())
		case <-time.After(10 * time.Millisecond):
		}
	}
	select {
	case err := <-serveErrCh:
		serveErrCh <- err
		t.Fatalf("Serve returned after the ready line: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	client := numenv1connect.NewPluginLifecycleServiceClient(
		http.DefaultClient,
		fmt.Sprintf("http://127.0.0.1:%d", port),
		connect.WithInterceptors(&bearerAuthInterceptor{token: token}),
	)
	if _, err := client.Ping(t.Context(), connect.NewRequest(&v1.PingRequest{})); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
