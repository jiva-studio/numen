package service

import (
	"bytes"
	"context"
	"net"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cleanup"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

const listenerMCPToken = "listener-mcp-token"

var readyMCPListenerPattern = regexp.MustCompile(`NUMEN_READY port=\d+[^\n]* listener\.mcp=(\d+)[ \n]`)

// lockedBuffer is a bytes.Buffer safe for the server goroutine writing and the test reading.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// serveMounted mounts the plugin on a plugin server, serves it and returns the server and its stdout.
func serveMounted(t *testing.T) (*server.Server, *lockedBuffer) {
	t.Helper()
	t.Setenv("NUMEN_HOST_PORT", "")
	t.Setenv("NUMEN_PLUGIN_TOKEN", "")
	t.Setenv("MCP_PORT", "")
	t.Setenv("MCP_TOKEN", listenerMCPToken)
	stdout := &lockedBuffer{}
	s := server.New(server.WithPort(0), server.WithToken("plugin-token"),
		server.WithStdout(stdout), server.WithStdin(nil), server.WithSignals())
	if err := Mount(s); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}
	serveErrCh := make(chan error, 1)
	go func() { serveErrCh <- s.Serve() }()
	t.Cleanup(func() {
		s.Lifecycle().TriggerShutdown()
		<-serveErrCh
	})
	return s, stdout
}

// waitForMCPListenerPort returns the port the ready line reports for the listener mcp.
func waitForMCPListenerPort(t *testing.T, stdout *lockedBuffer) int {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		if match := readyMCPListenerPattern.FindStringSubmatch(stdout.String()); match != nil {
			port, err := strconv.Atoi(match[1])
			if err != nil {
				t.Fatalf("listener port %q: %v", match[1], err)
			}
			return port
		}
		select {
		case <-deadline:
			t.Fatalf("no ready line with listener.mcp, stdout %q", stdout.String())
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestMountServesMCPOnTheListenerTheReadyLineReports(t *testing.T) {
	_, stdout := serveMounted(t)

	port := waitForMCPListenerPort(t, stdout)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	session := connectStreamableClient(ctx, t, "http://127.0.0.1:"+strconv.Itoa(port)+"/mcp", listenerMCPToken)
	defer cleanup.Close(t, session)
	if result := session.InitializeResult(); result == nil || result.ServerInfo == nil {
		t.Errorf("initialize result = %+v, want the server info", result)
	}
}

func TestAConfiguredPortOpensNoListener(t *testing.T) {
	probe, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := strconv.Itoa(probe.Addr().(*net.TCPAddr).Port)
	if err := probe.Close(); err != nil {
		t.Fatal(err)
	}
	s, _ := serveMounted(t)

	if _, err := s.Lifecycle().ApplyConfig(t.Context(), map[string]string{"port": port}); err != nil {
		t.Fatalf("configure failed: %v", err)
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", port), time.Second)
	if err == nil {
		cleanup.Close(t, conn)
		t.Errorf("a connection to the configured port %s was accepted", port)
	}
}
