package plugin

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

const listenerPlugin = "org.example.relay"

// listenLoopback binds a TCP port on loopback and closes it when the test ends.
func listenLoopback(t *testing.T) (net.Listener, int) {
	t.Helper()
	ln, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address %v is not TCP", ln.Addr())
	}
	t.Cleanup(func() {
		if err := ln.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("close the listener: %v", err)
		}
	})
	return ln, addr.Port
}

// getFreePort returns a loopback port that was free a moment ago.
func getFreePort(t *testing.T) int {
	t.Helper()
	ln, port := listenLoopback(t)
	testsupport.Close(t, ln)
	return port
}

// createListenerPlugin writes a plugin under root whose manifest declares the number setting port with
// defaultPort as its default, and the listener name on that setting.
func createListenerPlugin(t *testing.T, root, name string, defaultPort int) {
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
		"contributes": {"listeners": [{"name": %q, "portSetting": "port", "protocol": "tcp"}]},
		"settings": [{"key": "port", "type": "number", "title": "Port", "default": %d, "step": 1}]
	}`, listenerPlugin, name, defaultPort)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(m), 0o600); err != nil {
		t.Fatal(err)
	}
}

// echoErrors collects the errors of the echo connections; a reset or closed connection is not one.
type echoErrors struct {
	mu   sync.Mutex
	errs []error
}

func (e *echoErrors) add(err error) {
	if err == nil || errors.Is(err, net.ErrClosed) || errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.errs = append(e.errs, err)
}

func (e *echoErrors) get() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return errors.Join(e.errs...)
}

// echo writes back what conn reads until the peer closes it, then closes conn.
func echo(conn net.Conn) error {
	_, err := io.Copy(conn, conn)
	return errors.Join(err, conn.Close())
}

// serveEcho starts a TCP server on loopback that writes back what it reads, and returns its port.
func serveEcho(t *testing.T) int {
	t.Helper()
	var failures echoErrors
	t.Cleanup(func() {
		if err := failures.get(); err != nil {
			t.Errorf("echo server: %v", err)
		}
	})
	ln, port := listenLoopback(t)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				failures.add(err)
				return
			}
			go func() { failures.add(echo(conn)) }()
		}
	}()
	return port
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

// teardownAfter stops the plugins of host and closes its listener ports when the test ends.
func teardownAfter(t *testing.T, host *Host) {
	t.Helper()
	t.Cleanup(func() {
		if err := host.TeardownAll(context.WithoutCancel(t.Context()), 50*time.Millisecond); err != nil {
			t.Errorf("TeardownAll failed: %v", err)
		}
	})
}

// listenerHost is a host with one plugin that declares the listener mcp on port, backed by an echo server.
type listenerHost struct {
	host    *Host
	port    int
	started *atomic.Int32
}

// newUndiscoveredListenerHost creates a host for a plugin whose listener name takes the port setting
// with defaultPort as its default, and whose ready line reports listenerToken. It does not discover.
func newUndiscoveredListenerHost(t *testing.T, name string, defaultPort int, listenerToken string) listenerHost {
	t.Helper()
	root := t.TempDir()
	createListenerPlugin(t, root, name, defaultPort)
	host := NewHost(testsupport.GetMachine(), "desktop.editor", root)
	teardownAfter(t, host)
	var started atomic.Int32
	host.SetLauncher(startRelayLauncher(t, &started, listenerToken))
	return listenerHost{host: host, port: defaultPort, started: &started}
}

func discover(t *testing.T, host *Host) {
	t.Helper()
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
}

// newListenerHost creates and discovers a host whose plugin has the listener mcp on the default of its
// port setting, forwarded to an echo server.
func newListenerHost(t *testing.T) listenerHost {
	t.Helper()
	w := newUndiscoveredListenerHost(t, "mcp", getFreePort(t), "listener.mcp="+strconv.Itoa(serveEcho(t)))
	discover(t, w.host)
	return w
}

// waitForFailure waits until the plugin is failed and returns its status.
func waitForFailure(t *testing.T, host *Host) PluginStatus {
	t.Helper()
	var status PluginStatus
	plugintest.WaitUntil(t, "the plugin to be failed", func() bool {
		var ok bool
		status, ok = host.GetPluginStatus(listenerPlugin)
		return ok && status.Failed
	})
	if status.FailureReason == "" {
		t.Errorf("status = %+v, want a failure reason", status)
	}
	return status
}

func dialLoopback(ctx context.Context, port int) (net.Conn, error) {
	dialer := net.Dialer{Timeout: time.Second}
	return dialer.DialContext(ctx, "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
}

// isPortOpen reports whether a TCP connection to the loopback port is accepted.
func isPortOpen(t *testing.T, port int) bool {
	t.Helper()
	conn, err := dialLoopback(t.Context(), port)
	if err != nil {
		return false
	}
	testsupport.Close(t, conn)
	return true
}

// connectListener dials the loopback port. The connection is closed after 5 seconds, so a read that
// gets no reply ends the test, and when the test ends.
func connectListener(t *testing.T, port int) net.Conn {
	t.Helper()
	conn, err := dialLoopback(t.Context(), port)
	if err != nil {
		t.Fatalf("dial the listener port %d: %v", port, err)
	}
	closeConn := func() {
		if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("close: %v", err)
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	stop := context.AfterFunc(ctx, closeConn)
	t.Cleanup(func() {
		stop()
		cancel()
		closeConn()
	})
	return conn
}

// sendLine writes line to conn and returns the line read back.
func sendLine(t *testing.T, conn net.Conn, line string) string {
	t.Helper()
	if _, err := io.WriteString(conn, line+"\n"); err != nil {
		t.Fatalf("write to the listener: %v", err)
	}
	reply, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatalf("read from the listener: %v (read %q)", err, reply)
	}
	return strings.TrimSuffix(reply, "\n")
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

	second := connectListener(t, w.port)
	if reply := sendLine(t, second, "second"); reply != "second" {
		t.Errorf("reply = %q, want the echo of the line", reply)
	}
	if got := w.started.Load(); got != 2 {
		t.Errorf("the plugin started %d times, want 2", got)
	}
}

func TestADisabledPluginsListenerPortRefusesConnections(t *testing.T) {
	w := newListenerHost(t)
	if !isPortOpen(t, w.port) {
		t.Fatal("the listener port of the enabled plugin is not bound")
	}

	if err := w.host.SetPluginEnabled(t.Context(), listenerPlugin, false); err != nil {
		t.Fatal(err)
	}
	plugintest.WaitUntil(t, "the port of the disabled plugin to close", func() bool { return !isPortOpen(t, w.port) })

	if err := w.host.SetPluginEnabled(t.Context(), listenerPlugin, true); err != nil {
		t.Fatal(err)
	}
	plugintest.WaitUntil(t, "the port of the enabled plugin to be bound", func() bool { return isPortOpen(t, w.port) })
}

func TestAPluginDisabledInTheSettingsHasNoBoundListenerPort(t *testing.T) {
	w := newListenerHost(t)
	if !isPortOpen(t, w.port) {
		t.Fatal("the listener port of the enabled plugin is not bound")
	}

	w.host.SetSettings(map[string]map[string]any{listenerPlugin: {"enabled": false}})

	plugintest.WaitUntil(t, "the port of the disabled plugin to close", func() bool { return !isPortOpen(t, w.port) })
}

func TestTheHostBindsTheDefaultOfThePortSetting(t *testing.T) {
	w := newListenerHost(t)

	if !isPortOpen(t, w.port) {
		t.Errorf("the default port %d of the setting is not bound", w.port)
	}
}

func TestTheHostBindsTheStoredValueOfThePortSetting(t *testing.T) {
	for name, storeBeforeDiscover := range map[string]bool{"stored before discovery": true, "stored after discovery": false} {
		t.Run(name, func(t *testing.T) {
			w := newUndiscoveredListenerHost(t, "mcp", getFreePort(t), "listener.mcp="+strconv.Itoa(serveEcho(t)))
			stored := getFreePort(t)
			settings := map[string]map[string]any{listenerPlugin: {"port": float64(stored)}}
			if storeBeforeDiscover {
				w.host.SetSettings(settings)
				discover(t, w.host)
			} else {
				discover(t, w.host)
				w.host.SetSettings(settings)
			}

			plugintest.WaitUntil(t, "the stored port to be bound", func() bool { return isPortOpen(t, stored) })
			plugintest.WaitUntil(t, "the default port to be closed", func() bool { return !isPortOpen(t, w.port) })
			if reply := sendLine(t, connectListener(t, stored), "stored"); reply != "stored" {
				t.Errorf("reply = %q, want the echo of the line", reply)
			}
		})
	}
}

func TestAChangedPortSettingBindsTheNewPortAndClosesTheOldOne(t *testing.T) {
	for name, change := range map[string]func(context.Context, *Host, int) error{
		"SetPluginSetting": func(ctx context.Context, host *Host, port int) error {
			return host.SetPluginSetting(ctx, listenerPlugin, "port", float64(port))
		},
		"UpdatePluginSetting": func(ctx context.Context, host *Host, port int) error {
			return host.UpdatePluginSetting(ctx, listenerPlugin, "port", float64(port))
		},
	} {
		t.Run(name, func(t *testing.T) {
			w := newListenerHost(t)
			if !isPortOpen(t, w.port) {
				t.Fatal("the default port is not bound")
			}
			next := getFreePort(t)

			if err := change(t.Context(), w.host, next); err != nil {
				t.Fatal(err)
			}

			plugintest.WaitUntil(t, "the new port to be bound", func() bool { return isPortOpen(t, next) })
			plugintest.WaitUntil(t, "the old port to be closed", func() bool { return !isPortOpen(t, w.port) })
			if reply := sendLine(t, connectListener(t, next), "moved"); reply != "moved" {
				t.Errorf("reply = %q, want the echo of the line", reply)
			}
		})
	}
}

func TestAPortSettingOutOfRangeMakesThePluginFailed(t *testing.T) {
	for name, value := range map[string]any{
		"port 80":      float64(80),
		"port 1023":    float64(1023),
		"port 65536":   float64(65536),
		"port 0":       float64(0),
		"port 4100.5":  4100.5,
		"a true value": true,
	} {
		t.Run(name, func(t *testing.T) {
			w := newUndiscoveredListenerHost(t, "mcp", getFreePort(t), "listener.mcp="+strconv.Itoa(serveEcho(t)))
			w.host.SetSettings(map[string]map[string]any{listenerPlugin: {"port": value}})
			discover(t, w.host)

			waitForFailure(t, w.host)
		})
	}
}

func TestAPortSettingChangedOutOfRangeMakesThePluginFailedAndClosesTheOldPort(t *testing.T) {
	w := newListenerHost(t)
	if !isPortOpen(t, w.port) {
		t.Fatal("the default port is not bound")
	}

	if err := w.host.SetPluginSetting(t.Context(), listenerPlugin, "port", float64(80)); err != nil {
		t.Logf("SetPluginSetting: %v", err)
	}

	waitForFailure(t, w.host)
	plugintest.WaitUntil(t, "the old port to be closed", func() bool { return !isPortOpen(t, w.port) })
}

func TestAListenerPortThatIsTakenMakesThePluginFailed(t *testing.T) {
	_, takenPort := listenLoopback(t)
	w := newUndiscoveredListenerHost(t, "mcp", takenPort, "listener.mcp="+strconv.Itoa(serveEcho(t)))

	discover(t, w.host)

	waitForFailure(t, w.host)
}

func TestAPortSettingChangedToATakenPortMakesThePluginFailedAndClosesTheOldPort(t *testing.T) {
	w := newListenerHost(t)
	if !isPortOpen(t, w.port) {
		t.Fatal("the default port is not bound")
	}
	_, takenPort := listenLoopback(t)

	if err := w.host.SetPluginSetting(t.Context(), listenerPlugin, "port", float64(takenPort)); err != nil {
		t.Logf("SetPluginSetting: %v", err)
	}

	waitForFailure(t, w.host)
	plugintest.WaitUntil(t, "the old port to be closed", func() bool { return !isPortOpen(t, w.port) })
}

func TestAPluginWhoseReadyLineDoesNotReportItsListenerFailsToActivate(t *testing.T) {
	w := newUndiscoveredListenerHost(t, "console", getFreePort(t), "")
	discover(t, w.host)

	_, err := w.host.EnsureActive(t.Context(), listenerPlugin)

	if err == nil {
		w.host.Release(listenerPlugin)
		t.Fatal("EnsureActive succeeded for a plugin that did not report its listener")
	}
	if !strings.Contains(err.Error(), "console") {
		t.Errorf("error = %v, want it to name the listener console", err)
	}
}

func TestAPluginWhoseReadyLineHasAMalformedListenerTokenFailsToActivate(t *testing.T) {
	for name, token := range map[string]string{
		"a port that is text": "listener.mcp=open",
		"port 0":              "listener.mcp=0",
		"no name":             "listener.=41235",
		"a name given twice":  "listener.mcp=41235 listener.mcp=41236",
	} {
		t.Run(name, func(t *testing.T) {
			w := newUndiscoveredListenerHost(t, "mcp", getFreePort(t), token)
			discover(t, w.host)

			if _, err := w.host.EnsureActive(t.Context(), listenerPlugin); err == nil {
				w.host.Release(listenerPlugin)
				t.Errorf("EnsureActive succeeded for the ready line token %q", token)
			}
		})
	}
}

func TestTeardownAllClosesTheListenerPort(t *testing.T) {
	w := newListenerHost(t)
	if !isPortOpen(t, w.port) {
		t.Fatal("the listener port is not bound")
	}

	if err := w.host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}

	plugintest.WaitUntil(t, "the listener port to close", func() bool { return !isPortOpen(t, w.port) })
}

// writeFixedPortListenerPlugin writes a plugin under root whose manifest declares the listener mcp on
// the fixed port.
func writeFixedPortListenerPlugin(t *testing.T, root string, port int) {
	t.Helper()
	dir := filepath.Join(root, "relay")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	m := fmt.Sprintf(`{
		"id": %q,
		"name": "Relay",
		"version": "1.0.0",
		"description": "test plugin with a listener on a fixed port",
		"targets": ["desktop.editor"],
		"backend": {"entrypoint": "bin/relay"},
		"contributes": {"listeners": [{"name": "mcp", "port": %d, "protocol": "tcp"}]},
		"settings": []
	}`, listenerPlugin, port)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(m), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestAListenerWithAFixedPortBindsThatPortAndForwardsTheBytes(t *testing.T) {
	root := t.TempDir()
	port := getFreePort(t)
	writeFixedPortListenerPlugin(t, root, port)
	host := NewHost(testsupport.GetMachine(), "desktop.editor", root)
	teardownAfter(t, host)
	var started atomic.Int32
	host.SetLauncher(startRelayLauncher(t, &started, "listener.mcp="+strconv.Itoa(serveEcho(t))))
	discover(t, host)

	conn := connectListener(t, port)

	if reply := sendLine(t, conn, "fixed"); reply != "fixed" {
		t.Errorf("reply = %q, want the echo of the line", reply)
	}
	if got := started.Load(); got != 1 {
		t.Errorf("the plugin started %d times, want 1", got)
	}
}
