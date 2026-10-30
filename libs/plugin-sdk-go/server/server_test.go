package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
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
		WithPort(0),
		WithToken(testToken),
		WithStdout(&stdout),
		WithStdin(nil),
		WithConfigure(func(_ context.Context, vals map[string]string) error {
			configureMu.Lock()
			configuredValues = vals
			configureMu.Unlock()
			return nil
		}),
	)

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

	ctx := context.Background()

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
		WithPort(0),
		WithToken(testToken),
		WithStdout(io.Discard),
		WithStdin(nil),
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
		<-serveErrCh
	}()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	ctx := context.Background()

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
		WithPort(0),
		WithStdout(io.Discard),
		WithStdin(stdinReader),
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
	_ = stdinWriter.Close()

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
		WithPort(0),
		WithToken("cancel-token"),
		WithStdout(io.Discard),
		WithStdin(nil),
	)

	if _, err := s.Listen(); err != nil {
		t.Fatalf("Listen failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
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
	life := NewLifecycleServer(ch, WithLifecycleUpdateConfig(func(m map[string]string) {
		updated = m
	}))

	ctx := context.Background()

	// Ping
	pingResp, err := life.Ping(ctx, connect.NewRequest(&v1.PingRequest{}))
	if err != nil || pingResp.Msg.GetStatus() != "ok" {
		t.Fatalf("Ping failed: %v, %v", pingResp, err)
	}

	// Configure
	confResp, err := life.Configure(ctx, connect.NewRequest(&v1.ConfigureRequest{
		Values: map[string]string{"foo": "bar"},
	}))
	if err != nil || !confResp.Msg.GetSuccess() {
		t.Fatalf("Configure failed: %v, %v", confResp, err)
	}
	if updated["foo"] != "bar" {
		t.Fatalf("updated[foo] = %q, want bar", updated["foo"])
	}

	// Shutdown
	shutResp, err := life.Shutdown(ctx, connect.NewRequest(&v1.ShutdownRequest{}))
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
		WithPort(0),
		WithToken("timeout-token"),
		WithStdout(io.Discard),
		WithStdin(nil),
		WithShutdownTimeout(10*time.Millisecond),
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

	ctx, cancel := context.WithCancel(context.Background())
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- s.ServeContext(ctx)
	}()

	reqDone := make(chan struct{})
	go func() {
		defer close(reqDone)
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/block", port))
		if err == nil && resp != nil {
			_ = resp.Body.Close()
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
		if !strings.Contains(err.Error(), "shutdown:") {
			t.Fatalf("expected error wrapping shutdown:, got %v", err)
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
	s := New(WithPort(0), WithToken(testToken), WithStdout(io.Discard), WithStdin(nil))
	port, err := s.Listen()
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	serveErrCh := make(chan error, 1)
	go func() { serveErrCh <- s.ServeContext(ctx) }()
	t.Cleanup(func() {
		cancel()
		<-serveErrCh
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
		resp, err := client.Ping(context.Background(), connect.NewRequest(&v1.PingRequest{}))
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
	s := New(WithPort(0), WithToken("closer-token"), WithStdout(io.Discard), WithStdin(nil))
	closed := 0
	s.OnClose(func() { closed++ })
	if _, err := s.Listen(); err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
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
	var exitCode int
	origExit := exitProcess
	origOutput := exitOutput
	t.Cleanup(func() {
		exitProcess = origExit
		exitOutput = origOutput
	})
	exitProcess = func(code int) {
		exitCode = code
	}
	exitOutput = &errBuf

	Run(func(*Server) error {
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
	var exitCode int
	origExit := exitProcess
	origOutput := exitOutput
	t.Cleanup(func() {
		exitProcess = origExit
		exitOutput = origOutput
	})
	exitProcess = func(code int) {
		exitCode = code
	}
	exitOutput = &errBuf

	Run(func(*Server) error {
		return nil
	}, WithPort(-1), WithStdout(io.Discard), WithStdin(nil))

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(errBuf.String(), "server error:") {
		t.Fatalf("stderr = %q, want 'server error:'", errBuf.String())
	}
}

// Run executes cleanly when the server terminates without an error.
func TestRunServesAndTerminatesCleanly(t *testing.T) {
	origExit := exitProcess
	t.Cleanup(func() { exitProcess = origExit })
	exitProcess = func(code int) {
		t.Fatalf("unexpected exit with code %d", code)
	}

	Run(func(*Server) error {
		return nil
	}, WithPort(0), WithStdout(io.Discard), WithStdin(strings.NewReader("clean-token\n")), WithSignals())
}

// A server reads its auth token from stdin when no token is configured.
func TestServerReadsTokenFromStdin(t *testing.T) {
	stdinReader, stdinWriter := io.Pipe()
	token := "stdin-secret-token"
	s := New(
		WithPort(0),
		WithStdout(io.Discard),
		WithStdin(stdinReader),
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
		_ = stdinWriter.Close()
		s.Lifecycle().TriggerShutdown()
		<-serveErrCh
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

// A server refuses to start when the auth token is empty.
func TestServerRejectsEmptyToken(t *testing.T) {
	s := New(
		WithPort(0),
		WithStdout(io.Discard),
		WithStdin(nil),
	)
	if err := s.Serve(); err == nil || !strings.Contains(err.Error(), "empty auth token") {
		t.Fatalf("Serve with nil stdin = %v, want empty auth token", err)
	}

	s2 := New(
		WithPort(0),
		WithStdout(io.Discard),
		WithStdin(strings.NewReader("   \n")),
	)
	if err := s2.Serve(); err == nil || !strings.Contains(err.Error(), "empty auth token") {
		t.Fatalf("Serve with whitespace stdin = %v, want empty auth token", err)
	}
}
