package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"connectrpc.com/connect"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

type fakeToolClient struct {
	mu       sync.Mutex
	tools    []*numenv1.PluginTool
	lastCall *numenv1.CallToolRequest
	callResp *numenv1.CallToolResponse
	callErr  error
	listErr  error
}

func (f *fakeToolClient) ListTools(_ context.Context, _ *connect.Request[numenv1.ListToolsRequest]) (*connect.Response[numenv1.ListToolsResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	return connect.NewResponse(&numenv1.ListToolsResponse{Tools: f.tools}), nil
}

func (f *fakeToolClient) CallTool(_ context.Context, req *connect.Request[numenv1.CallToolRequest]) (*connect.Response[numenv1.CallToolResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastCall = req.Msg
	if f.callErr != nil {
		return nil, f.callErr
	}
	if f.callResp != nil {
		return connect.NewResponse(f.callResp), nil
	}
	return connect.NewResponse(&numenv1.CallToolResponse{
		Result: &numenv1.ToolResult{
			Text: "default answer",
		},
	}), nil
}

func TestToolAggregatorSyncTools(t *testing.T) {
	t.Parallel()

	client := &fakeToolClient{
		tools: []*numenv1.PluginTool{
			{
				Name:        "note_read",
				Title:       "Read Note",
				Description: "Reads note contents",
				InputSchema: []byte(`{"type":"object","properties":{"path":{"type":"string"}}}`),
				Effect:      numenv1.ToolEffect_TOOL_EFFECT_READS,
			},
			{
				Name:        "note_write",
				Title:       "Write Note",
				Description: "Writes note contents",
				InputSchema: []byte(`{"type":"object"}`),
				Effect:      numenv1.ToolEffect_TOOL_EFFECT_WRITES,
			},
		},
	}

	mcpServer := sdk.NewServer(&sdk.Implementation{Name: "test"}, nil)
	agg := NewToolAggregator(client, mcpServer)

	if err := agg.SyncTools(context.Background()); err != nil {
		t.Fatalf("SyncTools failed: %v", err)
	}

	tools := agg.GetRegisteredTools()
	if len(tools) != 2 || tools[0] != "note_read" || tools[1] != "note_write" {
		t.Fatalf("unexpected registered tools: %v", tools)
	}

	// Update host tools: remove note_write, add note_search
	client.mu.Lock()
	client.tools = []*numenv1.PluginTool{
		{
			Name:        "note_read",
			InputSchema: []byte(`{"type":"object"}`),
		},
		{
			Name:        "note_search",
			InputSchema: []byte(`{"type":"object"}`),
		},
	}
	client.mu.Unlock()

	if err := agg.SyncTools(context.Background()); err != nil {
		t.Fatalf("second SyncTools failed: %v", err)
	}

	updated := agg.GetRegisteredTools()
	if len(updated) != 2 || updated[0] != "note_read" || updated[1] != "note_search" {
		t.Fatalf("unexpected updated tools: %v", updated)
	}
}

func TestToolAggregatorExecuteSuccess(t *testing.T) {
	t.Parallel()

	client := &fakeToolClient{
		callResp: &numenv1.CallToolResponse{
			Result: &numenv1.ToolResult{
				Text:       "note content here",
				Structured: []byte(`{"found":true}`),
			},
		},
	}
	mcpServer := sdk.NewServer(&sdk.Implementation{Name: "test"}, nil)
	agg := NewToolAggregator(client, mcpServer)
	agg.SetContext(map[string]string{"vault": "test-vault"})

	result, err := agg.ExecuteTool(context.Background(), "note_read", json.RawMessage(`{"path":"a.md"}`))
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
	if client.lastCall.GetContext()["vault"] != "test-vault" {
		t.Fatalf("expected context 'vault' to be passed, got: %v", client.lastCall.GetContext())
	}
}

func TestToolAggregatorExecuteRefusal(t *testing.T) {
	t.Parallel()

	client := &fakeToolClient{
		callResp: &numenv1.CallToolResponse{
			Result: &numenv1.ToolResult{
				Text:      "permission denied to note",
				IsRefusal: true,
			},
		},
	}
	mcpServer := sdk.NewServer(&sdk.Implementation{Name: "test"}, nil)
	agg := NewToolAggregator(client, mcpServer)

	result, err := agg.ExecuteTool(context.Background(), "note_read", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError true for refusal")
	}
}

func TestToolAggregatorExecuteToolErrors(t *testing.T) {
	t.Parallel()

	client := &fakeToolClient{
		callResp: &numenv1.CallToolResponse{
			Error: numenv1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL,
		},
	}
	mcpServer := sdk.NewServer(&sdk.Implementation{Name: "test"}, nil)
	agg := NewToolAggregator(client, mcpServer)

	result, err := agg.ExecuteTool(context.Background(), "unknown_tool", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError true for unknown tool")
	}

	client.callResp = &numenv1.CallToolResponse{
		Error: numenv1.ToolErrorCode_TOOL_ERROR_CODE_BAD_ARGUMENTS,
	}
	result, err = agg.ExecuteTool(context.Background(), "note_read", []byte(`invalid-json`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError true for bad arguments")
	}
}

func TestToolAggregatorExecuteTransportFailure(t *testing.T) {
	t.Parallel()

	client := &fakeToolClient{
		callErr: errors.New("connection reset"),
	}
	mcpServer := sdk.NewServer(&sdk.Implementation{Name: "test"}, nil)
	agg := NewToolAggregator(client, mcpServer)

	_, err := agg.ExecuteTool(context.Background(), "note_read", nil)
	if err == nil {
		t.Fatal("expected error on transport failure")
	}
}

func TestSanitizeInputSchema(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []byte
		validate func(t *testing.T, raw json.RawMessage)
	}{
		{
			name:  "nil input",
			input: nil,
			validate: func(t *testing.T, raw json.RawMessage) {
				if string(raw) != `{"type":"object"}` {
					t.Fatalf("got %s", string(raw))
				}
			},
		},
		{
			name:  "invalid json",
			input: []byte(`{invalid`),
			validate: func(t *testing.T, raw json.RawMessage) {
				if string(raw) != `{"type":"object"}` {
					t.Fatalf("got %s", string(raw))
				}
			},
		},
		{
			name:  "missing type",
			input: []byte(`{"properties":{"q":{"type":"string"}}}`),
			validate: func(t *testing.T, raw json.RawMessage) {
				var parsed map[string]any
				if err := json.Unmarshal(raw, &parsed); err != nil {
					t.Fatal(err)
				}
				if parsed["type"] != "object" {
					t.Fatalf("expected type object, got: %v", parsed["type"])
				}
			},
		},
		{
			name:  "already object",
			input: []byte(`{"type":"object","properties":{"a":{"type":"number"}}}`),
			validate: func(t *testing.T, raw json.RawMessage) {
				if string(raw) != `{"type":"object","properties":{"a":{"type":"number"}}}` {
					t.Fatalf("got %s", string(raw))
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := sanitizeInputSchema(tc.input)
			tc.validate(t, result)
		})
	}
}
