package plugin

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

const listenerPlugin = "org.example.relay"

// getFreePort returns a loopback port that was free a moment ago.
func getFreePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

// createListenerPlugin writes a plugin under root whose manifest declares the listener name on port.
func createListenerPlugin(t *testing.T, root, name string, port int) {
	t.Helper()
	dir := filepath.Join(root, "relay")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	m := fmt.Sprintf(`{
		"id": %q,
		"name": "Relay",
		"version": "1.0.0",
		"description": "test plugin with a listener",
		"targets": ["desktop.editor"],
		"backend": {"entrypoint": "bin/relay"},
		"contributes": {"listeners": [{"name": %q, "port": %d, "protocol": "tcp"}]},
		"settings": []
	}`, listenerPlugin, name, port)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(m), 0o600); err != nil {
		t.Fatal(err)
	}
}

// serveEcho starts a TCP server on loopback that writes back what it reads, and returns its port.
func serveEcho(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = conn.Close() }()
				_, _ = io.Copy(conn, conn)
			}()
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port
}

// startRelayLauncher returns a launcher that starts the idle helper process and reports a ping server
// on its port and, when listenerToken is not empty, that token in the ready line.
func startRelayLauncher(t *testing.T, started *atomic.Int32, listenerToken string) Launcher {
	t.Helper()
	pingPort := plugintest.ServeHostPing(t)
	ready := fmt.Sprintf("NUMEN_READY port=%d", pingPort)
	if listenerToken != "" {
		ready += " " + listenerToken
	}
	return func(ctx context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		cmd := plugintest.NewIdleCommand(context.WithoutCancel(ctx))
		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		t.Cleanup(func() {
			plugintest.KillProcess(t, cmd.Process)
			plugintest.ReapProcess(t, cmd.Process)
		})
		started.Add(1)
		return cmd.Process, io.NopCloser(strings.NewReader(ready + "\n")), nil
	}
}

// listenerHost is a host with one plugin that declares the listener mcp on port, backed by an echo server.
type listenerHost struct {
	host    *Host
	port    int
	started *atomic.Int32
}

func newListenerHost(t *testing.T) listenerHost {
	t.Helper()
	root := t.TempDir()
	port := getFreePort(t)
	createListenerPlugin(t, root, "mcp", port)
	host := NewHost(testsupport.GetMachine(), "desktop.editor", root)
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), 50*time.Millisecond) })
	var started atomic.Int32
	host.SetLauncher(startRelayLauncher(t, &started, "listener.mcp="+strconv.Itoa(serveEcho(t))))
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	return listenerHost{host: host, port: port, started: &started}
}

func dialLoopback(port int) (net.Conn, error) {
	return net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
}

// isPortOpen reports whether a TCP connection to the loopback port is accepted.
func isPortOpen(port int) bool {
	conn, err := dialLoopback(port)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// sendLine writes line to conn and returns the line read back.
func sendLine(t *testing.T, conn net.Conn, line string) string {
	t.Helper()
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(conn, line+"\n"); err != nil {
		t.Fatalf("write to the listener: %v", err)
	}
	reply, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatalf("read from the listener: %v (read %q)", err, reply)
	}
	return strings.TrimSuffix(reply, "\n")
}

func connectListener(t *testing.T, port int) net.Conn {
	t.Helper()
	conn, err := dialLoopback(port)
	if err != nil {
		t.Fatalf("dial the listener port %d: %v", port, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestAConnectionToTheListenerPortStartsThePluginAndForwardsTheBytes(t *testing.T) {
	w := newListenerHost(t)
	if w.started.Load() != 0 {
		t.Fatalf("the plugin started %d times before a connection", w.started.Load())
	}

	conn := connectListener(t, w.port)

	if reply := sendLine(t, conn, "hello relay"); reply != "hello relay" {
		t.Errorf("reply = %q, want the echo of the line", reply)
	}
	if got := w.started.Load(); got != 1 {
		t.Errorf("the plugin started %d times, want 1", got)
	}
}

func TestIdleEvictionStopsThePluginOnlyAfterTheConnectionCloses(t *testing.T) {
	w := newListenerHost(t)
	conn := connectListener(t, w.port)
	if reply := sendLine(t, conn, "first"); reply != "first" {
		t.Fatalf("reply = %q, want the echo of the line", reply)
	}

	if evicted := w.host.EvictIdle(t.Context(), 0, 50*time.Millisecond); len(evicted) != 0 {
		t.Fatalf("evicted %v while a connection is open", evicted)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	plugintest.WaitUntil(t, "the idle plugin to be evicted", func() bool {
		return slices.Contains(w.host.EvictIdle(t.Context(), 0, 50*time.Millisecond), listenerPlugin)
	})

	second, err := dialLoopback(w.port)
	if err != nil {
		t.Fatalf("the listener port does not accept a connection after the plugin stopped: %v", err)
	}
	t.Cleanup(func() { _ = second.Close() })
	if reply := sendLine(t, second, "second"); reply != "second" {
		t.Errorf("reply = %q, want the echo of the line", reply)
	}
	if got := w.started.Load(); got != 2 {
		t.Errorf("the plugin started %d times, want 2", got)
	}
}

func TestADisabledPluginsListenerPortRefusesConnections(t *testing.T) {
	w := newListenerHost(t)
	if !isPortOpen(w.port) {
		t.Fatal("the listener port of the enabled plugin is not bound")
	}

	if err := w.host.SetPluginEnabled(t.Context(), listenerPlugin, false); err != nil {
		t.Fatal(err)
	}
	plugintest.WaitUntil(t, "the port of the disabled plugin to close", func() bool { return !isPortOpen(w.port) })

	if err := w.host.SetPluginEnabled(t.Context(), listenerPlugin, true); err != nil {
		t.Fatal(err)
	}
	plugintest.WaitUntil(t, "the port of the enabled plugin to be bound", func() bool { return isPortOpen(w.port) })
}

func TestAPluginDisabledInTheSettingsHasNoBoundListenerPort(t *testing.T) {
	w := newListenerHost(t)
	if !isPortOpen(w.port) {
		t.Fatal("the listener port of the enabled plugin is not bound")
	}

	w.host.SetSettings(map[string]map[string]any{listenerPlugin: {"enabled": false}})

	plugintest.WaitUntil(t, "the port of the disabled plugin to close", func() bool { return !isPortOpen(w.port) })
}

func TestAListenerPortThatIsTakenMakesThePluginFailed(t *testing.T) {
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = taken.Close() })
	root := t.TempDir()
	createListenerPlugin(t, root, "mcp", taken.Addr().(*net.TCPAddr).Port)
	host := NewHost(testsupport.GetMachine(), "desktop.editor", root)
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), 50*time.Millisecond) })
	var started atomic.Int32
	host.SetLauncher(startRelayLauncher(t, &started, "listener.mcp="+strconv.Itoa(serveEcho(t))))

	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	var status PluginStatus
	plugintest.WaitUntil(t, "the plugin to be failed", func() bool {
		status, _ = host.GetPluginStatus(listenerPlugin)
		return status.Failed
	})
	if status.FailureReason == "" {
		t.Errorf("status = %+v, want a failure reason", status)
	}
}

func TestAPluginWhoseReadyLineDoesNotReportItsListenerFailsToActivate(t *testing.T) {
	root := t.TempDir()
	createListenerPlugin(t, root, "console", getFreePort(t))
	host := NewHost(testsupport.GetMachine(), "desktop.editor", root)
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), 50*time.Millisecond) })
	var started atomic.Int32
	host.SetLauncher(startRelayLauncher(t, &started, ""))
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	_, err := host.EnsureActive(t.Context(), listenerPlugin)

	if err == nil {
		host.Release(listenerPlugin)
		t.Fatal("EnsureActive succeeded for a plugin that did not report its listener")
	}
	if !strings.Contains(err.Error(), "console") {
		t.Errorf("error = %v, want it to name the listener console", err)
	}
}

func TestTeardownAllClosesTheListenerPort(t *testing.T) {
	w := newListenerHost(t)
	if !isPortOpen(w.port) {
		t.Fatal("the listener port is not bound")
	}

	if err := w.host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}

	plugintest.WaitUntil(t, "the listener port to close", func() bool { return !isPortOpen(w.port) })
}
