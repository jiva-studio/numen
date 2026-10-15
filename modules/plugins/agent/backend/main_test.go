package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

func TestAgentServerRPC(t *testing.T) {
	agentSvc := &agentServer{
		systemPrompt: "You are an assistant.",
		tools: []*numenv1.AgentToolDefinition{
			{Name: "initial_tool", Description: "Initial tool description"},
		},
	}

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(agentSvc))

	server := httptest.NewServer(mux)
	defer server.Close()

	client := numenv1connect.NewAgentServiceClient(http.DefaultClient, server.URL)
	ctx := context.Background()

	// Test GetAgentState
	stateResp, err := client.GetAgentState(ctx, connect.NewRequest(&numenv1.GetAgentStateRequest{}))
	if err != nil {
		t.Fatalf("GetAgentState failed: %v", err)
	}
	if stateResp.Msg.GetUnreachable() != "" {
		t.Errorf("expected empty unreachable, got %q", stateResp.Msg.GetUnreachable())
	}

	// Test RegisterTool
	regResp, err := client.RegisterTool(ctx, connect.NewRequest(&numenv1.RegisterToolRequest{
		Tool: &numenv1.AgentToolDefinition{
			Name:        "new_tool",
			Description: "New tool description",
		},
	}))
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}
	if !regResp.Msg.GetSuccess() {
		t.Errorf("expected success true")
	}

	// Test ExtendSystemPrompt
	extResp, err := client.ExtendSystemPrompt(ctx, connect.NewRequest(&numenv1.ExtendSystemPromptRequest{
		Extension: " Extra rules.",
	}))
	if err != nil {
		t.Fatalf("ExtendSystemPrompt failed: %v", err)
	}
	if extResp.Msg.GetFullPrompt() != "You are an assistant. Extra rules." {
		t.Errorf("unexpected full prompt: %q", extResp.Msg.GetFullPrompt())
	}

	// Test GetAgentTools
	toolsResp, err := client.GetAgentTools(ctx, connect.NewRequest(&numenv1.GetAgentToolsRequest{}))
	if err != nil {
		t.Fatalf("GetAgentTools failed: %v", err)
	}
	if len(toolsResp.Msg.GetTools()) != 2 {
		t.Errorf("expected 2 tools, got %d", len(toolsResp.Msg.GetTools()))
	}
	if toolsResp.Msg.GetSystemPromptExtension() != "You are an assistant. Extra rules." {
		t.Errorf("unexpected prompt extension: %q", toolsResp.Msg.GetSystemPromptExtension())
	}

	// Test AskAgent stream
	stream, err := client.AskAgent(ctx, connect.NewRequest(&numenv1.AskAgentRequest{
		Prompt: "Hello assistant",
	}))
	if err != nil {
		t.Fatalf("AskAgent failed: %v", err)
	}

	var hasModelStart, hasSaid, hasStop bool
	for stream.Receive() {
		msg := stream.Msg()
		switch msg.GetStep().(type) {
		case *numenv1.AskAgentResponse_ModelStarted:
			hasModelStart = true
		case *numenv1.AskAgentResponse_Said:
			hasSaid = true
		case *numenv1.AskAgentResponse_Stopped:
			hasStop = true
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}
	if !hasModelStart {
		t.Errorf("expected ModelStarted step")
	}
	if !hasSaid {
		t.Errorf("expected Said step")
	}
	if !hasStop {
		t.Errorf("expected Stopped step")
	}

	// Test FinishConversation
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

func TestLifecycleServer(t *testing.T) {
	shutdownCh := make(chan struct{})
	life := &lifecycleServer{shutdownTrigger: shutdownCh}

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(life))

	server := httptest.NewServer(mux)
	defer server.Close()

	client := numenv1connect.NewPluginLifecycleServiceClient(http.DefaultClient, server.URL)
	ctx := context.Background()

	pingResp, err := client.Ping(ctx, connect.NewRequest(&numenv1.PingRequest{}))
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if pingResp.Msg.GetStatus() != "ok" {
		t.Errorf("unexpected status: %s", pingResp.Msg.GetStatus())
	}
}
