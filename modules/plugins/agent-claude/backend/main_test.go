package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

func TestLifecycleServer(t *testing.T) {
	shutdownCh := make(chan struct{})
	agentSvc := &agentServer{}
	life := server.NewLifecycleServer(shutdownCh, server.WithLifecycleConfigure(func(ctx context.Context, values map[string]string) error {
		agentSvc.applyConfig(ctx, values)
		return nil
	}))

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(life))

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	client := numenv1connect.NewPluginLifecycleServiceClient(http.DefaultClient, testServer.URL)
	ctx := context.Background()

	pingResp, err := client.Ping(ctx, connect.NewRequest(&numenv1.PingRequest{}))
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if pingResp.Msg.GetStatus() != "ok" {
		t.Errorf("unexpected status: %s", pingResp.Msg.GetStatus())
	}

	_, err = client.Configure(ctx, connect.NewRequest(&numenv1.ConfigureRequest{
		Values: map[string]string{
			"model": "claude-3-5-sonnet-latest",
		},
	}))
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}
	if agentSvc.modelName != "claude-3-5-sonnet-latest" {
		t.Errorf("expected modelName claude-3-5-sonnet-latest, got %q", agentSvc.modelName)
	}
}

func TestFetchAnthropicModels(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"id": "claude-3-5-sonnet", "display_name": "Claude 3.5 Sonnet"}
			]
		}`))
	}))
	defer mockServer.Close()

	ctx := context.Background()
	models, err := fetchAnthropicModels(ctx, "test-key", mockServer.URL)
	if err != nil {
		t.Fatalf("fetchAnthropicModels failed: %v", err)
	}
	if len(models) != 1 || models[0].Name != "claude-3-5-sonnet" {
		t.Errorf("unexpected models: %+v", models)
	}
}

func TestAgentServiceRPC(t *testing.T) {
	agentSvc := &agentServer{
		systemPrompt: "Base prompt.",
		tools: []*numenv1.AgentToolDefinition{
			{Name: "search_tool", Description: "Search notes"},
		},
		command: []string{"echo"},
	}

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(agentSvc))

	server := httptest.NewServer(mux)
	defer server.Close()

	client := numenv1connect.NewAgentServiceClient(http.DefaultClient, server.URL)
	ctx := context.Background()

	stateResp, err := client.GetAgentState(ctx, connect.NewRequest(&numenv1.GetAgentStateRequest{}))
	if err != nil {
		t.Fatalf("GetAgentState failed: %v", err)
	}
	if stateResp.Msg.GetUnreachable() != "" {
		t.Errorf("expected reachable, got %q", stateResp.Msg.GetUnreachable())
	}

	regResp, err := client.RegisterTool(ctx, connect.NewRequest(&numenv1.RegisterToolRequest{
		Tool: &numenv1.AgentToolDefinition{Name: "new_tool", Description: "New tool"},
	}))
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}
	if !regResp.Msg.GetSuccess() {
		t.Errorf("expected success true")
	}

	extResp, err := client.ExtendSystemPrompt(ctx, connect.NewRequest(&numenv1.ExtendSystemPromptRequest{
		Extension: " Extra instructions.",
	}))
	if err != nil {
		t.Fatalf("ExtendSystemPrompt failed: %v", err)
	}
	if extResp.Msg.GetFullPrompt() != "Base prompt. Extra instructions." {
		t.Errorf("unexpected full prompt: %q", extResp.Msg.GetFullPrompt())
	}

	toolsResp, err := client.GetAgentTools(ctx, connect.NewRequest(&numenv1.GetAgentToolsRequest{}))
	if err != nil {
		t.Fatalf("GetAgentTools failed: %v", err)
	}
	if len(toolsResp.Msg.GetTools()) != 2 {
		t.Errorf("expected 2 tools, got %d", len(toolsResp.Msg.GetTools()))
	}

	finResp, err := client.FinishConversation(ctx, connect.NewRequest(&numenv1.FinishConversationRequest{
		Conversation: "conv-1",
	}))
	if err != nil {
		t.Fatalf("FinishConversation failed: %v", err)
	}
	if finResp == nil {
		t.Errorf("expected non-nil response")
	}
}

func TestAskAgentWithMockScript(t *testing.T) {
	scriptContent := "#!/bin/sh\n" +
		"cat << 'JSONEOF'\n" +
		"{\"type\":\"system\",\"subtype\":\"status\",\"status\":\"requesting\"}\n" +
		"{\"type\":\"stream_event\",\"event\":{\"type\":\"content_block_start\",\"content_block\":{\"type\":\"tool_use\",\"id\":\"call_1\",\"name\":\"mcp__numen__note_search\"}}}\n" +
		"{\"type\":\"stream_event\",\"event\":{\"type\":\"content_block_delta\",\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"query\\\":\\\"algebra\\\",\\\"path\\\":\\\"notes/math.md\\\"}\"}}}\n" +
		"{\"type\":\"stream_event\",\"event\":{\"type\":\"content_block_stop\"}}\n" +
		"{\"type\":\"user\",\"message\":{\"content\":[{\"type\":\"tool_result\"}]}}\n" +
		"{\"type\":\"stream_event\",\"event\":{\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Algebra notes found.\"}}}\n" +
		"{\"type\":\"result\",\"is_error\":false}\n" +
		"JSONEOF\n"

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "mock-claude.sh")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("write mock script: %v", err)
	}

	agentSvc := &agentServer{
		command: []string{scriptPath},
	}

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(agentSvc))

	server := httptest.NewServer(mux)
	defer server.Close()

	client := numenv1connect.NewAgentServiceClient(http.DefaultClient, server.URL)
	ctx := context.Background()

	stream, err := client.AskAgent(ctx, connect.NewRequest(&numenv1.AskAgentRequest{
		Prompt: "Find algebra notes",
	}))
	if err != nil {
		t.Fatalf("AskAgent failed: %v", err)
	}

	var steps []string
	var toolPaths []string
	for stream.Receive() {
		msg := stream.Msg()
		switch s := msg.GetStep().(type) {
		case *numenv1.AskAgentResponse_ModelStarted:
			steps = append(steps, "model_started")
		case *numenv1.AskAgentResponse_ToolCall:
			steps = append(steps, "tool_call:"+s.ToolCall.GetTool())
			toolPaths = append(toolPaths, s.ToolCall.GetPath())
		case *numenv1.AskAgentResponse_ToolFinished:
			steps = append(steps, "tool_finished")
		case *numenv1.AskAgentResponse_Said:
			steps = append(steps, "said:"+s.Said)
		case *numenv1.AskAgentResponse_Stopped:
			steps = append(steps, "stopped")
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}

	expected := []string{
		"model_started",
		"tool_call:note_search",
		"tool_call:note_search",
		"tool_finished",
		"said:Algebra notes found.",
		"stopped",
	}

	if len(steps) != len(expected) {
		t.Fatalf("expected %d steps, got %d: %v", len(expected), len(steps), steps)
	}
	for i, exp := range expected {
		if steps[i] != exp {
			t.Errorf("step %d: got %s, want %s", i, steps[i], exp)
		}
	}

	if len(toolPaths) != 2 || toolPaths[1] != "notes/math.md" {
		t.Errorf("expected final tool path 'notes/math.md', got %v", toolPaths)
	}
}

func TestExtractCallDetails(t *testing.T) {
	about, path, span := extractCallDetails(`{"query":"algebra","path":"notes/math.md","start":10,"length":5}`)
	if path != "notes/math.md" {
		t.Errorf("expected path notes/math.md, got %q", path)
	}
	if about != "algebra" {
		t.Errorf("expected about algebra, got %q", about)
	}
	if span == nil || span.From != 10 || span.To != 15 {
		t.Errorf("unexpected span: %+v", span)
	}

	aboutFile, pathFile, _ := extractCallDetails(`{"file":"notes/file.md"}`)
	if pathFile != "notes/file.md" {
		t.Errorf("expected path notes/file.md, got %q", pathFile)
	}
	if aboutFile != "notes/file.md" {
		t.Errorf("expected about notes/file.md, got %q", aboutFile)
	}
}
