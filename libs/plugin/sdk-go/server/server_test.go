package server

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

type bearerAuthInterceptor struct {
	token string
}

func (b *bearerAuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if b.token != "" {
			req.Header().Set("Authorization", "Bearer "+b.token)
		}
		return next(ctx, req)
	}
}

func (b *bearerAuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (b *bearerAuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func TestServerLifecycleRPC(t *testing.T) {
	var stdout safeBuffer
	testToken := "secret-token-123"

	var configuredValues map[string]string
	var configureMu sync.Mutex

	s := New(
		withPort(0),
		withToken(testToken),
		withStdout(&stdout),
		withStdin(nil),
	)
	s.SetConfigure(func(_ context.Context, vals map[string]string) error {
		configureMu.Lock()
		configuredValues = vals
		configureMu.Unlock()
		return nil
	})

	port, err := s.Listen()
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	if port <= 0 {
		t.Fatalf("expected positive port, got %d", port)
	}

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- s.Serve()
	}()

	client := numenv1connect.NewPluginLifecycleServiceClient(
		http.DefaultClient,
		fmt.Sprintf("http://127.0.0.1:%d", port),
		connect.WithInterceptors(&bearerAuthInterceptor{token: testToken}),
	)

	ctx := t.Context()

	// 1. Ping
	pingResp, err := client.Ping(ctx, connect.NewRequest(&v1.PingRequest{}))
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if pingResp.Msg.GetStatus() != "ok" {
		t.Errorf("status = %q, want 'ok'", pingResp.Msg.GetStatus())
	}
	if got := pingResp.Msg.GetProtocol(); got != ProtocolVersion {
		t.Errorf("protocol = %d, want %d", got, ProtocolVersion)
	}

	// 2. Configure
	confResp, err := client.Configure(ctx, connect.NewRequest(&v1.ConfigureRequest{
		Values: map[string]string{"theme": "dark", "zoom": "120"},
	}))
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}
	if !confResp.Msg.GetSuccess() {
		t.Errorf("Configure success = false, want true")
	}

	configureMu.Lock()
	if configuredValues["theme"] != "dark" || configuredValues["zoom"] != "120" {
		t.Errorf("configuredValues = %v, want theme=dark zoom=120", configuredValues)
	}
	configureMu.Unlock()

	// Verify GetConfig and AllConfigs
	val, ok := s.Lifecycle().GetConfig("theme")
	if !ok || val != "dark" {
		t.Errorf("GetConfig(theme) = (%q, %v), want (dark, true)", val, ok)
	}
	all := s.Lifecycle().AllConfigs()
	if len(all) != 2 || all["zoom"] != "120" {
		t.Errorf("AllConfigs = %v, want 2 items", all)
	}

	// 3. Verify handshake output
	handshake := stdout.String()
	expectedHandshake := fmt.Sprintf("NUMEN_READY port=%d\n", port)
	if !strings.Contains(handshake, expectedHandshake) {
		t.Errorf("handshake output = %q, want containing %q", handshake, expectedHandshake)
	}

	// 4. Shutdown
	shutResp, err := client.Shutdown(ctx, connect.NewRequest(&v1.ShutdownRequest{}))
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
	if !shutResp.Msg.GetSuccess() {
		t.Errorf("Shutdown success = false, want true")
	}

	select {
	case err := <-serveErrCh:
		if err != nil {
			t.Fatalf("Serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not exit within timeout")
	}
}

func TestAuthInterceptor(t *testing.T) {
	testToken := "valid-auth-token"
	s := New(
		withPort(0),
		withToken(testToken),
		withStdout(io.Discard),
		withStdin(nil),
	)

	port, err := s.Listen()
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- s.Serve()
	}()
	defer func() {
		s.Lifecycle().TriggerShutdown()
		waitServe(t, serveErrCh)
	}()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	ctx := t.Context()

	// Case 1: missing token -> unauthenticated
	unauthClient := numenv1connect.NewPluginLifecycleServiceClient(http.DefaultClient, baseURL)
	_, err = unauthClient.Ping(ctx, connect.NewRequest(&v1.PingRequest{}))
	if err == nil || connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", err)
	}

	// Case 2: wrong token -> unauthenticated
	wrongClient := numenv1connect.NewPluginLifecycleServiceClient(
		http.DefaultClient,
		baseURL,
		connect.WithInterceptors(&bearerAuthInterceptor{token: "wrong-token"}),
	)
	_, err = wrongClient.Ping(ctx, connect.NewRequest(&v1.PingRequest{}))
	if err == nil || connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", err)
	}

	// Case 3: valid token -> ok
	validClient := numenv1connect.NewPluginLifecycleServiceClient(
		http.DefaultClient,
		baseURL,
		connect.WithInterceptors(&bearerAuthInterceptor{token: testToken}),
	)
	resp, err := validClient.Ping(ctx, connect.NewRequest(&v1.PingRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetStatus() != "ok" {
		t.Errorf("status = %q, want 'ok'", resp.Msg.GetStatus())
	}
}

func TestServerStdinEOFShutdown(t *testing.T) {
	stdinReader, stdinWriter := io.Pipe()

	s := New(
		withPort(0),
		withStdout(io.Discard),
		withStdin(stdinReader),
	)

	if _, err := s.Listen(); err != nil {
		t.Fatalf("Listen failed: %v", err)
	}

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- s.Serve()
	}()

	if _, err := io.WriteString(stdinWriter, "stdin-token\n"); err != nil {
		t.Fatalf("write token to stdin: %v", err)
	}

	// Closing writer sends EOF to stdin reader
	if err := stdinWriter.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-serveErrCh:
		if err != nil {
			t.Fatalf("Serve returned error on stdin EOF: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not exit on stdin EOF")
	}
}

func TestServerContextCancellation(t *testing.T) {
	s := New(
		withPort(0),
		withToken("cancel-token"),
		withStdout(io.Discard),
		withStdin(nil),
	)

	if _, err := s.Listen(); err != nil {
		t.Fatalf("Listen failed: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- s.ServeContext(ctx)
	}()

	cancel()

	select {
	case err := <-serveErrCh:
		if err != nil {
			t.Fatalf("ServeContext returned error on cancel: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ServeContext did not exit on cancellation")
	}
}

func TestLifecycleUnitMethods(t *testing.T) {
	ch := make(chan struct{})
	var updated map[string]string
	life := NewLifecycleServer(ch)
	life.SetUpdateConfig(func(m map[string]string) {
		updated = m
	})

	ctx := t.Context()
	handler := lifecycleHandler{state: life}

	// Ping
	pingResp, err := handler.Ping(ctx, connect.NewRequest(&v1.PingRequest{}))
	if err != nil || pingResp.Msg.GetStatus() != "ok" {
		t.Fatalf("Ping failed: %v, %v", pingResp, err)
	}

	// Configure
	confResp, err := handler.Configure(ctx, connect.NewRequest(&v1.ConfigureRequest{
		Values: map[string]string{"foo": "bar"},
	}))
	if err != nil || !confResp.Msg.GetSuccess() {
		t.Fatalf("Configure failed: %v, %v", confResp, err)
	}
	if updated["foo"] != "bar" {
		t.Fatalf("updated[foo] = %q, want bar", updated["foo"])
	}

	// Shutdown
	shutResp, err := handler.Shutdown(ctx, connect.NewRequest(&v1.ShutdownRequest{}))
	if err != nil || !shutResp.Msg.GetSuccess() {
		t.Fatalf("Shutdown failed: %v, %v", shutResp, err)
	}

	select {
	case <-life.ShutdownTrigger():
	default:
		t.Fatal("expected ShutdownTrigger to be closed immediately")
	}
}

func TestServerShutdownTimeoutError(t *testing.T) {
	reqStarted := make(chan struct{})
	blockReq := make(chan struct{})
	var closeOnce sync.Once
	defer closeOnce.Do(func() {
		close(blockReq)
	})

	s := New(
		withPort(0),
		withToken("timeout-token"),
		withStdout(io.Discard),
		withStdin(nil),
		withShutdownTimeout(10*time.Millisecond),
	)

	s.Register("/block", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(reqStarted)
		<-blockReq
		w.WriteHeader(http.StatusOK)
	}))

	port, err := s.Listen()
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- s.ServeContext(ctx)
	}()

	reqDone := make(chan struct{})
	go func() {
		defer close(reqDone)
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/block", port), nil)
		if err != nil {
			t.Error(err)
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			if closeErr := resp.Body.Close(); closeErr != nil {
				t.Error(closeErr)
			}
		}
	}()

	select {
	case <-reqStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("in-flight request did not reach handler")
	}

	cancel()

	select {
	case err := <-serveErrCh:
		if err == nil {
			t.Fatal("expected shutdown error, got nil")
		}
		if !strings.Contains(err.Error(), "shutdown:") || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected error wrapping shutdown: and DeadlineExceeded, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ServeContext did not return within timeout")
	}

	closeOnce.Do(func() {
		close(blockReq)
	})
	select {
	case <-reqDone:
	case <-time.After(2 * time.Second):
		t.Fatal("in-flight request did not complete")
	}
}

// protoRecorder records the HTTP version of every response it carries.
type protoRecorder struct {
	next   http.RoundTripper
	mu     sync.Mutex
	majors []int
}

func (p *protoRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := p.next.RoundTrip(req)
	if resp != nil {
		p.mu.Lock()
		p.majors = append(p.majors, resp.ProtoMajor)
		p.mu.Unlock()
	}
	return resp, err
}

func TestServerAnswersHTTP1AndUnencryptedHTTP2(t *testing.T) {
	const testToken = "proto-token"
	s := New(withPort(0), withToken(testToken), withStdout(io.Discard), withStdin(nil))
	port, err := s.Listen()
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	serveErrCh := make(chan error, 1)
	go func() { serveErrCh <- s.ServeContext(ctx) }()
	t.Cleanup(func() {
		cancel()
		waitServe(t, serveErrCh)
	})

	for _, want := range []int{1, 2} {
		var protocols http.Protocols
		protocols.SetHTTP1(want == 1)
		protocols.SetUnencryptedHTTP2(want == 2)
		recorder := &protoRecorder{next: &http.Transport{Protocols: &protocols}}
		client := numenv1connect.NewPluginLifecycleServiceClient(
			&http.Client{Transport: recorder},
			fmt.Sprintf("http://127.0.0.1:%d", port),
			connect.WithInterceptors(&bearerAuthInterceptor{token: testToken}),
		)
		resp, err := client.Ping(t.Context(), connect.NewRequest(&v1.PingRequest{}))
		if err != nil {
			t.Fatalf("Ping over HTTP/%d failed: %v", want, err)
		}
		if resp.Msg.GetStatus() != "ok" {
			t.Errorf("Ping over HTTP/%d status = %q, want ok", want, resp.Msg.GetStatus())
		}
		recorder.mu.Lock()
		if len(recorder.majors) != 1 || recorder.majors[0] != want {
			t.Errorf("response versions = %v, want [%d]", recorder.majors, want)
		}
		recorder.mu.Unlock()
	}
}

// A server that serves on a port runs what OnClose added when Serve returns.
func TestServerRunsItsClosersWhenServeReturns(t *testing.T) {
	s := New(withPort(0), withToken("closer-token"), withStdout(io.Discard), withStdin(nil))
	closed := 0
	s.OnClose(func() error { closed++; return nil })
	if _, err := s.Listen(); err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if err := s.ServeContext(ctx); err != nil {
		t.Fatalf("ServeContext: %v", err)
	}
	if closed != 1 {
		t.Fatalf("closers ran %d times, want once", closed)
	}
}

// Run reports an error and terminates with exit code 1 when the mount function fails.
func TestRunExitsOnMountError(t *testing.T) {
	var errBuf bytes.Buffer
	exitCode := run(&errBuf, func(*Server) error {
		return errors.New("mount failed")
	})

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(errBuf.String(), "server error: mount failed\n") {
		t.Fatalf("stderr = %q, want 'server error: mount failed\\n'", errBuf.String())
	}
}

// Run reports an error and terminates with exit code 1 when serving fails.
func TestRunExitsOnServeError(t *testing.T) {
	var errBuf bytes.Buffer
	exitCode := run(&errBuf, func(*Server) error {
		return nil
	}, withPort(-1), withStdout(io.Discard), withStdin(nil))

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(errBuf.String(), "server error:") {
		t.Fatalf("stderr = %q, want 'server error:'", errBuf.String())
	}
}

// Run executes cleanly when the server terminates without an error.
func TestRunServesAndTerminatesCleanly(t *testing.T) {
	var errBuf bytes.Buffer
	code := run(&errBuf, func(*Server) error {
		return nil
	}, withPort(0), withStdout(io.Discard), withStdin(strings.NewReader("clean-token\n")), withSignals())
	if code != 0 {
		t.Fatalf("exit code %d (%s), want 0", code, errBuf.String())
	}
}

// A server reads its auth token from stdin when no token is configured.
func TestServerReadsTokenFromStdin(t *testing.T) {
	stdinReader, stdinWriter := io.Pipe()
	token := "stdin-secret-token"
	s := New(
		withPort(0),
		withStdout(io.Discard),
		withStdin(stdinReader),
	)

	port, err := s.Listen()
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- s.Serve()
	}()

	if _, err := io.WriteString(stdinWriter, token+"\n"); err != nil {
		t.Fatalf("write token: %v", err)
	}
	defer func() {
		if err := stdinWriter.Close(); err != nil {
			t.Error(err)
		}
		s.Lifecycle().TriggerShutdown()
		waitServe(t, serveErrCh)
	}()

	client := numenv1connect.NewPluginLifecycleServiceClient(
		http.DefaultClient,
		fmt.Sprintf("http://127.0.0.1:%d", port),
		connect.WithInterceptors(&bearerAuthInterceptor{token: token}),
	)
	resp, err := client.Ping(t.Context(), connect.NewRequest(&v1.PingRequest{}))
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if resp.Msg.GetStatus() != "ok" {
		t.Errorf("status = %q, want 'ok'", resp.Msg.GetStatus())
	}
	if got := os.Getenv(tokenVar); got != token {
		t.Errorf("os.Getenv(%s) = %q, want %q", tokenVar, got, token)
	}
}

// A server fails to start when the auth token is empty.
func TestServerRejectsEmptyToken(t *testing.T) {
	s := New(
		withPort(0),
		withStdout(io.Discard),
		withStdin(nil),
	)
	if err := s.Serve(); err == nil || !strings.Contains(err.Error(), "empty auth token") {
		t.Fatalf("Serve with nil stdin = %v, want empty auth token", err)
	}

	s2 := New(
		withPort(0),
		withStdout(io.Discard),
		withStdin(strings.NewReader("   \n")),
	)
	if err := s2.Serve(); err == nil || !strings.Contains(err.Error(), "empty auth token") {
		t.Fatalf("Serve with whitespace stdin = %v, want empty auth token", err)
	}
}

func waitServe(t *testing.T, serveErrCh <-chan error) {
	t.Helper()
	if err := <-serveErrCh; err != nil {
		t.Errorf("Serve returned error: %v", err)
	}
}

type errWriter struct{}

var errWriteFailed = errors.New("write failed")

func (errWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}

type errReader struct{}

var errReadFailed = errors.New("read failed")

func (errReader) Read([]byte) (int, error) {
	return 0, errReadFailed
}

type multiStepStdinReader struct {
	step int
	mode string
}

func (r *multiStepStdinReader) Read(p []byte) (int, error) {
	r.step++
	if r.mode == "zero-nil" {
		return 0, nil
	}
	if r.step == 1 {
		p[0] = 'a'
		return 1, nil
	}
	return 1, io.EOF
}

type closeErrListener struct {
	net.Listener
	closeErr error
	acceptFn func() (net.Conn, error)
}

func (l *closeErrListener) Close() error {
	if err := l.Listener.Close(); err != nil && l.closeErr == nil {
		return err
	}
	return l.closeErr
}

func (l *closeErrListener) Accept() (net.Conn, error) {
	if l.acceptFn != nil {
		return l.acceptFn()
	}
	return l.Listener.Accept()
}

type stepTokenReader struct {
	step int
}

func (r *stepTokenReader) Read(p []byte) (int, error) {
	r.step++
	switch r.step {
	case 1:
		p[0] = 'x'
		return 1, nil
	default:
		return 0, io.EOF
	}
}

type bufSizeReader struct {
	sizes chan int
}

func (r *bufSizeReader) Read(p []byte) (int, error) {
	r.sizes <- len(p)
	return 0, io.EOF
}

func TestAuthInterceptor_EdgeCases(t *testing.T) {
	emptyAuth := NewAuthInterceptor("")
	if err := emptyAuth.checkToken(http.Header{"Authorization": {"Bearer "}}); err == nil {
		t.Fatal("expected error for empty configured token even with Bearer header")
	}
	if !emptyAuth.mu.TryLock() {
		t.Fatal("RUnlock was not called")
	}
	emptyAuth.mu.Unlock()

	auth := NewAuthInterceptor("secret")
	if err := auth.checkToken(http.Header{"Authorization": {"secret"}}); err == nil {
		t.Fatal("expected error when Authorization header lacks Bearer prefix")
	}
}

func TestLifecycleServer_EdgeCases(t *testing.T) {
	ls := NewLifecycleServer(
		make(chan struct{}),
		WithLifecycleShutdown(func() error {
			return errors.New("shutdown hook error")
		}),
	)
	if _, err := ls.ApplyConfig(t.Context(), map[string]string{"k1": "v1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ls.ApplyConfig(t.Context(), map[string]string{"k2": "v2"}); err != nil {
		t.Fatal(err)
	}
	if v, ok := ls.GetConfig("k1"); !ok || v != "v1" {
		t.Fatalf("expected k1=v1 to be preserved across configure calls, got %q %v", v, ok)
	}
	if ok, err := ls.RequestShutdown(t.Context()); ok || err == nil || err.Error() != "shutdown hook error" {
		t.Fatalf("expected shutdown hook error, got ok=%v err=%v", ok, err)
	}

	var zeroLS LifecycleServer
	if err := zeroLS.configureWith(t.Context(), map[string]string{"z": "1"}); err != nil {
		t.Fatal(err)
	}
	if v, ok := zeroLS.GetConfig("z"); !ok || v != "1" {
		t.Fatalf("expected z=1 on zero LifecycleServer, got %q %v", v, ok)
	}
}

func TestParseDefaultFlagsAndOptions(t *testing.T) {
	origCmd := flag.CommandLine
	origArgs := os.Args
	t.Cleanup(func() {
		flag.CommandLine = origCmd
		os.Args = origArgs
	})

	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	os.Args = []string{"test", "-port=6123"}
	if got := parseDefaultFlags(); got != 6123 {
		t.Fatalf("parseDefaultFlags with -port=6123 = %d, want 6123", got)
	}

	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	os.Args = []string{"test"}
	t.Setenv("NUMEN_PLUGIN_PORT", "")
	if got := parseDefaultFlags(); got != 0 {
		t.Fatalf("parseDefaultFlags with default -port = %d, want 0", got)
	}

	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	os.Args = []string{"test"}
	if err := flag.CommandLine.Parse(nil); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NUMEN_PLUGIN_PORT", "7234")
	if got := parseDefaultFlags(); got != 7234 {
		t.Fatalf("parseDefaultFlags with NUMEN_PLUGIN_PORT=7234 = %d, want 7234", got)
	}

	t.Setenv("NUMEN_PLUGIN_PORT", "not-a-number")
	if got := readEnvPort(); got != 0 {
		t.Fatalf("readEnvPort invalid = %d, want 0", got)
	}
	t.Setenv("NUMEN_PLUGIN_PORT", "99999999999999999999")
	if got := readEnvPort(); got != 0 {
		t.Fatalf("readEnvPort out of range = %d, want 0", got)
	}

	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	flag.CommandLine.String("port", "", "port as text")
	os.Args = []string{"test", "-port=99999999999999999999"}
	if got := readFlagPort(); got != 0 {
		t.Fatalf("readFlagPort out of range = %d, want 0", got)
	}

	s := New(withPort(8123), withSignals(os.Interrupt))
	if s.Port() != 8123 {
		t.Fatalf("s.Port() = %d, want 8123", s.Port())
	}
	if len(s.signals) != 1 || s.signals[0] != os.Interrupt {
		t.Fatalf("s.signals = %v, want [interrupt]", s.signals)
	}
	if s.shutdownTimeout != 2*time.Second || s.readHeaderTimeout != 5*time.Second {
		t.Fatalf("default timeouts = %v, %v; want 2s, 5s", s.shutdownTimeout, s.readHeaderTimeout)
	}

	_, err := NewLinkedHandler(map[string]string{tokenVar: "tok"}, func(ls *Server) error {
		if ls.Port() != 0 {
			t.Errorf("linked server Port() = %d, want 0", ls.Port())
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if code := run(errWriter{}, func(*Server) error { return errors.New("boom") }); code != 2 {
		t.Fatalf("run with failing errOut = %d, want 2", code)
	}
}

func TestServerListenServeAndHelpers(t *testing.T) {
	sBad := New(withPort(-1), withToken("tok"), withStdout(nil), withStdin(nil))
	var opErr *net.OpError
	if port, err := sBad.Listen(); err == nil || port != 0 || !errors.As(err, &opErr) {
		t.Fatalf("expected wrapped *net.OpError on port -1, got port=%d err=%v", port, err)
	}
	if err := sBad.Serve(); err == nil || !strings.Contains(err.Error(), "listen on") {
		t.Fatalf("expected Serve listen error, got %v", err)
	}

	s := New(withPort(0), withToken("tok"), withStdout(nil), withStdin(nil), withSignals())
	p1, err := s.Listen()
	if err != nil || p1 <= 0 || s.Port() != p1 {
		t.Fatalf("first Listen = %d, %v, Port()=%d", p1, err, s.Port())
	}
	p2, err := s.Listen()
	if err != nil || p2 != p1 {
		t.Fatalf("second Listen = %d, %v; want %d, nil", p2, err, p1)
	}
	if err := s.closeListener(); err != nil {
		t.Fatalf("first closeListener: %v", err)
	}
	if err := s.closeListener(); err != nil {
		t.Fatalf("second closeListener (already closed): %v", err)
	}

	// closeListener returning non-ErrClosed error
	var lc net.ListenConfig
	ln, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	wantCloseErr := errors.New("custom close err")
	sCloseErr := New(withPort(0), withToken("tok"), withStdout(nil), withStdin(nil), withSignals())
	sCloseErr.listener = &closeErrListener{Listener: ln, closeErr: wantCloseErr}
	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatal("expected *net.TCPAddr")
	}
	sCloseErr.actualPort = tcpAddr.Port
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := sCloseErr.ServeContext(ctx); !errors.Is(err, wantCloseErr) {
		t.Fatalf("expected wrapped custom close err from ServeContext, got %v", err)
	}

	// ServeContext with OnClose returning error
	sCloserErr := New(withPort(0), withToken("tok"), withStdout(nil), withStdin(nil), withSignals())
	sCloserErr.OnClose(func() error { return errors.New("closer boom") })
	if err := sCloserErr.ServeContext(ctx); err == nil || !strings.Contains(err.Error(), "closer boom") {
		t.Fatalf("expected closer boom from ServeContext, got %v", err)
	}

	// ServeContext with failing stdout in announceReady
	sStdoutErr := New(withPort(0), withToken("tok"), withStdout(errWriter{}), withStdin(nil), withSignals())
	if err := sStdoutErr.Serve(); !errors.Is(err, errWriteFailed) {
		t.Fatalf("expected wrapped write ready line error, got %v", err)
	}

	// ServeContext where serveHTTP fails immediately
	lnFail, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	wantAcceptErr := errors.New("accept boom")
	sServeFail := New(withPort(0), withToken("tok"), withStdout(nil), withStdin(nil), withSignals())
	sServeFail.listener = &closeErrListener{
		Listener: lnFail,
		acceptFn: func() (net.Conn, error) {
			return nil, wantAcceptErr
		},
	}
	tcpAddrFail, ok := lnFail.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatal("expected *net.TCPAddr")
	}
	sServeFail.actualPort = tcpAddrFail.Port
	if err := sServeFail.Serve(); !errors.Is(err, wantAcceptErr) {
		t.Fatalf("expected wrapped serve: error when accept fails, got %v", err)
	}

	// serveHTTP closes errs channel on ErrServerClosed
	lnClose, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if err := lnClose.Close(); err != nil {
		t.Fatal(err)
	}
	errsCh := make(chan error, 1)
	httpSrv := &http.Server{ReadHeaderTimeout: time.Minute}
	if err := httpSrv.Close(); err != nil {
		t.Fatal(err)
	}
	serveHTTP(httpSrv, lnClose, errsCh)
	if err, open := <-errsCh; open {
		t.Fatalf("expected serveHTTP to close errs channel, got %v", err)
	}

	// authenticate edge cases
	sTokenStdin := New(withToken("preset"), withStdin(strings.NewReader("stdin-tok\n")))
	if err := sTokenStdin.authenticate(); err != nil || sTokenStdin.token != "preset" {
		t.Fatalf("expected preset token to be kept, got %q err=%v", sTokenStdin.token, err)
	}

	sReadErr := New(withStdin(errReader{}))
	if err := sReadErr.authenticate(); !errors.Is(err, errReadFailed) {
		t.Fatalf("expected wrapped read auth token error, got %v", err)
	}

	sBadEnv := New(withToken("bad\x00token"), withStdin(nil))
	if err := sBadEnv.authenticate(); !errors.Is(err, syscall.EINVAL) {
		t.Fatalf("expected wrapped EINVAL setenv error for NUL byte token, got %v", err)
	}

	// watchStdin edge cases
	sNilStdin := New(withStdin(nil))
	if sNilStdin.watchStdin() == nil {
		t.Fatal("expected non-nil channel from watchStdin when stdin is nil")
	}
	bsr := &bufSizeReader{sizes: make(chan int, 1)}
	sBufStdin := New(withStdin(bsr))
	eofCh := sBufStdin.watchStdin()
	if gotSize := <-bsr.sizes; gotSize != 1 {
		t.Fatalf("watchStdin read buffer size = %d, want 1", gotSize)
	}
	<-eofCh

	sZeroNil := New(withStdin(&multiStepStdinReader{mode: "zero-nil"}))
	<-sZeroNil.watchStdin()

	stepReader := &multiStepStdinReader{mode: "step-eof"}
	sStepEOF := New(withStdin(stepReader))
	<-sStepEOF.watchStdin()
	if stepReader.step != 2 {
		t.Fatalf("watchStdin step = %d, want 2", stepReader.step)
	}

	// readTokenLine edge cases
	tok, err := readTokenLine(&stepTokenReader{})
	if err != nil || tok != "x" {
		t.Fatalf("readTokenLine stepTokenReader = %q, %v; want 'x', nil", tok, err)
	}
	if _, err := readTokenLine(errReader{}); err == nil {
		t.Fatal("expected error from readTokenLine with errReader")
	}
}
