package service

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
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

// serveMounted serves Mount as a launched process of the plugin: the server reads its token from the
// standard input and writes the ready line to the standard output, which the test holds while New runs.
func serveMounted(t *testing.T) (*server.Server, *lockedBuffer) {
	t.Helper()
	t.Setenv("NUMEN_HOST_PORT", "")
	t.Setenv("NUMEN_PLUGIN_TOKEN", "")
	t.Setenv("NUMEN_PLUGIN_PORT", "")
	t.Setenv("MCP_PORT", "")
	stdinRead, stdinWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdoutRead, stdoutWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	processStdin, processStdout := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = stdinRead, stdoutWrite
	s := server.New()
	os.Stdin, os.Stdout = processStdin, processStdout
	if _, err := io.WriteString(stdinWrite, "plugin-token\n"); err != nil {
		t.Fatal(err)
	}
	stdout := &lockedBuffer{}
	copyErrCh := make(chan error, 1)
	go func() {
		_, err := io.Copy(stdout, stdoutRead)
		copyErrCh <- err
	}()
	if err := Mount(s); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}
	serveErrCh := make(chan error, 1)
	go func() { serveErrCh <- s.Serve() }()
	t.Cleanup(func() {
		s.Lifecycle().TriggerShutdown()
		if err := <-serveErrCh; err != nil {
			t.Errorf("Serve: %v", err)
		}
		for _, file := range []*os.File{stdinWrite, stdinRead, stdoutWrite} {
			if err := file.Close(); err != nil {
				t.Error(err)
			}
		}
		if err := <-copyErrCh; err != nil {
			t.Errorf("copy stdout: %v", err)
		}
		if err := stdoutRead.Close(); err != nil {
			t.Error(err)
		}
	})
	return s, stdout
}

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
	s, stdout := serveMounted(t)
	configureToken(t, s, listenerMCPToken)

	port := waitForMCPListenerPort(t, stdout)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	session := connectStreamableClient(ctx, t, "http://127.0.0.1:"+strconv.Itoa(port)+"/mcp", listenerMCPToken)
	defer closeAndCheck(t, session)
	if result := session.InitializeResult(); result == nil || result.ServerInfo == nil {
		t.Errorf("initialize result = %+v, want the server info", result)
	}
}

func configureToken(t *testing.T, s *server.Server, token string) {
	t.Helper()
	if _, err := s.Lifecycle().ApplyConfig(t.Context(), map[string]string{"token": token}); err != nil {
		t.Fatalf("configure failed: %v", err)
	}
}

func TestConfiguredNewTokenRejectsTheOldOne(t *testing.T) {
	s, stdout := serveMounted(t)
	configureToken(t, s, listenerMCPToken)
	mcpURL := "http://127.0.0.1:" + strconv.Itoa(waitForMCPListenerPort(t, stdout)) + "/mcp"
	if res := sendRequest(t, http.MethodPost, mcpURL, map[string]string{"Authorization": "Bearer " + listenerMCPToken}); res.StatusCode != http.StatusOK {
		t.Fatalf("an initialize with the configured token returned %d, want 200", res.StatusCode)
	}

	configureToken(t, s, "listener-mcp-token-2")

	assertUnauthorizedResponse(t, sendRequest(t, http.MethodPost, mcpURL,
		map[string]string{"Authorization": "Bearer " + listenerMCPToken}), "an initialize with the old token")
	if res := sendRequest(t, http.MethodPost, mcpURL, map[string]string{"Authorization": "Bearer listener-mcp-token-2"}); res.StatusCode != http.StatusOK {
		t.Errorf("an initialize with the new token returned %d, want 200", res.StatusCode)
	}
}

func TestTheMCPTokenVariableOpensNothing(t *testing.T) {
	t.Setenv("MCP_TOKEN", "variable-token-3307")
	_, stdout := serveMounted(t)
	mcpURL := "http://127.0.0.1:" + strconv.Itoa(waitForMCPListenerPort(t, stdout)) + "/mcp"

	res := sendRequest(t, http.MethodPost, mcpURL, map[string]string{"Authorization": "Bearer variable-token-3307"})

	assertUnauthorizedResponse(t, res, "an initialize with the token of MCP_TOKEN")
}

func TestAConfiguredPortOpensNoListener(t *testing.T) {
	probe, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr, ok := probe.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("probe address %v is not TCP", probe.Addr())
	}
	port := strconv.Itoa(addr.Port)
	if err := probe.Close(); err != nil {
		t.Fatal(err)
	}
	s, _ := serveMounted(t)

	if _, err := s.Lifecycle().ApplyConfig(t.Context(), map[string]string{"port": port}); err != nil {
		t.Fatalf("configure failed: %v", err)
	}

	dialer := net.Dialer{Timeout: time.Second}
	conn, err := dialer.DialContext(t.Context(), "tcp", net.JoinHostPort("127.0.0.1", port))
	if err == nil {
		closeAndCheck(t, conn)
		t.Errorf("a connection to the configured port %s was accepted", port)
	}
}
