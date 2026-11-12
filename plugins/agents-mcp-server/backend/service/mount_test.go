package service

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func getServerPort(t *testing.T, rawURL string) string {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	_, port, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		t.Fatal(err)
	}
	return port
}

func TestMountAndConfigure(t *testing.T) {
	t.Setenv("NUMEN_HOST_PORT", "")
	t.Setenv("NUMEN_PLUGIN_TOKEN", "")

	host := &standInHost{}
	mux := http.NewServeMux()
	mux.Handle(tools.NewServiceHandler(host))
	served := httptest.NewServer(mux)
	t.Cleanup(served.Close)
	hostPort := getServerPort(t, served.URL)

	s := server.New()
	if err := Mount(s); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	_, err := s.Lifecycle().ApplyConfig(t.Context(), map[string]string{
		"token":              "configured-token",
		"NUMEN_HOST_PORT":    hostPort,
		"NUMEN_PLUGIN_TOKEN": "plugin-token",
	})
	if err != nil {
		t.Fatalf("configure failed: %v", err)
	}

	host.mu.Lock()
	syncCalls := len(host.headers)
	host.mu.Unlock()
	if syncCalls == 0 {
		t.Fatal("expected SyncTools to query host during ApplyConfig")
	}

	if err := s.Close(); err != nil {
		t.Errorf("close failed: %v", err)
	}
}

func TestMountFromEnv(t *testing.T) {
	host := &standInHost{}
	mux := http.NewServeMux()
	mux.Handle(tools.NewServiceHandler(host))
	served := httptest.NewServer(mux)
	t.Cleanup(served.Close)
	hostPort := getServerPort(t, served.URL)

	t.Setenv("NUMEN_HOST_PORT", hostPort)
	t.Setenv("NUMEN_PLUGIN_TOKEN", "plugin-token")

	s := server.New()
	if err := Mount(s); err != nil {
		t.Fatalf("Mount with env host failed: %v", err)
	}

	// ApplyConfig with an unreachable host port keeps the initial client created from env.
	_, err := s.Lifecycle().ApplyConfig(t.Context(), map[string]string{
		"NUMEN_HOST_PORT": "1",
	})
	if err != nil {
		t.Fatalf("configure failed: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	host.mu.Lock()
	syncCalls := len(host.headers)
	host.mu.Unlock()
	if syncCalls < 2 {
		t.Fatalf("expected at least 2 host sync calls (initial + configure), got %d", syncCalls)
	}
}

func TestNewHandler(t *testing.T) {
	t.Parallel()

	handler, err := NewHandler(map[string]string{
		"NUMEN_PLUGIN_TOKEN": "secret-plugin-token",
	})
	if err != nil {
		t.Fatalf("NewHandler failed: %v", err)
	}
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	paths := []string{
		"/mcp",
		"/mcp/sub",
		"/md.numen.agents.mcp.server/sub",
	}
	for _, path := range paths {
		rec := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("path %s: expected 401, got %d", path, rec.Code)
		}
	}
}

func TestStreamableHTTPEndpoint(t *testing.T) {
	t.Parallel()

	hostClient := &fakeToolClient{
		tools: []*tools.ToolInfo{
			{
				Name:        "stream_tool",
				Title:       "Stream Tool",
				Description: "Tool tested over Streamable HTTP",
				InputSchema: []byte(`{"type":"object"}`),
			},
		},
		callResp: &tools.CallResponse{
			Result: &tools.CallResult{
				Text: "streamable response",
			},
		},
	}

	cfg := Config{Token: "stream-token"}

	srv, err := NewServer(cfg, hostClient)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	serveOnLoopback(t, srv)
	defer func() {
		if err := srv.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()

	httpURL := srv.GetHTTPURL()
	if httpURL == "" {
		t.Fatal("expected non-empty HTTP URL")
	}

	authClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.Header.Set("Authorization", "Bearer stream-token")
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	clientTransport := &sdk.StreamableClientTransport{
		Endpoint:             httpURL,
		HTTPClient:           authClient,
		DisableStandaloneSSE: true,
	}

	mcpClient := sdk.NewClient(&sdk.Implementation{Name: "stream-client", Version: "1.0.0"}, nil)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("mcpClient.Connect over Streamable HTTP failed: %v", err)
	}
	defer closeAndCheck(t, session)

	verifyStreamableToolCall(ctx, t, session)
}

func verifyStreamableToolCall(ctx context.Context, t *testing.T, session *sdk.ClientSession) {
	t.Helper()

	listResult, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if len(listResult.Tools) != 1 || listResult.Tools[0].Name != "stream_tool" {
		t.Fatalf("unexpected tools: %+v", listResult.Tools)
	}

	callResult, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name: "stream_tool",
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if callResult.IsError || len(callResult.Content) == 0 {
		t.Fatal("expected successful tool call with content")
	}
	text, ok := callResult.Content[0].(*sdk.TextContent)
	if !ok || text.Text != "streamable response" {
		t.Fatalf("unexpected content: %v", callResult.Content[0])
	}
}
