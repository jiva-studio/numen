package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

type fakeToolClient struct {
	mu       sync.Mutex
	tools    []*tools.ToolInfo
	lastCall *tools.Call
	callResp *tools.CallResponse
	callErr  error
	listErr  error
}

func (f *fakeToolClient) ListToolInfos(_ context.Context) ([]*tools.ToolInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.tools, nil
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

func TestToolAggregatorSyncTools(t *testing.T) {
	t.Parallel()

	client := &fakeToolClient{
		tools: []*tools.ToolInfo{
			{
				Name:        "note_read",
				Title:       "Read Note",
				Description: "Reads note contents",
				InputSchema: []byte(`{"type":"object","properties":{"path":{"type":"string"}}}`),
				Effect:      tools.ReadOnly,
			},
			{
				Name:        "note_write",
				Title:       "Write Note",
				Description: "Writes note contents",
				InputSchema: []byte(`{"type":"object"}`),
				Effect:      tools.ReadWrite,
			},
		},
	}

	mcpServer := sdk.NewServer(&sdk.Implementation{Name: "test"}, nil)
	agg := NewToolAggregator(client, mcpServer)

	if err := agg.SyncTools(t.Context()); err != nil {
		t.Fatalf("SyncTools failed: %v", err)
	}

	names := agg.GetRegisteredTools()
	if len(names) != 2 || names[0] != "note_read" || names[1] != "note_write" {
		t.Fatalf("unexpected registered tools: %v", names)
	}

	// Update host tools: remove note_write, add note_search
	client.mu.Lock()
	client.tools = []*tools.ToolInfo{
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

	if err := agg.SyncTools(t.Context()); err != nil {
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
		callResp: &tools.CallResponse{
			Result: &tools.CallResult{
				Text:       "note content here",
				Structured: []byte(`{"found":true}`),
			},
		},
	}
	mcpServer := sdk.NewServer(&sdk.Implementation{Name: "test"}, nil)
	agg := NewToolAggregator(client, mcpServer)
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

func TestToolAggregatorExecuteMarksDeniedResultAsError(t *testing.T) {
	t.Parallel()

	client := &fakeToolClient{
		callResp: &tools.CallResponse{
			Result: &tools.CallResult{
				Text:      "permission denied to note",
				IsRefusal: true,
			},
		},
	}
	mcpServer := sdk.NewServer(&sdk.Implementation{Name: "test"}, nil)
	agg := NewToolAggregator(client, mcpServer)

	result, err := agg.ExecuteTool(t.Context(), "note_read", nil)
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
		callResp: &tools.CallResponse{
			Error: tools.ErrorCodeUnknownTool,
		},
	}
	mcpServer := sdk.NewServer(&sdk.Implementation{Name: "test"}, nil)
	agg := NewToolAggregator(client, mcpServer)

	result, err := agg.ExecuteTool(t.Context(), "unknown_tool", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected IsError true for unknown tool")
	}

	client.callResp = &tools.CallResponse{
		Error: tools.ErrorCodeBadArguments,
	}
	result, err = agg.ExecuteTool(t.Context(), "note_read", []byte(`invalid-json`))
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

	_, err := agg.ExecuteTool(t.Context(), "note_read", nil)
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
