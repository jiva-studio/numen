package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
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
	pingResp, err := client.Ping(ctx, connect.NewRequest(&numenv1.PingRequest{}))
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if pingResp.Msg.GetStatus() != "ok" {
		t.Errorf("status = %q, want 'ok'", pingResp.Msg.GetStatus())
	}

	// 2. Configure
	confResp, err := client.Configure(ctx, connect.NewRequest(&numenv1.ConfigureRequest{
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
	shutResp, err := client.Shutdown(ctx, connect.NewRequest(&numenv1.ShutdownRequest{}))
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
	_, err = unauthClient.Ping(ctx, connect.NewRequest(&numenv1.PingRequest{}))
	if err == nil || connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", err)
	}

	// Case 2: wrong token -> unauthenticated
	wrongClient := numenv1connect.NewPluginLifecycleServiceClient(
		http.DefaultClient,
		baseURL,
		connect.WithInterceptors(&bearerAuthInterceptor{token: "wrong-token"}),
	)
	_, err = wrongClient.Ping(ctx, connect.NewRequest(&numenv1.PingRequest{}))
	if err == nil || connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", err)
	}

	// Case 3: valid token -> ok
	validClient := numenv1connect.NewPluginLifecycleServiceClient(
		http.DefaultClient,
		baseURL,
		connect.WithInterceptors(&bearerAuthInterceptor{token: testToken}),
	)
	resp, err := validClient.Ping(ctx, connect.NewRequest(&numenv1.PingRequest{}))
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
		WithToken(""),
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
	pingResp, err := life.Ping(ctx, connect.NewRequest(&numenv1.PingRequest{}))
	if err != nil || pingResp.Msg.GetStatus() != "ok" {
		t.Fatalf("Ping failed: %v, %v", pingResp, err)
	}

	// Configure
	confResp, err := life.Configure(ctx, connect.NewRequest(&numenv1.ConfigureRequest{
		Values: map[string]string{"foo": "bar"},
	}))
	if err != nil || !confResp.Msg.GetSuccess() {
		t.Fatalf("Configure failed: %v, %v", confResp, err)
	}
	if updated["foo"] != "bar" {
		t.Fatalf("updated[foo] = %q, want bar", updated["foo"])
	}

	// Shutdown
	shutResp, err := life.Shutdown(ctx, connect.NewRequest(&numenv1.ShutdownRequest{}))
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
		WithToken(""),
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
