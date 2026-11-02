package service

import (
	"context"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cleanup"
	"net/http"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestServerLifecycleAndToolCall(t *testing.T) {
	t.Parallel()

	hostClient := &fakeToolClient{
		tools: []*tools.ToolInfo{
			{
				Name:        "note_get",
				Title:       "Get Note",
				Description: "Gets a note by path",
				InputSchema: []byte(`{"type":"object","properties":{"path":{"type":"string"}}}`),
				Effect:      tools.ReadOnly,
			},
		},
		callResp: &tools.CallResponse{
			Result: &tools.CallResult{
				Text: "note text for test",
			},
		},
	}

	cfg := Config{
		Addr:  "127.0.0.1:0",
		Token: "test-token-123",
	}

	srv, err := NewServer(cfg, hostClient)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	if err := srv.Start(); err != nil {
		t.Fatalf("srv.Start failed: %v", err)
	}
	defer func() {
		if err := srv.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()

	sseURL := srv.GetSSEURL()
	if sseURL == "" {
		t.Fatal("expected non-empty SSE URL")
	}

	assertUnauthorized(t, sseURL)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	session := connectSSEClient(ctx, t, sseURL, "test-token-123")
	callNoteGetTool(ctx, t, session)
	assertDynamicDiscovery(ctx, t, session, hostClient)

	cleanup.Close(t, session)

	if err := srv.Close(t.Context()); err != nil {
		t.Fatalf("srv.Close failed: %v", err)
	}
}

func assertUnauthorized(t *testing.T, url string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http.Get failed: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Error(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

func callNoteGetTool(ctx context.Context, t *testing.T, session *sdk.ClientSession) {
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

func assertDynamicDiscovery(ctx context.Context, t *testing.T, session *sdk.ClientSession, hostClient *fakeToolClient) {
	t.Helper()

	hostClient.mu.Lock()
	hostClient.tools = append(hostClient.tools, &tools.ToolInfo{
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

func connectSSEClient(ctx context.Context, t *testing.T, sseURL, token string) *sdk.ClientSession {
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
