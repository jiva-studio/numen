package plugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

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
		"capabilities": ["test"],
		"targets": ["desktop.editor"],
		"permissions": ["vault:read"],
		"lifecycle": {
			"mode": "on-demand",
			"idleTimeoutSeconds": 10
		},
		"backend": {
			"type": "process",
			"entrypoint": "bin/test",
			"serviceName": "numen.v1.TestService"
		},
		"frontend": {
			"entrypoint": "dist/index.js"
		},
		"settings": []
	}`, id, id)
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}
}

func createTestPluginTree(t *testing.T, dir string) {
	t.Helper()
	pDir := filepath.Join(dir, "ocr")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	manifest := `{
		"id": "md.numen.ocr",
		"name": "OCR Plugin",
		"version": "1.0.0",
		"description": "OCR",
		"capabilities": ["recognition"],
		"targets": ["desktop.editor"],
		"activationEvents": ["onFile:*.pdf"],
		"permissions": ["vault:read"],
		"lifecycle": {
			"mode": "on-demand",
			"idleTimeoutSeconds": 1,
			"keepAliveWhileVisible": false
		},
		"backend": {
			"type": "process",
			"entrypoint": "bin/ocr",
			"serviceName": "numen.v1.OcrService"
		},
		"frontend": {
			"entrypoint": "dist/index.js",
			"styles": "dist/styles.css"
		},
		"settings": []
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}
}

func TestHostDiscoveryAndLookup(t *testing.T) {
	tmp := t.TempDir()
	createTestPluginTree(t, tmp)

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	plugins := host.GetPlugins()
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}

	p, ok := host.GetPlugin("md.numen.ocr")
	if !ok || p.Manifest.ID != "md.numen.ocr" {
		t.Fatal("expected to find md.numen.ocr")
	}

	capPlugin, ok := host.FindPluginByCapability("recognition")
	if !ok || capPlugin.Manifest.ID != "md.numen.ocr" {
		t.Fatal("expected to find plugin by capability recogniser")
	}
}

func TestHostLifecycleWithMockLauncher(t *testing.T) {
	tmp := t.TempDir()
	createTestPluginTree(t, tmp)

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, port int, _ string) (*os.Process, io.ReadCloser, error) {
		actualPort := port
		if actualPort == 0 {
			actualPort = 54321
		}
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", actualPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	ctx := t.Context()
	endpoint, err := host.EnsureActive(ctx, "md.numen.ocr")
	if err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	if endpoint.Port == 0 || endpoint.Token == "" {
		t.Fatalf("expected non-zero port and token, got %+v", endpoint)
	}

	// Calling EnsureActive again should return the existing active endpoint
	endpoint2, err := host.EnsureActive(ctx, "md.numen.ocr")
	if err != nil {
		t.Fatalf("second EnsureActive failed: %v", err)
	}
	if endpoint.Port != endpoint2.Port || endpoint.Token != endpoint2.Token {
		t.Fatalf("expected identical endpoints, got %+v and %+v", endpoint, endpoint2)
	}

	// Release twice to enter idle state
	host.Release("md.numen.ocr")
	host.Release("md.numen.ocr")

	// TriggerEvent for matching event
	activated, err := host.TriggerEvent(ctx, "onFile:test.pdf")
	if err != nil {
		t.Fatalf("TriggerEvent failed: %v", err)
	}
	if len(activated) != 1 || activated[0] != "md.numen.ocr" {
		t.Fatalf("expected md.numen.ocr to be activated, got %v", activated)
	}

	// EvictIdle test
	evicted := host.EvictIdle(ctx, 10*time.Millisecond, 50*time.Millisecond)
	// Because idleTimeoutSeconds is 1s, right now it is not evicted yet
	if len(evicted) != 0 {
		t.Fatalf("expected 0 evicted before threshold, got %v", evicted)
	}

	// Cleanup
	host.Release("md.numen.ocr")
	if err := host.TeardownAll(ctx, 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestHostEvictLoopTeardown(t *testing.T) {
	host := NewHost("desktop.editor")
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
		if !isWhitelistedEnv(k) {
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
		if isWhitelistedEnv(k) {
			t.Errorf("expected %s to be blocked from environment whitelist", k)
		}
	}
}

func TestHostEmitEventActivation(t *testing.T) {
	tmp := t.TempDir()
	createTestPluginTree(t, tmp)

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, port int, _ string) (*os.Process, io.ReadCloser, error) {
		actualPort := port
		if actualPort == 0 {
			actualPort = 54321
		}
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", actualPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	ctx := t.Context()

	// Initial state is dormant
	sup := host.supervisors["md.numen.ocr"]
	if sup == nil || sup.GetState() != StateDormant {
		t.Fatalf("expected initial state dormant, got %v", sup.GetState())
	}

	// Emit onFile event via debounced event bus
	host.EmitEvent("onFile:document.pdf")
	host.GetEventBus().Flush(ctx)

	// Plugin should now be activated
	if sup.GetState() != StateActive && sup.GetState() != StateIdle {
		t.Fatalf("expected state active or idle after EmitEvent, got %v", sup.GetState())
	}

	host.Release("md.numen.ocr")
	if err := host.TeardownAll(ctx, 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestHostDrainsStdoutAfterHandshake(t *testing.T) {
	tmp := t.TempDir()
	createTestPluginTree(t, tmp)

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe failed: %v", err)
	}

	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		return cmd.Process, pr, nil
	})

	writeDone := make(chan error, 1)
	go func() {
		defer pw.Close()
		if _, err := pw.Write([]byte("NUMEN_READY port=54321\n")); err != nil {
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
	endpoint, err := host.EnsureActive(ctx, "md.numen.ocr")
	if err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	if endpoint.Port != 54321 {
		t.Fatalf("expected port 54321, got %d", endpoint.Port)
	}

	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatalf("write to pipe failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for large stdout write to complete")
	}

	host.Release("md.numen.ocr")
	if err := host.TeardownAll(ctx, 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestHostEnsureActiveConcurrentSamePlugin(t *testing.T) {
	tmp := t.TempDir()
	createNamedTestPlugin(t, tmp, "test.concurrent")

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	var launchCount atomic.Int32
	launchStarted := make(chan struct{})
	allowHandshake := make(chan struct{})
	var once sync.Once

	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, port int, _ string) (*os.Process, io.ReadCloser, error) {
		launchCount.Add(1)
		once.Do(func() {
			close(launchStarted)
		})
		<-allowHandshake
		actualPort := port
		if actualPort == 0 {
			actualPort = 54321
		}
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", actualPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	const numGoroutines = 8
	var wg sync.WaitGroup
	endpoints := make([]Endpoint, numGoroutines)
	errs := make([]error, numGoroutines)

	wg.Add(1)
	go func() {
		defer wg.Done()
		endpoints[0], errs[0] = host.EnsureActive(t.Context(), "test.concurrent")
	}()

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
		if endpoints[i].Port != 54321 {
			t.Fatalf("goroutine %d port = %d, want 54321", i, endpoints[i].Port)
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
	createNamedTestPlugin(t, tmp, "plugin.slow")
	createNamedTestPlugin(t, tmp, "plugin.fast")

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	cmdSlow := exec.Command("sleep", "10")
	if err := cmdSlow.Start(); err != nil {
		t.Fatalf("start slow proc failed: %v", err)
	}
	defer func() {
		_ = cmdSlow.Process.Kill()
		_, _ = cmdSlow.Process.Wait()
	}()

	cmdFast := exec.Command("sleep", "10")
	if err := cmdFast.Start(); err != nil {
		t.Fatalf("start fast proc failed: %v", err)
	}
	defer func() {
		_ = cmdFast.Process.Kill()
		_, _ = cmdFast.Process.Wait()
	}()

	slowStarted := make(chan struct{})
	slowAllowHandshake := make(chan struct{})

	host.SetLauncher(func(_ context.Context, p DiscoveredPlugin, port int, _ string) (*os.Process, io.ReadCloser, error) {
		if p.Manifest.ID == "plugin.slow" {
			close(slowStarted)
			<-slowAllowHandshake
			actualPort := port
			if actualPort == 0 {
				actualPort = 11111
			}
			return cmdSlow.Process, io.NopCloser(strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", actualPort))), nil
		}
		actualPort := port
		if actualPort == 0 {
			actualPort = 22222
		}
		return cmdFast.Process, io.NopCloser(strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", actualPort))), nil
	})

	var slowWg sync.WaitGroup
	var slowEndpoint Endpoint
	var slowErr error

	slowWg.Add(1)
	go func() {
		defer slowWg.Done()
		slowEndpoint, slowErr = host.EnsureActive(t.Context(), "plugin.slow")
	}()

	<-slowStarted

	fastEndpoint, err := host.EnsureActive(t.Context(), "plugin.fast")
	if err != nil {
		t.Fatalf("EnsureActive on fast plugin failed while slow plugin was handshaking: %v", err)
	}
	if fastEndpoint.Port != 22222 {
		t.Fatalf("fast plugin expected port 22222, got %d", fastEndpoint.Port)
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
	if slowEndpoint.Port != 11111 {
		t.Fatalf("slow plugin expected port 11111, got %d", slowEndpoint.Port)
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

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	launchStarted := make(chan struct{})
	allowFailure := make(chan struct{})
	var once sync.Once
	failErr := errors.New("simulated launch failure")

	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		once.Do(func() {
			close(launchStarted)
		})
		<-allowFailure
		return nil, nil, failErr
	})

	const numGoroutines = 4
	var wg sync.WaitGroup
	errs := make([]error, numGoroutines)

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, errs[0] = host.EnsureActive(t.Context(), "test.fail")
	}()

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
	createNamedTestPlugin(t, tmp, "test.cancel")

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	launchStarted := make(chan struct{})
	allowHandshake := make(chan struct{})

	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, port int, _ string) (*os.Process, io.ReadCloser, error) {
		close(launchStarted)
		<-allowHandshake
		actualPort := port
		if actualPort == 0 {
			actualPort = 54321
		}
		return cmd.Process, io.NopCloser(strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", actualPort))), nil
	})

	var wg sync.WaitGroup
	var activeEp Endpoint
	var activeErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		activeEp, activeErr = host.EnsureActive(t.Context(), "test.cancel")
	}()

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
	if activeEp.Port != 54321 {
		t.Fatalf("expected port 54321, got %d", activeEp.Port)
	}

	host.Release("test.cancel")
	if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

type testCloser struct {
	mu     sync.Mutex
	closed bool
}

func (c *testCloser) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *testCloser) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

type testLifecycleHandler struct {
	numenv1connect.UnimplementedPluginLifecycleServiceHandler
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
	createTestPluginTree(t, tmp)

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	handler := &testLifecycleHandler{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(handler))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	tcpAddr, ok := srv.Listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected *net.TCPAddr, got %T", srv.Listener.Addr())
	}
	serverPort := tcpAddr.Port

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}()

	handler.onShutdown = func() {
		_ = cmd.Process.Kill()
	}

	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	ctx := t.Context()
	endpoint, err := host.EnsureActive(ctx, "md.numen.ocr")
	if err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	if endpoint.Port != serverPort {
		t.Fatalf("expected port %d, got %d", serverPort, endpoint.Port)
	}

	closer := &testCloser{}
	host.stdinsMu.Lock()
	host.stdins["md.numen.ocr"] = closer
	host.stdinsMu.Unlock()

	host.mu.Lock()
	p := host.plugins["md.numen.ocr"]
	p.Manifest.Lifecycle.IdleTimeoutSeconds = 0
	host.plugins["md.numen.ocr"] = p
	host.mu.Unlock()

	host.Release("md.numen.ocr")
	host.Release("md.numen.ocr")

	evicted := host.EvictIdle(ctx, 0, 500*time.Millisecond)
	if len(evicted) != 1 || evicted[0] != "md.numen.ocr" {
		t.Fatalf("expected md.numen.ocr to be evicted, got %v", evicted)
	}

	if !handler.isShutdownCalled() {
		t.Error("expected PluginLifecycleService.Shutdown RPC to be sent")
	}

	if !closer.isClosed() {
		t.Error("expected stdin closer to be closed on eviction")
	}

	host.stdinsMu.Lock()
	_, stdinExists := host.stdins["md.numen.ocr"]
	host.stdinsMu.Unlock()
	if stdinExists {
		t.Error("expected stdin pipe to be deleted from host.stdins")
	}

	sup := host.supervisors["md.numen.ocr"]
	if sup.GetState() != StateDormant {
		t.Errorf("expected StateDormant, got %v", sup.GetState())
	}
}

func TestHostEvictMemoryExceeded(t *testing.T) {
	tmp := t.TempDir()
	createTestPluginTree(t, tmp)

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	handler := &testLifecycleHandler{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(handler))
	server := httptest.NewServer(mux)
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	serverPort, err := strconv.Atoi(serverURL.Port())
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	handler.onShutdown = func() {
		_ = cmd.Process.Kill()
	}

	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	ctx := t.Context()
	_, err = host.EnsureActive(ctx, "md.numen.ocr")
	if err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}

	closer := &testCloser{}
	host.stdinsMu.Lock()
	host.stdins["md.numen.ocr"] = closer
	host.stdinsMu.Unlock()

	sup := host.supervisors["md.numen.ocr"]
	sup.SetMemoryLimit(100 * 1024 * 1024)

	// Simulate RSS below limit
	sup.SetRSSReader(func(_ int) (uint64, error) {
		return 50 * 1024 * 1024, nil
	})
	evicted := host.EvictMemoryExceeded(ctx, 500*time.Millisecond)
	if len(evicted) != 0 {
		t.Fatalf("expected no eviction when below limit, got %v", evicted)
	}

	// Simulate RSS above limit
	sup.SetRSSReader(func(_ int) (uint64, error) {
		return 200 * 1024 * 1024, nil
	})
	evicted = host.EvictMemoryExceeded(ctx, 500*time.Millisecond)
	if len(evicted) != 1 || evicted[0] != "md.numen.ocr" {
		t.Fatalf("expected md.numen.ocr to be evicted for memory, got %v", evicted)
	}

	if !handler.isShutdownCalled() {
		t.Error("expected PluginLifecycleService.Shutdown RPC to be sent")
	}

	if !closer.isClosed() {
		t.Error("expected stdin closer to be closed on eviction")
	}

	if sup.GetState() != StateDormant {
		t.Errorf("expected StateDormant, got %v", sup.GetState())
	}
}

func TestHostTeardownAllSendsRPCShutdownAndClosesStdin(t *testing.T) {
	tmp := t.TempDir()
	createTestPluginTree(t, tmp)

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	handler := &testLifecycleHandler{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(handler))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	tcpAddr, ok := srv.Listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected *net.TCPAddr, got %T", srv.Listener.Addr())
	}
	serverPort := tcpAddr.Port

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}()

	handler.onShutdown = func() {
		_ = cmd.Process.Kill()
	}

	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	ctx := t.Context()
	if _, err := host.EnsureActive(ctx, "md.numen.ocr"); err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}

	closer := &testCloser{}
	host.stdinsMu.Lock()
	host.stdins["md.numen.ocr"] = closer
	host.stdinsMu.Unlock()

	if err := host.TeardownAll(ctx, 500*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}

	if !handler.isShutdownCalled() {
		t.Error("expected PluginLifecycleService.Shutdown RPC to be sent during TeardownAll")
	}

	if !closer.isClosed() {
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
	createTestPluginTree(t, tmp)

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	handler := &testLifecycleHandler{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(handler))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	tcpAddr, ok := srv.Listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected *net.TCPAddr, got %T", srv.Listener.Addr())
	}
	serverPort := tcpAddr.Port

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}()

	rpcReceived := make(chan struct{})
	releaseShutdown := make(chan struct{})
	handler.onShutdown = func() {
		close(rpcReceived)
		<-releaseShutdown
		_ = cmd.Process.Kill()
	}

	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	ctx := t.Context()
	if _, err := host.EnsureActive(ctx, "md.numen.ocr"); err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}

	host.mu.Lock()
	p := host.plugins["md.numen.ocr"]
	p.Manifest.Lifecycle.IdleTimeoutSeconds = 0
	host.plugins["md.numen.ocr"] = p
	host.mu.Unlock()

	host.Release("md.numen.ocr")
	host.Release("md.numen.ocr")

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = host.EvictIdle(ctx, 0, 1*time.Second)
	}()

	<-rpcReceived

	if !host.mu.TryLock() {
		t.Fatal("expected host.mu to be unlocked while EvictIdle is tearing down")
	}
	host.mu.Unlock()

	close(releaseShutdown)
	<-done
}

func killProcess(t *testing.T, p *os.Process) {
	t.Helper()
	if err := p.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		t.Errorf("kill process: %v", err)
	}
}

func waitUntil(t *testing.T, what string, condition func() bool) {
	t.Helper()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	for !condition() {
		select {
		case <-timeout.C:
			t.Fatalf("timed out waiting for %s", what)
		case <-ticker.C:
		}
	}
}

func startSleepLauncher(t *testing.T, started *atomic.Int32, procs *[]*os.Process) Launcher {
	t.Helper()
	var mu sync.Mutex
	return func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		cmd := exec.Command("sleep", "10")
		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		t.Cleanup(func() { killProcess(t, cmd.Process) })
		mu.Lock()
		if procs != nil {
			*procs = append(*procs, cmd.Process)
		}
		mu.Unlock()
		started.Add(1)
		return cmd.Process, io.NopCloser(strings.NewReader("NUMEN_READY port=54321\n")), nil
	}
}

func TestACrashedPluginIsStartedAgainAfterAConsumerReleasesIt(t *testing.T) {
	tmp := t.TempDir()
	createTestPluginTree(t, tmp)
	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	var started atomic.Int32
	var procs []*os.Process
	host.SetLauncher(startSleepLauncher(t, &started, &procs))

	first, err := host.EnsureActive(t.Context(), "md.numen.ocr")
	if err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}

	host.mu.RLock()
	sup := host.supervisors["md.numen.ocr"]
	host.mu.RUnlock()
	killProcess(t, procs[0])
	waitUntil(t, "the crashed plugin to become dormant", func() bool {
		host.mu.RLock()
		defer host.mu.RUnlock()
		_, hasEndpoint := host.endpoints["md.numen.ocr"]
		return sup.GetState() == StateDormant && !hasEndpoint
	})

	host.Release("md.numen.ocr")
	if sup.GetState() != StateDormant {
		t.Fatalf("Release moved a crashed plugin to %v", sup.GetState())
	}

	second, err := host.EnsureActive(t.Context(), "md.numen.ocr")
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
	sup := NewSupervisor()
	if sup.Acquire(time.Now()) {
		t.Fatal("Acquire held a dormant supervisor")
	}
	sup.Release(time.Now())
	if sup.GetState() != StateDormant {
		t.Fatalf("Release moved a dormant supervisor to %v", sup.GetState())
	}
	sup.SetActivating()
	if sup.Acquire(time.Now()) {
		t.Fatal("Acquire held an activating supervisor")
	}
	sup.Release(time.Now())
	if sup.GetState() != StateActivating {
		t.Fatalf("Release moved an activating supervisor to %v", sup.GetState())
	}
}

func TestEnsureActiveRetiresARunningProcessWhoseEndpointIsMissing(t *testing.T) {
	tmp := t.TempDir()
	createTestPluginTree(t, tmp)
	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	var started atomic.Int32
	var procs []*os.Process
	host.SetLauncher(startSleepLauncher(t, &started, &procs))

	if _, err := host.EnsureActive(t.Context(), "md.numen.ocr"); err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	host.mu.Lock()
	delete(host.endpoints, "md.numen.ocr")
	host.mu.Unlock()

	endpoint, err := host.EnsureActive(t.Context(), "md.numen.ocr")
	if err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	if isEmptyEndpoint(endpoint) {
		t.Fatal("EnsureActive returned an empty endpoint")
	}
	if started.Load() != 2 {
		t.Fatalf("expected a replacement launch, got %d launches", started.Load())
	}
	waitUntil(t, "the unreachable process to be gone", func() bool {
		return procs[0].Signal(syscall.Signal(0)) != nil
	})
	if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestTheExitOfAnOldProcessKeepsTheEndpointOfItsReplacement(t *testing.T) {
	host := NewHost("desktop.editor")
	sup := NewSupervisor()
	replacement := Endpoint{Host: "127.0.0.1", Port: 4321, Token: "new"}

	host.mu.Lock()
	host.endpoints["p"] = replacement
	host.mu.Unlock()
	sup.SetActivating()
	sup.ActivateInProcess(time.Now())
	host.dropEndpointOfStoppedPlugin("p", sup)
	host.mu.RLock()
	kept := host.endpoints["p"]
	host.mu.RUnlock()
	if kept != replacement {
		t.Fatalf("the endpoint of a running replacement was dropped: %+v", kept)
	}

	sup.SetDormant(nil)
	host.dropEndpointOfStoppedPlugin("p", sup)
	host.mu.RLock()
	_, stillThere := host.endpoints["p"]
	host.mu.RUnlock()
	if stillThere {
		t.Fatal("the endpoint of a stopped plugin was kept")
	}
}
