package exposetools

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

type fakeToolClient struct {
	mu       sync.Mutex
	tools    []*tools.ToolInfo
	lastCall *tools.Call
	callResp *tools.CallResponse
	callErr  error
	listErr  error
}

func (f *fakeToolClient) ListTools(_ context.Context) (tools.Listing, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return tools.Listing{}, f.listErr
	}
	return tools.Listing{Tools: f.tools}, nil
}

func (f *fakeToolClient) InvokeTool(_ context.Context, call tools.Call) (*tools.CallResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastCall = &call
	if f.callErr != nil {
		return nil, f.callErr
	}
	if f.callResp != nil {
		return f.callResp, nil
	}
	return &tools.CallResponse{
		Result: &tools.CallResult{
			Text: "default answer",
		},
	}, nil
}

func TestAggregatorSyncTools(t *testing.T) {
	t.Parallel()

	client := &fakeToolClient{
		tools: []*tools.ToolInfo{
			{
				Name: "",
			},
			{
				Name:         "note_read",
				Title:        "Read Note",
				Description:  "Reads note contents",
				InputSchema:  []byte(`{"type":"object","properties":{"path":{"type":"string"}}}`),
				OutputSchema: []byte(`{"type":"object","properties":{"text":{"type":"string"}}}`),
				Effect:       tools.ReadOnly,
			},
			{
				Name:        "note_write",
				Title:       "Write Note",
				Description: "Writes note contents",
				InputSchema: []byte(`{"type":"object"}`),
				Effect:      tools.ReadWrite,
			},
			{
				Name:        "note_delete",
				Title:       "Delete Note",
				Description: "Deletes a note",
				InputSchema: []byte(`{"type":"object"}`),
				Effect:      tools.ReadWrite,
			},
		},
	}

	if err := NewAggregator(client, nil).SyncTools(t.Context()); err != nil {
		t.Fatalf("SyncTools with nil server failed: %v", err)
	}

	mcpServer := sdk.NewServer(&sdk.Implementation{Name: "test"}, nil)
	agg := NewAggregator(client, mcpServer)

	if err := agg.SyncTools(t.Context()); err != nil {
		t.Fatalf("SyncTools failed: %v", err)
	}

	names := agg.GetRegisteredTools()
	if len(names) != 3 || names[0] != "note_delete" || names[1] != "note_read" || names[2] != "note_write" {
		t.Fatalf("unexpected registered tools: %v", names)
	}

	serverSide, clientSide := sdk.NewInMemoryTransports()
	if _, err := mcpServer.Connect(t.Context(), serverSide, nil); err != nil {
		t.Fatal(err)
	}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "test-client"}, nil).Connect(t.Context(), clientSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { closeAndCheck(t, session) })

	listed, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Tools) != 3 {
		t.Fatalf("expected 3 tools in MCP server, got %d", len(listed.Tools))
	}
	for _, tool := range listed.Tools {
		switch tool.Name {
		case "note_read":
			if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
				t.Fatalf("expected ReadOnlyHint true for note_read, got %+v", tool.Annotations)
			}
			if tool.OutputSchema == nil {
				t.Fatal("expected non-nil OutputSchema for note_read")
			}
		case "note_write", "note_delete":
			if tool.Annotations == nil || tool.Annotations.ReadOnlyHint {
				t.Fatalf("expected ReadOnlyHint false for %s, got %+v", tool.Name, tool.Annotations)
			}
			if tool.OutputSchema != nil {
				t.Fatalf("expected nil OutputSchema for %s, got %v", tool.Name, tool.OutputSchema)
			}
		default:
			t.Fatalf("unexpected tool in list: %s", tool.Name)
		}
	}

	// Remove 1 tool first (kills len(toRemove) > 0 -> > 1), then remove 2 tools at once (kills RANGE_BREAK).
	client.mu.Lock()
	client.tools = []*tools.ToolInfo{
		{Name: "note_read", InputSchema: []byte(`{"type":"object"}`)},
		{Name: "note_write", InputSchema: []byte(`{"type":"object"}`)},
	}
	client.mu.Unlock()

	if err := agg.SyncTools(t.Context()); err != nil {
		t.Fatalf("second SyncTools failed: %v", err)
	}
	listedAfterOneRemoval, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(listedAfterOneRemoval.Tools) != 2 {
		t.Fatalf("expected 2 tools after removing 1, got %d", len(listedAfterOneRemoval.Tools))
	}

	client.mu.Lock()
	client.tools = []*tools.ToolInfo{
		{Name: "note_search", InputSchema: []byte(`{"type":"object"}`)},
	}
	client.mu.Unlock()

	if err := agg.SyncTools(t.Context()); err != nil {
		t.Fatalf("third SyncTools failed: %v", err)
	}
	listedAfterTwoRemovals, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(listedAfterTwoRemovals.Tools) != 1 || listedAfterTwoRemovals.Tools[0].Name != "note_search" {
		t.Fatalf("unexpected tools after removing 2: %+v", listedAfterTwoRemovals.Tools)
	}

	sentinelErr := errors.New("host list error")
	client.mu.Lock()
	client.listErr = sentinelErr
	client.mu.Unlock()
	if err := agg.SyncTools(t.Context()); !errors.Is(err, sentinelErr) {
		t.Fatalf("expected wrapped sentinelErr, got %v", err)
	}
}

func TestAggregatorExecuteSuccess(t *testing.T) {
	t.Parallel()

	client := &fakeToolClient{
		callResp: &tools.CallResponse{
			Result: &tools.CallResult{
				Text:       "note content here",
				Structured: []byte(`{"found":true}`),
			},
		},
	}
	mcpServer := sdk.NewServer(&sdk.Implementation{Name: "test"}, nil)
	agg := NewAggregator(client, mcpServer)
	result, err := agg.ExecuteTool(t.Context(), "note_read", json.RawMessage(`{"path":"a.md"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected IsError false")
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}
	text, ok := result.Content[0].(*sdk.TextContent)
	if !ok || text.Text != "note content here" {
		t.Fatalf("unexpected content text: %v", result.Content[0])
	}
	rawStructured, ok := result.StructuredContent.(json.RawMessage)
	if !ok || string(rawStructured) != `{"found":true}` {
		t.Fatalf("unexpected structured content: %v", result.StructuredContent)
	}
	if client.lastCall.GetName() != "note_read" || string(client.lastCall.GetArguments()) != `{"path":"a.md"}` {
		t.Fatalf("the host was called with %v", client.lastCall)
	}
}

func TestAggregatorExecuteMarksDeniedResultAsError(t *testing.T) {
	t.Parallel()

	client := &fakeToolClient{
		callResp: &tools.CallResponse{
			Result: &tools.CallResult{
				Text:    "permission denied to note",
				IsError: true,
			},
		},
	}
	mcpServer := sdk.NewServer(&sdk.Implementation{Name: "test"}, nil)
	agg := NewAggregator(client, mcpServer)

	result, err := agg.ExecuteTool(t.Context(), "note_read", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError true for refusal")
	}
	if result.StructuredContent != nil {
		t.Fatalf("expected nil StructuredContent when empty, got %v", result.StructuredContent)
	}

	// Structured-only result with 1-byte payload falls back to structured text.
	client.callResp = &tools.CallResponse{
		Result: &tools.CallResult{
			Text:       "",
			Structured: []byte("1"),
		},
	}
	structuredOnly, err := agg.ExecuteTool(t.Context(), "note_read", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw, ok := structuredOnly.StructuredContent.(json.RawMessage); !ok || string(raw) != "1" {
		t.Fatalf("unexpected 1-byte structured content: %v", structuredOnly.StructuredContent)
	}
	if len(structuredOnly.Content) != 1 {
		t.Fatalf("expected 1 content item, got %+v", structuredOnly.Content)
	}
	if textContent, ok := structuredOnly.Content[0].(*sdk.TextContent); !ok || textContent.Text != "1" {
		t.Fatalf("expected fallback text '1', got %+v", structuredOnly.Content)
	}

	// Nil wireResult returns empty non-error result.
	client.callResp = &tools.CallResponse{Result: nil}
	emptyRes, err := agg.ExecuteTool(t.Context(), "note_read", nil)
	if err != nil || emptyRes.IsError || len(emptyRes.Content) != 0 {
		t.Fatalf("expected empty non-error result, got %+v, err=%v", emptyRes, err)
	}
}

func verifyErrorResultText(t *testing.T, res *sdk.CallToolResult, want string) {
	t.Helper()
	if !res.IsError || len(res.Content) == 0 {
		t.Fatalf("expected error result with content, got %+v", res)
	}
	textContent, ok := res.Content[0].(*sdk.TextContent)
	if !ok || textContent.Text != want {
		t.Fatalf("got error text %v, want %q", res.Content[0], want)
	}
}

func TestAggregatorExecuteToolErrors(t *testing.T) {
	t.Parallel()

	mcpServer := sdk.NewServer(&sdk.Implementation{Name: "test"}, nil)
	noClientRes, err := NewAggregator(nil, mcpServer).ExecuteTool(t.Context(), "note_read", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	verifyErrorResultText(t, noClientRes, "no host client configured")

	client := &fakeToolClient{
		callResp: &tools.CallResponse{
			Error: tools.ErrorCodeUnknownTool,
		},
	}
	agg := NewAggregator(client, mcpServer)

	result, err := agg.ExecuteTool(t.Context(), "unknown_tool", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	verifyErrorResultText(t, result, "unknown tool: unknown_tool")

	client.callResp = &tools.CallResponse{
		Error: tools.ErrorCodeBadArguments,
	}
	result, err = agg.ExecuteTool(t.Context(), "note_read", []byte(`invalid-json`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	verifyErrorResultText(t, result, "invalid tool arguments")
}

func TestAggregatorExecuteTransportFailure(t *testing.T) {
	t.Parallel()

	sentinelErr := errors.New("connection reset")
	client := &fakeToolClient{
		callErr: sentinelErr,
	}
	mcpServer := sdk.NewServer(&sdk.Implementation{Name: "test"}, nil)
	agg := NewAggregator(client, mcpServer)

	_, err := agg.ExecuteTool(t.Context(), "note_read", nil)
	if !errors.Is(err, sentinelErr) {
		t.Fatalf("expected wrapped sentinelErr, got %v", err)
	}
}

func closeAndCheck(tb testing.TB, c io.Closer) {
	tb.Helper()
	if err := c.Close(); err != nil {
		tb.Errorf("close: %v", err)
	}
}
