package service

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestServerLifecycleAndToolCall(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	hostClient := &fakeToolClient{
		tools: []*numenv1.PluginTool{
			{
				Name:        "note_get",
				Title:       "Get Note",
				Description: "Gets a note by path",
				InputSchema: []byte(`{"type":"object","properties":{"path":{"type":"string"}}}`),
				Effect:      numenv1.ToolEffect_TOOL_EFFECT_READS,
			},
		},
		callResp: &numenv1.CallToolResponse{
			Result: &numenv1.ToolResult{
				Text: "note text for test",
			},
		},
	}

	cfg := Config{
		Addr:      "127.0.0.1:0",
		Token:     "test-token-123",
		VaultRoot: tmpDir,
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

	// Verify discovery files exist
	agentPath := filepath.Join(tmpDir, ".numen", "agent.json")
	if _, err := os.Stat(agentPath); err != nil {
		t.Fatalf("expected discovery file at %s: %v", agentPath, err)
	}

	sseURL := srv.GetSSEURL()
	if sseURL == "" {
		t.Fatal("expected non-empty SSE URL")
	}

	assertUnauthorized(t, sseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session := connectSSEClient(t, sseURL, "test-token-123", ctx)
	callNoteGetTool(t, session, ctx)
	assertDynamicDiscovery(t, session, ctx, hostClient)

	_ = session.Close()

	if err := srv.Close(context.Background()); err != nil {
		t.Fatalf("srv.Close failed: %v", err)
	}
	if _, err := os.Stat(agentPath); !os.IsNotExist(err) {
		t.Fatal("expected discovery file to be removed after close")
	}
}

func assertUnauthorized(t *testing.T, url string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("http.Get failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

func callNoteGetTool(t *testing.T, session *sdk.ClientSession, ctx context.Context) {
	t.Helper()

	listResult, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("session.ListTools failed: %v", err)
	}
	if len(listResult.Tools) != 1 || listResult.Tools[0].Name != "note_get" {
		t.Fatalf("unexpected tools returned: %+v", listResult.Tools)
	}

	callResult, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name: "note_get",
		Arguments: map[string]any{
			"path": "hello.md",
		},
	})
	if err != nil {
		t.Fatalf("session.CallTool failed: %v", err)
	}
	if callResult.IsError || len(callResult.Content) == 0 {
		t.Fatal("expected successful tool call with content")
	}
	textContent, ok := callResult.Content[0].(*sdk.TextContent)
	if !ok || textContent.Text != "note text for test" {
		t.Fatalf("unexpected call result content: %v", callResult.Content[0])
	}
}

func assertDynamicDiscovery(t *testing.T, session *sdk.ClientSession, ctx context.Context, hostClient *fakeToolClient) {
	t.Helper()

	hostClient.mu.Lock()
	hostClient.tools = append(hostClient.tools, &numenv1.PluginTool{
		Name:        "new_vault_tool",
		Title:       "New Tool",
		Description: "Newly registered tool",
		InputSchema: []byte(`{"type":"object"}`),
	})
	hostClient.mu.Unlock()

	listResult, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("second ListTools failed: %v", err)
	}
	if len(listResult.Tools) != 2 {
		t.Fatalf("expected 2 tools after dynamic addition, got %d", len(listResult.Tools))
	}
}

func connectSSEClient(t *testing.T, sseURL, token string, ctx context.Context) *sdk.ClientSession {
	t.Helper()
	authClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.Header.Set("Authorization", "Bearer "+token)
			return http.DefaultTransport.RoundTrip(req)
		}),
	}
	clientTransport := &sdk.SSEClientTransport{
		Endpoint:   sseURL,
		HTTPClient: authClient,
	}
	mcpClient := sdk.NewClient(&sdk.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("mcpClient.Connect failed: %v", err)
	}
	return session
}
