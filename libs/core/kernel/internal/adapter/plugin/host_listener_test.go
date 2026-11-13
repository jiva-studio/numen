package plugin_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
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

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const listenerPlugin = "org.example.relay"

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

func getFreePort(t *testing.T) int {
	t.Helper()
	ln, port := listenLoopback(t)
	testsupport.Close(t, ln)
	return port
}

func changePortToFree(t *testing.T, host *container.PluginRoles, change func(context.Context, *container.PluginRoles, int) error) int {
	t.Helper()
	var err error
	for range 5 {
		port := getFreePort(t)
		if err = change(t.Context(), host, port); err == nil {
			return port
		}
		if !errors.Is(err, syscall.EADDRINUSE) {
			break
		}
	}
	t.Fatal(err)
	return 0
}

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
		"settings": [
			{"key": "port", "type": "number", "title": "Port", "default": %d, "step": 1},
			{"key": "greeting", "type": "string", "title": "Greeting", "default": "good morning"}
		]
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

func echo(conn net.Conn) error {
	_, err := io.Copy(conn, conn)
	return errors.Join(err, conn.Close())
}

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

func startRelayLauncher(t *testing.T, started *atomic.Int32, listenerToken string) supervision.Launcher {
	t.Helper()
	pingPort := wiretest.ServeHostPing(t)
	ready := fmt.Sprintf("NUMEN_READY port=%d", pingPort)
	if listenerToken != "" {
		ready += " " + listenerToken
	}
	return func(ctx context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		cmd := wiretest.NewIdleCommand(context.WithoutCancel(ctx))
		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		t.Cleanup(func() {
			wiretest.KillProcess(t, cmd.Process)
			wiretest.ReapProcess(t, cmd.Process)
		})
		started.Add(1)
		return cmd.Process, io.NopCloser(strings.NewReader(ready + "\n")), nil
	}
}

func teardownAfter(t *testing.T, host *container.PluginRoles) {
	t.Helper()
	t.Cleanup(func() {
		if err := host.Teardown.TeardownAll(context.WithoutCancel(t.Context()), 50*time.Millisecond); err != nil {
			t.Errorf("TeardownAll failed: %v", err)
		}
	})
}

// listenerHost is a host with one plugin that declares the listener mcp on port, backed by an echo server.
type listenerHost struct {
	host    *container.PluginRoles
	port    int
	started *atomic.Int32
}

func newUndiscoveredListenerHost(t *testing.T, name string, defaultPort int, listenerToken string) listenerHost {
	t.Helper()
	root := t.TempDir()
	createListenerPlugin(t, root, name, defaultPort)
	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", root)
	teardownAfter(t, host)
	var started atomic.Int32
	host.Processes.SetLauncher(startRelayLauncher(t, &started, listenerToken))
	return listenerHost{host: host, port: defaultPort, started: &started}
}

func discover(t *testing.T, host *container.PluginRoles) {
	t.Helper()
	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if _, ok := host.Registry.GetPluginStatus(listenerPlugin); !ok {
		t.Fatalf("the plugin %s is not registered; statuses %+v", listenerPlugin, host.Registry.GetPluginStatuses())
	}
}

func newListenerHost(t *testing.T) listenerHost {
	t.Helper()
	w := newUndiscoveredListenerHost(t, "mcp", getFreePort(t), "listener.mcp="+strconv.Itoa(serveEcho(t)))
	discover(t, w.host)
	return w
}

func waitForFailure(t *testing.T, host *container.PluginRoles) plugin.PluginStatus {
	t.Helper()
	var status plugin.PluginStatus
	wiretest.WaitUntil(t, "the plugin to be failed", func() bool {
		var ok bool
		status, ok = host.Registry.GetPluginStatus(listenerPlugin)
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

func isPortOpen(t *testing.T, port int) bool {
	t.Helper()
	conn, err := dialLoopback(t.Context(), port)
	if err != nil {
		return false
	}
	testsupport.Close(t, conn)
	return true
}

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

func TestListenerConnectionStartsThePluginAndForwardsBytes(t *testing.T) {
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

	if evicted := w.host.Teardown.EvictIdle(t.Context(), 0, 50*time.Millisecond); len(evicted) != 0 {
		t.Fatalf("evicted %v while a connection is open", evicted)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	wiretest.WaitUntil(t, "the idle plugin to be evicted", func() bool {
		return slices.Contains(w.host.Teardown.EvictIdle(t.Context(), 0, 50*time.Millisecond), listenerPlugin)
	})

	second := connectListener(t, w.port)
	if reply := sendLine(t, second, "second"); reply != "second" {
		t.Errorf("reply = %q, want the echo of the line", reply)
	}
	if got := w.started.Load(); got != 2 {
		t.Errorf("the plugin started %d times, want 2", got)
	}
}

func TestDisabledPluginListenerPortRejectsConnections(t *testing.T) {
	w := newListenerHost(t)
	if !isPortOpen(t, w.port) {
		t.Fatal("the listener port of the enabled plugin is not bound")
	}

	if err := setEnabled(t.Context(), w.host, listenerPlugin, false); err != nil {
		t.Fatal(err)
	}
	wiretest.WaitUntil(t, "the port of the disabled plugin to close", func() bool { return !isPortOpen(t, w.port) })

	if err := setEnabled(t.Context(), w.host, listenerPlugin, true); err != nil {
		t.Fatal(err)
	}
	wiretest.WaitUntil(t, "the port of the enabled plugin to be bound", func() bool { return isPortOpen(t, w.port) })
}

func TestAPluginDisabledInTheSettingsHasNoBoundListenerPort(t *testing.T) {
	w := newListenerHost(t)
	if !isPortOpen(t, w.port) {
		t.Fatal("the listener port of the enabled plugin is not bound")
	}

	w.host.Settings.SetSettings(map[string]map[string]any{listenerPlugin: {"enabled": false}})

	wiretest.WaitUntil(t, "the port of the disabled plugin to close", func() bool { return !isPortOpen(t, w.port) })
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
				w.host.Settings.SetSettings(settings)
				discover(t, w.host)
			} else {
				discover(t, w.host)
				w.host.Settings.SetSettings(settings)
			}

			wiretest.WaitUntil(t, "the stored port to be bound", func() bool { return isPortOpen(t, stored) })
			wiretest.WaitUntil(t, "the default port to be closed", func() bool { return !isPortOpen(t, w.port) })
			if reply := sendLine(t, connectListener(t, stored), "stored"); reply != "stored" {
				t.Errorf("reply = %q, want the echo of the line", reply)
			}
		})
	}
}

func TestAChangedPortSettingBindsTheNewPortAndClosesTheOldOne(t *testing.T) {
	for name, change := range map[string]func(context.Context, *container.PluginRoles, int) error{
		"UpdatePluginSetting": func(ctx context.Context, host *container.PluginRoles, port int) error {
			return updateSetting(ctx, host, listenerPlugin, "port", float64(port))
		},
	} {
		t.Run(name, func(t *testing.T) {
			w := newListenerHost(t)
			if !isPortOpen(t, w.port) {
				t.Fatal("the default port is not bound")
			}
			next := changePortToFree(t, w.host, change)

			wiretest.WaitUntil(t, "the new port to be bound", func() bool { return isPortOpen(t, next) })
			wiretest.WaitUntil(t, "the old port to be closed", func() bool { return !isPortOpen(t, w.port) })
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
			w.host.Settings.SetSettings(map[string]map[string]any{listenerPlugin: {"port": value}})
			discover(t, w.host)

			waitForFailure(t, w.host)
		})
	}
}

func TestListenerPortInUseMarksThePluginFailed(t *testing.T) {
	_, takenPort := listenLoopback(t)
	w := newUndiscoveredListenerHost(t, "mcp", takenPort, "listener.mcp="+strconv.Itoa(serveEcho(t)))

	discover(t, w.host)

	waitForFailure(t, w.host)
}

func TestPortSettingChangedToUnbindablePortKeepsOldPortAndStatus(t *testing.T) {
	changes := map[string]func(context.Context, *container.PluginRoles, any) error{
		"UpdatePluginSetting": func(ctx context.Context, host *container.PluginRoles, value any) error {
			return updateSetting(ctx, host, listenerPlugin, "port", value)
		},
	}
	values := map[string]func(t *testing.T) any{
		"port 80": func(*testing.T) any { return float64(80) },
		"a taken port": func(t *testing.T) any {
			t.Helper()
			_, takenPort := listenLoopback(t)
			return float64(takenPort)
		},
	}
	for changeName, change := range changes {
		for valueName, getValue := range values {
			t.Run(changeName+" to "+valueName, func(t *testing.T) {
				w := newListenerHost(t)
				if reply := sendLine(t, connectListener(t, w.port), "before"); reply != "before" {
					t.Fatalf("reply = %q, want the echo of the line", reply)
				}
				value := getValue(t)

				if err := change(t.Context(), w.host, value); err == nil {
					t.Errorf("changing the port setting to %v returned no error", value)
				}

				if status, _ := w.host.Registry.GetPluginStatus(listenerPlugin); status.Failed {
					t.Errorf("status = %+v after the port setting change was rejected, want not failed", status)
				}
				if reply := sendLine(t, connectListener(t, w.port), "kept"); reply != "kept" {
					t.Errorf("reply = %q on the old port, want the echo of the line", reply)
				}
			})
		}
	}
}

func TestReadyLineWithoutListenerFailsActivation(t *testing.T) {
	w := newUndiscoveredListenerHost(t, "console", getFreePort(t), "")
	discover(t, w.host)

	_, err := w.host.Activation.EnsureActive(t.Context(), listenerPlugin, testHolder)

	if err == nil {
		w.host.Activation.Release(listenerPlugin, endpointOf(w.host, listenerPlugin).Token, testHolder)
		t.Fatal("EnsureActive succeeded for a plugin that did not report its listener")
	}
	if !strings.Contains(err.Error(), "console") {
		t.Errorf("error = %v, want it to name the listener console", err)
	}
}

func TestReadyLineWithMalformedListenerTokenFailsActivation(t *testing.T) {
	for name, token := range map[string]string{
		"a port that is text": "listener.mcp=open",
		"port 0":              "listener.mcp=0",
		"no name":             "listener.=41235",
		"a name given twice":  "listener.mcp=41235 listener.mcp=41236",
	} {
		t.Run(name, func(t *testing.T) {
			w := newUndiscoveredListenerHost(t, "mcp", getFreePort(t), token)
			discover(t, w.host)

			if _, err := w.host.Activation.EnsureActive(t.Context(), listenerPlugin, testHolder); err == nil {
				w.host.Activation.Release(listenerPlugin, endpointOf(w.host, listenerPlugin).Token, testHolder)
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

	if err := w.host.Teardown.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}

	wiretest.WaitUntil(t, "the listener port to close", func() bool { return !isPortOpen(t, w.port) })
}

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
	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", root)
	teardownAfter(t, host)
	var started atomic.Int32
	host.Processes.SetLauncher(startRelayLauncher(t, &started, "listener.mcp="+strconv.Itoa(serveEcho(t))))
	discover(t, host)

	conn := connectListener(t, port)

	if reply := sendLine(t, conn, "fixed"); reply != "fixed" {
		t.Errorf("reply = %q, want the echo of the line", reply)
	}
	if got := started.Load(); got != 1 {
		t.Errorf("the plugin started %d times, want 1", got)
	}
}

func getStoredSettings(t *testing.T, host *container.PluginRoles) map[string]any {
	t.Helper()
	raw, err := host.Settings.GetPluginConfigJSON(listenerPlugin)
	if err != nil {
		t.Fatal(err)
	}
	var stored map[string]any
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		t.Fatalf("parse %s: %v", raw, err)
	}
	return stored
}

// persistedKeys records the keys the persist handler of a host is called with.
type persistedKeys struct {
	mu   sync.Mutex
	keys []string
}

func (p *persistedKeys) add(_, key string, _ any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.keys = append(p.keys, key)
	return nil
}

func (p *persistedKeys) get() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.keys)
}

func TestCheckPluginSettingAcceptsOnlyFreeOrBoundPort(t *testing.T) {
	w := newListenerHost(t)
	_, takenPort := listenLoopback(t)
	freePort := getFreePort(t)

	if err := w.host.Settings.CheckPluginSetting(listenerPlugin, "port", float64(takenPort)); err == nil {
		t.Errorf("CheckPluginSetting accepted the taken port %d", takenPort)
	}
	if err := w.host.Settings.CheckPluginSetting(listenerPlugin, "port", float64(80)); !errors.Is(err, plugin.ErrListenerPortValue) {
		t.Errorf("CheckPluginSetting of port 80 = %v, want plugin.ErrListenerPortValue", err)
	}
	if err := w.host.Settings.CheckPluginSetting(listenerPlugin, "port", float64(freePort)); err != nil {
		t.Errorf("CheckPluginSetting of the free port %d = %v, want nil", freePort, err)
	}
	if isPortOpen(t, freePort) {
		t.Errorf("the free port %d stays bound after the check", freePort)
	}
	if err := w.host.Settings.CheckPluginSetting(listenerPlugin, "port", float64(w.port)); err != nil {
		t.Errorf("CheckPluginSetting of the bound port %d = %v, want nil", w.port, err)
	}
	if err := w.host.Settings.CheckPluginSetting(listenerPlugin, "greeting", "good evening"); err != nil {
		t.Errorf("CheckPluginSetting of greeting = %v, want nil", err)
	}
	if reply := sendLine(t, connectListener(t, w.port), "checked"); reply != "checked" {
		t.Errorf("reply = %q on the bound port after the checks, want the echo of the line", reply)
	}
}

func TestUnbindablePortSettingIsNotStoredAndOthersUpdate(t *testing.T) {
	changes := map[string]func(context.Context, *container.PluginRoles, string, any) error{
		"UpdatePluginSetting": func(ctx context.Context, host *container.PluginRoles, key string, value any) error {
			return updateSetting(ctx, host, listenerPlugin, key, value)
		},
	}
	values := map[string]func(t *testing.T) any{
		"port 80": func(*testing.T) any { return float64(80) },
		"a taken port": func(t *testing.T) any {
			t.Helper()
			_, takenPort := listenLoopback(t)
			return float64(takenPort)
		},
	}
	for changeName, change := range changes {
		for valueName, getValue := range values {
			t.Run(changeName+" to "+valueName, func(t *testing.T) {
				w := newListenerHost(t)
				var persisted persistedKeys
				w.host.Settings.SetPersistHandler(persisted.add)
				value := getValue(t)

				if err := change(t.Context(), w.host, "port", value); err == nil {
					t.Errorf("writing the port setting %v returned no error", value)
				}

				if keys := persisted.get(); slices.Contains(keys, "port") {
					t.Errorf("persisted keys = %v, want the rejected port not persisted", keys)
				}
				if got := getStoredSettings(t, w.host)["port"]; got != float64(w.port) {
					t.Errorf("port in memory = %v, want the old port %d", got, w.port)
				}
				if reply := sendLine(t, connectListener(t, w.port), "kept"); reply != "kept" {
					t.Errorf("reply = %q on the old port, want the echo of the line", reply)
				}

				if err := change(t.Context(), w.host, "greeting", "good evening"); err != nil {
					t.Fatalf("writing greeting after the rejected port = %v, want nil", err)
				}
				if got := getStoredSettings(t, w.host)["greeting"]; got != "good evening" {
					t.Errorf("greeting in memory = %v, want good evening", got)
				}
				if got := getStoredSettings(t, w.host)["port"]; got != float64(w.port) {
					t.Errorf("port in memory after writing greeting = %v, want the old port %d", got, w.port)
				}
			})
		}
	}
}
