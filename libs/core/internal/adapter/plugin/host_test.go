package plugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/process"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/sharedmimeinfo"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	machineport "github.com/jiva-studio/numen/libs/core/usecase/file/machinecontract"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

func newDiscoveredHostAt(t *testing.T, clock machineport.Clock, root string) *Host {
	t.Helper()
	host := NewHost(clock, "desktop.editor", root)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	return host
}

func launchProcess(cmd *exec.Cmd, fallback int) Launcher {
	return func(_ context.Context, _ manifest.Discovered, port int, _ string) (*os.Process, io.ReadCloser, error) {
		if port == 0 {
			port = fallback
		}
		return cmd.Process, plugintest.GetReadyOutput(port), nil
	}
}

func launchProcessAt(cmd *exec.Cmd, port int) Launcher {
	return func(context.Context, manifest.Discovered, int, string) (*os.Process, io.ReadCloser, error) {
		return cmd.Process, plugintest.GetReadyOutput(port), nil
	}
}

func serveLifecycle(t *testing.T, handler numenv1connect.PluginLifecycleServiceHandler) int {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(handler))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return plugintest.GetPort(t, srv)
}

func createNamedTestPlugin(t *testing.T, dir, id string) {
	t.Helper()
	pDir := filepath.Join(dir, id)
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	manifest := fmt.Sprintf(`{
		"id": %q,
		"name": %q,
		"version": "1.0.0",
		"description": "test plugin",
		"capabilities": ["tools"],
		"targets": ["desktop.editor"],
		"permissions": ["vault:read"],
		"services": ["%[1]s.TestService"], "backend": {"entrypoint": "bin/test"},
		"frontend": {},
		"settings": []
	}`, id, id)
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}
}

func TestHostDiscoveryAndLookup(t *testing.T) {
	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)

	host := newDiscoveredHostAt(t, testsupport.GetMachine(), tmp)

	plugins := host.GetPlugins()
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}

	p, ok := host.GetPlugin("org.example.sample")
	if !ok || p.Manifest.ID != "org.example.sample" {
		t.Fatal("expected to find org.example.sample")
	}

	capPlugin, ok := host.FindPluginByCapability("tools")
	if !ok || capPlugin.Manifest.ID != "org.example.sample" {
		t.Fatal("expected to find plugin by capability tools")
	}
}

func TestHostLifecycleWithMockLauncher(t *testing.T) {
	tmp := t.TempDir()
	pingPort := plugintest.ServeHostPing(t)
	plugintest.CreateTestPluginTree(t, tmp)

	host := NewHost(testsupport.GetMachine(), "desktop.editor", tmp)
	host.SetContentTypeTable(sharedmimeinfo.Open())
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	cmd := plugintest.StartSleepProcess(t)

	host.SetLauncher(launchProcess(cmd, pingPort))

	ctx := t.Context()
	endpoint, err := host.EnsureActive(ctx, "org.example.sample")
	if err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	if endpoint.Port == 0 || endpoint.Token == "" {
		t.Fatalf("expected non-zero port and token, got %+v", endpoint)
	}

	// Calling EnsureActive again should return the existing active endpoint
	endpoint2, err := host.EnsureActive(ctx, "org.example.sample")
	if err != nil {
		t.Fatalf("second EnsureActive failed: %v", err)
	}
	if endpoint.Port != endpoint2.Port || endpoint.Token != endpoint2.Token {
		t.Fatalf("expected identical endpoints, got %+v and %+v", endpoint, endpoint2)
	}

	// Release twice to enter idle state
	host.Release("org.example.sample")
	host.Release("org.example.sample")

	// TriggerEvent for matching event
	activated, err := host.TriggerEvent(ctx, "onFile:test.pdf")
	if err != nil {
		t.Fatalf("TriggerEvent failed: %v", err)
	}
	if len(activated) != 1 || activated[0] != "org.example.sample" {
		t.Fatalf("expected org.example.sample to be activated, got %v", activated)
	}

	// EvictIdle test
	evicted := host.EvictIdle(ctx, 10*time.Millisecond, 50*time.Millisecond)
	// The plugin holds a lease, so it is not evicted
	if len(evicted) != 0 {
		t.Fatalf("expected 0 evicted before threshold, got %v", evicted)
	}

	// Cleanup
	host.Release("org.example.sample")
	if err := host.TeardownAll(ctx, 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestHostEvictLoopTeardown(t *testing.T) {
	host := NewHost(testsupport.GetMachine(), "desktop.editor")
	if host.isEvictStopped {
		t.Fatal("expected evict loop to be running")
	}

	ctx := t.Context()
	if err := host.TeardownAll(ctx, 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}

	if !host.isEvictStopped {
		t.Fatal("expected evict loop to be stopped after TeardownAll")
	}

	if err := host.TeardownAll(ctx, 50*time.Millisecond); err != nil {
		t.Fatalf("second TeardownAll failed: %v", err)
	}
}

func TestIsWhitelistedEnv(t *testing.T) {
	allowed := []string{
		"PATH", "path", "Path",
		"HOME", "home",
		"USER", "user",
		"TMP", "tmp",
		"TEMP", "temp",
		"SYSTEMROOT", "SystemRoot",
		"SYSTEMDRIVE", "SystemDrive",
		"APPDATA", "AppData",
		"LOCALAPPDATA", "LocalAppData",
		"LANG", "lang",
		"LC_ALL", "lc_ctype", "LC_MESSAGES",
	}
	for _, k := range allowed {
		if !process.IsWhitelistedEnv(k) {
			t.Errorf("expected %s to be allowed in environment whitelist", k)
		}
	}

	disallowed := []string{
		"AWS_SECRET_ACCESS_KEY",
		"GITHUB_TOKEN",
		"PRIVATE_KEY",
		"DATABASE_URL",
		"SECRET",
		"MY_VAR",
		"FOO",
	}
	for _, k := range disallowed {
		if process.IsWhitelistedEnv(k) {
			t.Errorf("expected %s to be blocked from environment whitelist", k)
		}
	}
}

func TestHostEmitEventActivation(t *testing.T) {
	tmp := t.TempDir()
	pingPort := plugintest.ServeHostPing(t)
	plugintest.CreateTestPluginTree(t, tmp)

	host := NewHost(testsupport.GetMachine(), "desktop.editor", tmp)
	host.SetContentTypeTable(sharedmimeinfo.Open())
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	cmd := plugintest.StartSleepProcess(t)

	host.SetLauncher(launchProcess(cmd, pingPort))

	ctx := t.Context()

	// Initial state is dormant
	sup := host.supervisors["org.example.sample"]
	if sup == nil || sup.GetState() != lifecycle.StateDormant {
		t.Fatalf("expected initial state dormant, got %v", sup.GetState())
	}

	// Emit onFile event via debounced event bus
	host.EmitEvent("onFile:document.pdf")
	if err := host.GetEventBus().Flush(ctx); err != nil {
		t.Fatal(err)
	}

	// Plugin should now be activated
	if sup.GetState() != lifecycle.StateActive && sup.GetState() != lifecycle.StateIdle {
		t.Fatalf("expected state active or idle after EmitEvent, got %v", sup.GetState())
	}

	host.Release("org.example.sample")
	if err := host.TeardownAll(ctx, 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestHostDrainsStdoutAfterPasseshake(t *testing.T) {
	tmp := t.TempDir()
	pingPort := plugintest.ServeHostPing(t)
	plugintest.CreateTestPluginTree(t, tmp)

	host := newDiscoveredHostAt(t, testsupport.GetMachine(), tmp)

	cmd := plugintest.StartSleepProcess(t)

	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe failed: %v", err)
	}

	host.SetLauncher(func(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		return cmd.Process, pr, nil
	})

	writeDone := make(chan error, 1)
	go func() {
		defer testsupport.Close(t, pw)
		if _, err := pw.Write([]byte(fmt.Sprintf("NUMEN_READY port=%d\n", pingPort))); err != nil {
			writeDone <- err
			return
		}
		// Write >64KB to overflow the standard OS pipe buffer.
		largePayload := bytes.Repeat([]byte("A"), 128*1024)
		if _, err := pw.Write(largePayload); err != nil {
			writeDone <- err
			return
		}
		writeDone <- nil
	}()

	ctx := t.Context()
	endpoint, err := host.EnsureActive(ctx, "org.example.sample")
	if err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	if endpoint.Port != pingPort {
		t.Fatalf("expected port %d, got %d", pingPort, endpoint.Port)
	}

	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatalf("write to pipe failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for large stdout write to complete")
	}

	host.Release("org.example.sample")
	if err := host.TeardownAll(ctx, 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestHostEnsureActiveConcurrentSamePlugin(t *testing.T) {
	tmp := t.TempDir()
	pingPort := plugintest.ServeHostPing(t)
	createNamedTestPlugin(t, tmp, "test.concurrent")

	host := newDiscoveredHostAt(t, testsupport.GetMachine(), tmp)

	cmd := plugintest.StartSleepProcess(t)

	var launchCount atomic.Int32
	launchStarted := make(chan struct{})
	allowHandshake := make(chan struct{})
	var once sync.Once

	host.SetLauncher(func(_ context.Context, _ manifest.Discovered, port int, _ string) (*os.Process, io.ReadCloser, error) {
		launchCount.Add(1)
		once.Do(func() {
			close(launchStarted)
		})
		<-allowHandshake
		actualPort := port
		if actualPort == 0 {
			actualPort = pingPort
		}
		return cmd.Process, plugintest.GetReadyOutput(actualPort), nil
	})

	const numGoroutines = 8
	var wg sync.WaitGroup
	endpoints := make([]transport.Endpoint, numGoroutines)
	errs := make([]error, numGoroutines)

	wg.Go(func() {
		endpoints[0], errs[0] = host.EnsureActive(t.Context(), "test.concurrent")
	})

	<-launchStarted

	for i := 1; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			endpoints[idx], errs[idx] = host.EnsureActive(t.Context(), "test.concurrent")
		}(i)
	}

	close(allowHandshake)
	wg.Wait()

	for i := range numGoroutines {
		if errs[i] != nil {
			t.Fatalf("goroutine %d returned error: %v", i, errs[i])
		}
		if endpoints[i].Port != pingPort {
			t.Fatalf("goroutine %d port = %d, want %d", i, endpoints[i].Port, pingPort)
		}
		if endpoints[i].Token != endpoints[0].Token {
			t.Fatalf("goroutine %d token mismatch: %s vs %s", i, endpoints[i].Token, endpoints[0].Token)
		}
	}

	if count := launchCount.Load(); count != 1 {
		t.Fatalf("expected 1 process launch, got %d", count)
	}

	for range numGoroutines {
		host.Release("test.concurrent")
	}
	if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestHostEnsureActiveEliminatesHeadOfLineBlocking(t *testing.T) {
	tmp := t.TempDir()
	slowPort := plugintest.ServeHostPing(t)
	fastPort := plugintest.ServeHostPing(t)
	createNamedTestPlugin(t, tmp, "plugin.slow")
	createNamedTestPlugin(t, tmp, "plugin.fast")

	host := newDiscoveredHostAt(t, testsupport.GetMachine(), tmp)

	cmdSlow := plugintest.StartSleepProcess(t)

	cmdFast := plugintest.StartSleepProcess(t)

	slowStarted := make(chan struct{})
	slowAllowHandshake := make(chan struct{})

	host.SetLauncher(func(_ context.Context, p manifest.Discovered, port int, _ string) (*os.Process, io.ReadCloser, error) {
		if p.Manifest.ID == "plugin.slow" {
			close(slowStarted)
			<-slowAllowHandshake
			actualPort := port
			if actualPort == 0 {
				actualPort = slowPort
			}
			return cmdSlow.Process, plugintest.GetReadyOutput(actualPort), nil
		}
		actualPort := port
		if actualPort == 0 {
			actualPort = fastPort
		}
		return cmdFast.Process, plugintest.GetReadyOutput(actualPort), nil
	})

	var slowWg sync.WaitGroup
	var slowEndpoint transport.Endpoint
	var slowErr error

	slowWg.Go(func() {
		slowEndpoint, slowErr = host.EnsureActive(t.Context(), "plugin.slow")
	})

	<-slowStarted

	fastEndpoint, err := host.EnsureActive(t.Context(), "plugin.fast")
	if err != nil {
		t.Fatalf("EnsureActive on fast plugin failed while slow plugin was handshaking: %v", err)
	}
	if fastEndpoint.Port != fastPort {
		t.Fatalf("fast plugin expected port %d, got %d", fastPort, fastEndpoint.Port)
	}

	plugins := host.GetPlugins()
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
	status, ok := host.GetPluginStatus("plugin.fast")
	if !ok || !status.IsActive {
		t.Fatalf("expected plugin.fast to be active, got ok=%v, active=%v", ok, status.IsActive)
	}

	close(slowAllowHandshake)
	slowWg.Wait()

	if slowErr != nil {
		t.Fatalf("slow plugin EnsureActive failed: %v", slowErr)
	}
	if slowEndpoint.Port != slowPort {
		t.Fatalf("slow plugin expected port %d, got %d", slowPort, slowEndpoint.Port)
	}

	host.Release("plugin.slow")
	host.Release("plugin.fast")
	if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestHostEnsureActiveConcurrentFailure(t *testing.T) {
	tmp := t.TempDir()
	createNamedTestPlugin(t, tmp, "test.fail")

	host := newDiscoveredHostAt(t, testsupport.GetMachine(), tmp)

	launchStarted := make(chan struct{})
	allowFailure := make(chan struct{})
	var once sync.Once
	failErr := errors.New("simulated launch failure")

	host.SetLauncher(func(_ context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		once.Do(func() {
			close(launchStarted)
		})
		<-allowFailure
		return nil, nil, failErr
	})

	const numGoroutines = 4
	var wg sync.WaitGroup
	errs := make([]error, numGoroutines)

	wg.Go(func() {
		_, errs[0] = host.EnsureActive(t.Context(), "test.fail")
	})

	<-launchStarted

	for i := 1; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = host.EnsureActive(t.Context(), "test.fail")
		}(i)
	}

	close(allowFailure)
	wg.Wait()

	for i := range numGoroutines {
		if errs[i] == nil {
			t.Fatalf("goroutine %d expected error, got nil", i)
		}
	}

	status, ok := host.GetPluginStatus("test.fail")
	if !ok {
		t.Fatal("expected status for test.fail")
	}
	if status.IsActive {
		t.Fatal("expected plugin to be inactive after failure")
	}
}

func TestHostEnsureActiveContextCanceledWhileActivating(t *testing.T) {
	tmp := t.TempDir()
	pingPort := plugintest.ServeHostPing(t)
	createNamedTestPlugin(t, tmp, "test.cancel")

	host := newDiscoveredHostAt(t, testsupport.GetMachine(), tmp)

	cmd := plugintest.StartSleepProcess(t)

	launchStarted := make(chan struct{})
	allowHandshake := make(chan struct{})

	host.SetLauncher(func(_ context.Context, _ manifest.Discovered, port int, _ string) (*os.Process, io.ReadCloser, error) {
		close(launchStarted)
		<-allowHandshake
		actualPort := port
		if actualPort == 0 {
			actualPort = pingPort
		}
		return cmd.Process, plugintest.GetReadyOutput(actualPort), nil
	})

	var wg sync.WaitGroup
	var activeEp transport.Endpoint
	var activeErr error

	wg.Go(func() {
		activeEp, activeErr = host.EnsureActive(t.Context(), "test.cancel")
	})

	<-launchStarted

	cancelCtx, cancel := context.WithCancel(t.Context())
	cancelErrCh := make(chan error, 1)
	go func() {
		_, err := host.EnsureActive(cancelCtx, "test.cancel")
		cancelErrCh <- err
	}()

	cancel()

	select {
	case err := <-cancelErrCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for canceled EnsureActive call")
	}

	close(allowHandshake)
	wg.Wait()

	if activeErr != nil {
		t.Fatalf("primary activation failed: %v", activeErr)
	}
	if activeEp.Port != pingPort {
		t.Fatalf("expected port %d, got %d", pingPort, activeEp.Port)
	}

	host.Release("test.cancel")
	if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

type testCloser struct {
	mu       sync.Mutex
	isClosed bool
}

func (c *testCloser) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isClosed = true
	return nil
}

func (c *testCloser) getIsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isClosed
}

type testLifecycleHandler struct {
	testsupport.PluginLifecycle
	mu             sync.Mutex
	shutdownCalled bool
	timeoutSecs    int32
	onShutdown     func()
}

func (h *testLifecycleHandler) Shutdown(
	_ context.Context,
	req *connect.Request[v1.ShutdownRequest],
) (*connect.Response[v1.ShutdownResponse], error) {
	h.mu.Lock()
	h.shutdownCalled = true
	h.timeoutSecs = req.Msg.GetTimeoutSeconds()
	onShutdown := h.onShutdown
	h.mu.Unlock()
	if onShutdown != nil {
		onShutdown()
	}
	return connect.NewResponse(&v1.ShutdownResponse{Success: true}), nil
}

func (h *testLifecycleHandler) isShutdownCalled() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.shutdownCalled
}

func TestHostEvictIdleSendsRPCShutdownAndClosesStdin(t *testing.T) {
	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)

	clock := testsupport.NewClock()
	host := newDiscoveredHostAt(t, clock, tmp)

	handler := &testLifecycleHandler{}
	serverPort := serveLifecycle(t, handler)

	cmd := plugintest.StartSleepProcess(t)

	handler.onShutdown = func() {
		testsupport.Kill(t, cmd)
	}

	host.SetLauncher(launchProcessAt(cmd, serverPort))

	ctx := t.Context()
	endpoint, err := host.EnsureActive(ctx, "org.example.sample")
	if err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	if endpoint.Port != serverPort {
		t.Fatalf("expected port %d, got %d", serverPort, endpoint.Port)
	}

	closer := &testCloser{}
	host.stdinsMu.Lock()
	host.stdins[stdinKey{pluginID: "org.example.sample", pid: cmd.Process.Pid}] = closer
	host.stdinsMu.Unlock()

	host.Release("org.example.sample")
	host.Release("org.example.sample")

	clock.Advance(1050 * time.Millisecond)

	evicted := host.EvictIdle(ctx, 0, 500*time.Millisecond)
	if len(evicted) != 1 || evicted[0] != "org.example.sample" {
		t.Fatalf("expected org.example.sample to be evicted, got %v", evicted)
	}

	if !handler.isShutdownCalled() {
		t.Error("expected PluginLifecycleService.Shutdown RPC to be sent")
	}

	if !closer.getIsClosed() {
		t.Error("expected stdin closer to be closed on eviction")
	}

	host.stdinsMu.Lock()
	_, stdinExists := host.stdins[stdinKey{pluginID: "org.example.sample", pid: cmd.Process.Pid}]
	host.stdinsMu.Unlock()
	if stdinExists {
		t.Error("expected stdin pipe to be deleted from host.stdins")
	}

	sup := host.supervisors["org.example.sample"]
	if sup.GetState() != lifecycle.StateDormant {
		t.Errorf("expected StateDormant, got %v", sup.GetState())
	}
}

func TestHostTeardownAllSendsRPCShutdownAndClosesStdin(t *testing.T) {
	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)

	host := newDiscoveredHostAt(t, testsupport.GetMachine(), tmp)

	handler := &testLifecycleHandler{}
	serverPort := serveLifecycle(t, handler)

	cmd := plugintest.StartSleepProcess(t)

	handler.onShutdown = func() {
		testsupport.Kill(t, cmd)
	}

	host.SetLauncher(launchProcessAt(cmd, serverPort))

	ctx := t.Context()
	if _, err := host.EnsureActive(ctx, "org.example.sample"); err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}

	closer := &testCloser{}
	host.stdinsMu.Lock()
	host.stdins[stdinKey{pluginID: "org.example.sample", pid: cmd.Process.Pid}] = closer
	host.stdinsMu.Unlock()

	if err := host.TeardownAll(ctx, 500*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}

	if !handler.isShutdownCalled() {
		t.Error("expected PluginLifecycleService.Shutdown RPC to be sent during TeardownAll")
	}

	if !closer.getIsClosed() {
		t.Error("expected stdin closer to be closed during TeardownAll")
	}

	host.stdinsMu.Lock()
	stdinCount := len(host.stdins)
	host.stdinsMu.Unlock()
	if stdinCount != 0 {
		t.Errorf("expected empty host.stdins, got %d entries", stdinCount)
	}
}

func TestHostEvictIdleNonBlocking(t *testing.T) {
	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)

	clock := testsupport.NewClock()
	host := newDiscoveredHostAt(t, clock, tmp)

	handler := &testLifecycleHandler{}
	serverPort := serveLifecycle(t, handler)

	cmd := plugintest.StartSleepProcess(t)

	rpcReceived := make(chan struct{})
	releaseShutdown := make(chan struct{})
	handler.onShutdown = func() {
		close(rpcReceived)
		<-releaseShutdown
		testsupport.Kill(t, cmd)
	}

	host.SetLauncher(launchProcessAt(cmd, serverPort))

	ctx := t.Context()
	if _, err := host.EnsureActive(ctx, "org.example.sample"); err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}

	host.Release("org.example.sample")
	host.Release("org.example.sample")

	clock.Advance(1050 * time.Millisecond)

	isDone := make(chan struct{})
	go func() {
		defer close(isDone)
		_ = host.EvictIdle(ctx, 0, 1*time.Second)
	}()

	<-rpcReceived

	if !host.mu.TryLock() {
		t.Fatal("expected host.mu to be unlocked while EvictIdle is tearing down")
	}
	host.mu.Unlock()

	close(releaseShutdown)
	<-isDone
}

func startSleepLauncher(t *testing.T, started *atomic.Int32, procs *[]*os.Process) Launcher {
	t.Helper()
	pingPort := plugintest.ServeHostPing(t)
	var mu sync.Mutex
	return func(ctx context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		cmd := plugintest.NewIdleCommand(context.WithoutCancel(ctx))
		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		t.Cleanup(func() { plugintest.KillProcess(t, cmd.Process) })
		mu.Lock()
		if procs != nil {
			*procs = append(*procs, cmd.Process)
		}
		mu.Unlock()
		started.Add(1)
		return cmd.Process, plugintest.GetReadyOutput(pingPort), nil
	}
}

func TestACrashedPluginIsStartedAgainAfterAConsumerReleasesIt(t *testing.T) {
	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)
	host := newDiscoveredHostAt(t, testsupport.GetMachine(), tmp)
	var started atomic.Int32
	var procs []*os.Process
	host.SetLauncher(startSleepLauncher(t, &started, &procs))

	first, err := host.EnsureActive(t.Context(), "org.example.sample")
	if err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}

	host.mu.RLock()
	sup := host.supervisors["org.example.sample"]
	host.mu.RUnlock()
	plugintest.KillProcess(t, procs[0])
	plugintest.WaitUntil(t, "the crashed plugin to become dormant", func() bool {
		host.mu.RLock()
		defer host.mu.RUnlock()
		_, hasEndpoint := host.endpoints["org.example.sample"]
		return sup.GetState() == lifecycle.StateDormant && !hasEndpoint
	})

	host.Release("org.example.sample")
	if sup.GetState() != lifecycle.StateDormant {
		t.Fatalf("Release moved a crashed plugin to %v", sup.GetState())
	}

	second, err := host.EnsureActive(t.Context(), "org.example.sample")
	if err != nil {
		t.Fatalf("EnsureActive after crash failed: %v", err)
	}
	if second.Port == 0 || second.Token == "" || second.Token == first.Token {
		t.Fatalf("expected a fresh running endpoint, got %+v", second)
	}
	if started.Load() != 2 {
		t.Fatalf("expected the plugin to be launched twice, got %d", started.Load())
	}
	if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestAcquireAndReleaseDoNothingWithoutARunningProcess(t *testing.T) {
	sup := lifecycle.New(testsupport.GetMachine())
	if sup.Acquire(testsupport.GetMachine().Now()) {
		t.Fatal("Acquire held a dormant supervisor")
	}
	sup.Release(testsupport.GetMachine().Now())
	if sup.GetState() != lifecycle.StateDormant {
		t.Fatalf("Release moved a dormant supervisor to %v", sup.GetState())
	}
	sup.SetActivating()
	if sup.Acquire(testsupport.GetMachine().Now()) {
		t.Fatal("Acquire held an activating supervisor")
	}
	sup.Release(testsupport.GetMachine().Now())
	if sup.GetState() != lifecycle.StateActivating {
		t.Fatalf("Release moved an activating supervisor to %v", sup.GetState())
	}
}

func TestEnsureActiveRetiresARunningProcessWhoseEndpointIsMissing(t *testing.T) {
	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)
	host := newDiscoveredHostAt(t, testsupport.GetMachine(), tmp)
	var started atomic.Int32
	var procs []*os.Process
	host.SetLauncher(startSleepLauncher(t, &started, &procs))

	if _, err := host.EnsureActive(t.Context(), "org.example.sample"); err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	host.mu.Lock()
	delete(host.endpoints, "org.example.sample")
	sup := host.supervisors["org.example.sample"]
	host.mu.Unlock()
	firstExit := sup.GetExitSignal()

	endpoint, err := host.EnsureActive(t.Context(), "org.example.sample")
	if err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	if isEmptyEndpoint(endpoint) {
		t.Fatal("EnsureActive returned an empty endpoint")
	}
	if started.Load() != 2 {
		t.Fatalf("expected a replacement launch, got %d launches", started.Load())
	}
	select {
	case <-firstExit:
	case <-time.After(5 * time.Second):
		t.Fatal("the unreachable process was not stopped")
	}
	if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestTheExitOfAProcessDropsItsEndpoint(t *testing.T) {
	host := NewHost(testsupport.GetMachine(), "desktop.editor")
	host.mu.Lock()
	host.endpoints["p"] = transport.Endpoint{Host: "127.0.0.1", Port: 4321, Token: "old"}
	host.mu.Unlock()

	host.dropEndpointOfStoppedPlugin("p")
	host.mu.RLock()
	_, stillThere := host.endpoints["p"]
	host.mu.RUnlock()
	if stillThere {
		t.Fatal("the endpoint of a stopped plugin was kept")
	}
}

func TestAPluginInUseIsNotEvictedWhileItIsBeingAcquired(t *testing.T) {
	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)
	clock := testsupport.NewClock()
	host := newDiscoveredHostAt(t, clock, tmp)
	var started atomic.Int32
	host.SetLauncher(startSleepLauncher(t, &started, nil))
	var evictions atomic.Int32
	stop := make(chan struct{})
	var evictor sync.WaitGroup
	evictor.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
				evictions.Add(int32(len(host.EvictIdle(t.Context(), 0, 20*time.Millisecond))))
			}
		}
	})
	var stopOnce sync.Once
	stopEvictor := func() {
		stopOnce.Do(func() { close(stop) })
		evictor.Wait()
	}
	t.Cleanup(stopEvictor)

	for i := 0; i < 60 || (evictions.Load() == 0 && i < 5000); i++ {
		endpoint, err := host.EnsureActive(t.Context(), "org.example.sample")
		if err != nil {
			t.Fatalf("round %d: EnsureActive failed: %v", i, err)
		}
		if isEmptyEndpoint(endpoint) {
			t.Fatalf("round %d: EnsureActive returned an empty endpoint", i)
		}
		host.mu.RLock()
		sup := host.supervisors["org.example.sample"]
		host.mu.RUnlock()
		if state := sup.GetState(); state != lifecycle.StateActive {
			t.Fatalf("round %d: a held plugin is in state %v", i, state)
		}
		host.Release("org.example.sample")
		if i == 0 {
			clock.Advance(1050 * time.Millisecond)
		}
	}
	stopEvictor()
	if evictions.Load() == 0 {
		t.Fatal("the evictor never stopped an idle plugin, so the race was not exercised")
	}
	if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestCloseStdin_PreservesNewerProcessStdin(t *testing.T) {
	host := NewHost(testsupport.GetMachine(), "desktop.editor")
	oldPipe, newPipe := &testCloser{}, &testCloser{}
	host.stdinsMu.Lock()
	host.stdins[stdinKey{pluginID: "p", pid: 1}] = oldPipe
	host.stdins[stdinKey{pluginID: "p", pid: 2}] = newPipe
	host.stdinsMu.Unlock()

	host.closeStdin("p", 1)

	if !oldPipe.getIsClosed() {
		t.Fatal("expected the evicted process stdin to be closed")
	}
	if newPipe.getIsClosed() {
		t.Fatal("closed the stdin of a newer process")
	}
	if err := host.TeardownAll(t.Context(), 10*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestEnsureActiveWaitsForAPluginThatIsBeingDeactivated(t *testing.T) {
	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)
	host := newDiscoveredHostAt(t, testsupport.GetMachine(), tmp)
	var started atomic.Int32
	host.SetLauncher(startSleepLauncher(t, &started, nil))
	first, err := host.EnsureActive(t.Context(), "org.example.sample")
	if err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	host.Release("org.example.sample")
	host.mu.Lock()
	sup := host.supervisors["org.example.sample"]
	delete(host.endpoints, "org.example.sample")
	host.mu.Unlock()
	if !sup.BeginDeactivation(0, testsupport.GetMachine().Now()) {
		t.Fatal("expected the idle plugin to begin deactivating")
	}

	type result struct {
		endpoint transport.Endpoint
		err      error
	}
	reply := make(chan result, 1)
	go func() {
		endpoint, err := host.EnsureActive(t.Context(), "org.example.sample")
		reply <- result{endpoint: endpoint, err: err}
	}()

	select {
	case got := <-reply:
		t.Fatalf("EnsureActive answered while the plugin was deactivating: %+v", got)
	case <-time.After(50 * time.Millisecond):
	}

	if err := sup.Teardown(t.Context(), 50*time.Millisecond, nil); err != nil {
		t.Fatalf("Teardown failed: %v", err)
	}
	got := <-reply
	if got.err != nil {
		t.Fatalf("EnsureActive failed: %v", got.err)
	}
	if isEmptyEndpoint(got.endpoint) || got.endpoint.Token == first.Token {
		t.Fatalf("expected a fresh endpoint from a new process, got %+v", got.endpoint)
	}
	if started.Load() != 2 {
		t.Fatalf("expected the plugin to be launched twice, got %d", started.Load())
	}
	if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func newHostOfProviders(t *testing.T) *Host {
	t.Helper()
	host := NewHost(testsupport.GetMachine(), "desktop.editor")
	t.Cleanup(func() {
		if err := host.TeardownAll(context.WithoutCancel(t.Context()), time.Second); err != nil {
			t.Error(err)
		}
	})
	for _, id := range []string{"md.numen.provider-c", "md.numen.provider-a", "md.numen.provider-b"} {
		host.plugins[id] = manifest.Discovered{Manifest: manifest.Manifest{
			ID:           id,
			Capabilities: []string{bridge.CapabilityEmbedding},
		}}
	}
	return host
}

// Of several enabled plugins with the capability, the one with the lowest id is
// found, at every lookup.
func TestTheLowestIDOfSeveralProvidersIsFound(t *testing.T) {
	host := newHostOfProviders(t)
	for range 100 {
		byCapability, ok := host.FindPluginByCapability(bridge.CapabilityEmbedding)
		if !ok || byCapability.Manifest.ID != "md.numen.provider-a" {
			t.Fatalf("by capability found %q, %v", byCapability.Manifest.ID, ok)
		}
	}

	host.SetSettings(map[string]map[string]any{"md.numen.provider-a": {settingEnabled: false}})
	if found, _ := host.FindPluginByCapability(bridge.CapabilityEmbedding); found.Manifest.ID != "md.numen.provider-b" {
		t.Errorf("with the lowest disabled, found %q", found.Manifest.ID)
	}
}

// The host lists its plugins in the order of their ids.
func TestTheHostListsPluginsInIDOrder(t *testing.T) {
	host := newHostOfProviders(t)
	want := []string{"md.numen.provider-a", "md.numen.provider-b", "md.numen.provider-c"}
	for range 20 {
		var items, infos, statuses []string
		for _, p := range host.GetPlugins() {
			items = append(items, p.Manifest.ID)
		}
		for _, info := range host.GetPluginInfos() {
			infos = append(infos, info.ID)
		}
		for _, status := range host.GetPluginStatuses() {
			statuses = append(statuses, status.ID)
		}
		for name, got := range map[string][]string{"plugins": items, "infos": infos, "statuses": statuses} {
			if !slices.Equal(got, want) {
				t.Fatalf("%s listed %v, want %v", name, got, want)
			}
		}
	}
}

func TestHostEvictIdleKeepsAPluginIdleForLessThanTheThreshold(t *testing.T) {
	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)

	host := newDiscoveredHostAt(t, testsupport.GetMachine(), tmp)

	var started atomic.Int32
	host.SetLauncher(startSleepLauncher(t, &started, nil))

	ctx := t.Context()
	if _, err := host.EnsureActive(ctx, "org.example.sample"); err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}

	host.Release("org.example.sample")
	host.Release("org.example.sample")

	evicted := host.EvictIdle(ctx, idleThreshold, 50*time.Millisecond)
	if len(evicted) != 0 {
		t.Fatalf("expected 0 evicted before the idle threshold, got %v", evicted)
	}

	host.mu.RLock()
	sup := host.supervisors["org.example.sample"]
	host.mu.RUnlock()
	if state := sup.GetState(); state != lifecycle.StateIdle {
		t.Errorf("expected supervisor to stay StateIdle, got %v", state)
	}

	if err := host.TeardownAll(ctx, 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

// TestHelperProcessIdles is the body of the child process the launcher starts; it does nothing in a normal run.
func TestHelperProcessIdles(t *testing.T) { plugintest.IdleChild(t) }

// runningPlugin is a plugin that the test host treats as running.
type runningPlugin struct {
	token string
	perms []string
}

func newRunningTestHost(t *testing.T, plugins map[string]runningPlugin) *Host {
	t.Helper()
	host := NewHost(testsupport.GetMachine(), "desktop.editor")
	t.Cleanup(func() {
		if err := host.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})
	for id, one := range plugins {
		sup := lifecycle.New(testsupport.GetMachine())
		sup.SetPermissions(one.perms)
		host.supervisors[id] = sup
		host.endpoints[id] = transport.Endpoint{Host: "127.0.0.1", Port: 1, Token: one.token}
	}
	return host
}

// The host names a running plugin by the token it was launched with.
func TestTheHostNamesAPluginByItsToken(t *testing.T) {
	host := newRunningTestHost(t, map[string]runningPlugin{
		"md.numen.a": {"token-a", []string{manifest.PermissionIndexRead}},
		"md.numen.b": {"token-b", nil},
		"md.numen.c": {"", nil},
	})

	for token, want := range map[string]string{"token-a": "md.numen.a", "token-b": "md.numen.b"} {
		if got, found := host.GetPluginIDByToken(token); !found || got != want {
			t.Errorf("token %q names %q, %v; want %q", token, got, found, want)
		}
	}
	for _, token := range []string{"", "token-", "token-ab", "other"} {
		if got, found := host.GetPluginIDByToken(token); found {
			t.Errorf("token %q names %q", token, got)
		}
	}
	if err := host.CheckPluginPermission("md.numen.b", manifest.PermissionIndexRead); !errors.Is(err, pluginport.ErrPermissionDenied) {
		t.Errorf("a plugin without the permission was allowed: %v", err)
	}
	if err := host.CheckPluginPermission("md.numen.a", manifest.PermissionIndexRead); err != nil {
		t.Errorf("a plugin with the permission failed: %v", err)
	}
}

// The host endpoint serves only once something answers the calls on models, and
// stops when the host does.
func TestTheHostEndpointOpensOnceAndStaysClosedAfterTeardown(t *testing.T) {
	host := NewHost(testsupport.GetMachine(), "desktop.editor")
	if got, err := host.openHostEndpoint(t.Context()); err != nil || got != 0 {
		t.Fatalf("without model calls: port %d, %v", got, err)
	}

	host.SetModelCalls(noModelCalls{})
	first, err := host.openHostEndpoint(t.Context())
	if err != nil || first <= 0 {
		t.Fatalf("with model calls: port %d, %v", first, err)
	}
	again, err := host.openHostEndpoint(t.Context())
	if err != nil || again != first {
		t.Errorf("a second call opened port %d, %v, want %d", again, err, first)
	}

	if err := host.TeardownAll(t.Context(), time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if got, err := host.openHostEndpoint(t.Context()); got != 0 {
		t.Errorf("a closed host opened port %d, %v", got, err)
	}
}

// The port is passed to a plugin process in its environment only while the
// endpoint serves, and the token is the one issued at launch.
func TestThePluginEnvironmentCarriesTheHostPort(t *testing.T) {
	with := process.GetPluginVars("md.numen.a", process.LaunchEnvironment{Port: 5, HostPort: 7777, Token: "t"})
	if !slices.Contains(with, "NUMEN_HOST_PORT=7777") || !slices.Contains(with, "NUMEN_PLUGIN_TOKEN=t") {
		t.Errorf("environment %v lacks the host port or the token", with)
	}
	defaultVars := process.GetPluginVars("md.numen.a", process.LaunchEnvironment{Port: 5, Token: "t"})
	for _, entry := range defaultVars {
		if strings.HasPrefix(entry, "NUMEN_HOST_PORT=") {
			t.Errorf("environment %v names a host that is not serving", defaultVars)
		}
	}
}

// A plugin process launched while the host endpoint serves receives its port.
func TestALaunchedPluginIsReportedTheHostPort(t *testing.T) {
	host := NewHost(testsupport.GetMachine(), "desktop.editor")
	t.Cleanup(func() {
		if err := host.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})
	host.SetModelCalls(noModelCalls{})

	vars, err := host.getLaunchVars(t.Context(), "md.numen.a", nil, 5, "t")
	if err != nil {
		t.Fatal(err)
	}
	served, err := host.openHostEndpoint(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if want := fmt.Sprintf("NUMEN_HOST_PORT=%d", served); served == 0 || !slices.Contains(vars, want) {
		t.Errorf("environment %v lacks %s", vars, want)
	}
}

// An endpoint that cannot start is reported and does not stop a plugin from launching.
func TestAHostEndpointThatCannotStartDoesNotStopALaunch(t *testing.T) {
	host := NewHost(testsupport.GetMachine(), "desktop.editor")
	t.Cleanup(func() {
		if err := host.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})
	host.hostAddress = "127.0.0.1:-1"
	host.SetModelCalls(noModelCalls{})
	var reported []error
	host.SetErrorHandler(func(err error) { reported = append(reported, err) })

	vars, err := host.getLaunchVars(t.Context(), "md.numen.a", nil, 5, "t")
	if err != nil {
		t.Fatalf("the launch failed: %v", err)
	}
	for _, entry := range vars {
		if strings.HasPrefix(entry, "NUMEN_HOST_PORT=") {
			t.Errorf("environment %v names a host that is not serving", vars)
		}
	}
	if len(reported) != 1 || !strings.Contains(reported[0].Error(), "md.numen.a") {
		t.Errorf("reported %v", reported)
	}
}

// The host endpoint binds the loopback interface and rejects any address that
// would bind more.
func TestHost_EndpointBindsLoopbackOnly(t *testing.T) {
	host := NewHost(testsupport.GetMachine(), "desktop.editor")
	t.Cleanup(func() {
		if err := host.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})
	for _, address := range []string{":0", "0.0.0.0:0", "[::]:0", "localhost:0", "192.168.1.5:0", "127.0.0.1", ""} {
		endpoint, err := hostendpoint.Open(t.Context(), address, host)
		if err == nil {
			if closeErr := endpoint.Close(t.Context()); closeErr != nil {
				t.Error(closeErr)
			}
			t.Errorf("address %q was bound", address)
		}
	}
	t.Cleanup(func() {
		if err := host.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})
	if host.hostAddress != "127.0.0.1:0" {
		t.Errorf("a host listens on %q by default", host.hostAddress)
	}
	endpoint, err := hostendpoint.Open(t.Context(), host.hostAddress, host)
	if err != nil {
		t.Fatal(err)
	}
	if err := endpoint.Close(t.Context()); err != nil {
		t.Error(err)
	}
}

func TestHostProcessLimitsConfig(t *testing.T) {
	h := NewHost(testsupport.GetMachine(), "test")
	if h.processLimits != process.DefaultLimits() {
		t.Fatalf("NewHost holds %v, want the default limits", h.processLimits)
	}

	customLimits := process.Limits{CPULimitSeconds: 60}
	h.SetProcessLimits(customLimits)

	h.mu.RLock()
	limits := h.processLimits
	h.mu.RUnlock()

	if limits != customLimits {
		t.Fatalf("got %v, want %v", limits, customLimits)
	}
}
