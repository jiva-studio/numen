package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMountAndConfigure(t *testing.T) {
	t.Parallel()

	s := server.New(server.WithPort(0))

	if err := Mount(s); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	_, err := s.Lifecycle().Configure(context.Background(), connect.NewRequest(&numenv1.ConfigureRequest{
		Values: map[string]string{
			"token":           "configured-token",
			"NUMEN_HOST_PORT": "12345",
		},
	}))
	if err != nil {
		t.Fatalf("configure failed: %v", err)
	}

	// Cleanup
	s.Close()
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

	// Request to /sse without token returns 401
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sse", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestStreamableHTTPEndpoint(t *testing.T) {
	t.Parallel()

	hostClient := &fakeToolClient{
		tools: []*numenv1.PluginTool{
			{
				Name:        "stream_tool",
				Title:       "Stream Tool",
				Description: "Tool tested over Streamable HTTP",
				InputSchema: []byte(`{"type":"object"}`),
			},
		},
		callResp: &numenv1.CallToolResponse{
			Result: &numenv1.ToolResult{
				Text: "streamable response",
			},
		},
	}

	cfg := Config{
		Addr:  "127.0.0.1:0",
		Token: "stream-token",
	}

	srv, err := NewServer(cfg, hostClient)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	if err := srv.Start(); err != nil {
		t.Fatalf("srv.Start failed: %v", err)
	}
	defer func() {
		_ = srv.Close(context.Background())
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("mcpClient.Connect over Streamable HTTP failed: %v", err)
	}
	defer session.Close()

	verifyStreamableToolCall(t, session, ctx)
}

func verifyStreamableToolCall(t *testing.T, session *sdk.ClientSession, ctx context.Context) {
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
