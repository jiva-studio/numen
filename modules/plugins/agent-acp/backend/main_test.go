package main

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func TestLifecycleServer(t *testing.T) {
	ch := make(chan struct{})
	srv := server.NewLifecycleServer(ch)

	ctx := context.Background()
	pingResp, err := srv.Ping(ctx, connect.NewRequest(&numenv1.PingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if pingResp.Msg.GetStatus() != "ok" {
		t.Errorf("expected ok, got %s", pingResp.Msg.GetStatus())
	}

	shutResp, err := srv.Shutdown(ctx, connect.NewRequest(&numenv1.ShutdownRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !shutResp.Msg.GetSuccess() {
		t.Error("expected shutdown success")
	}
}

func TestAgentServerToolsAndPrompt(t *testing.T) {
	srv := &agentServer{
		runner: &AgentRunner{Registry: NewRegistry()},
	}

	ctx := context.Background()

	// RegisterTool
	_, err := srv.RegisterTool(ctx, connect.NewRequest(&numenv1.RegisterToolRequest{
		Tool: &numenv1.AgentToolDefinition{
			Name:        "test_tool",
			Description: "A test tool",
		},
	}))
	if err != nil {
		t.Fatal(err)
	}

	// ExtendSystemPrompt
	extResp, err := srv.ExtendSystemPrompt(ctx, connect.NewRequest(&numenv1.ExtendSystemPromptRequest{
		Extension: "Always be polite.",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if extResp.Msg.GetFullPrompt() != "Always be polite." {
		t.Errorf("expected prompt 'Always be polite.', got %q", extResp.Msg.GetFullPrompt())
	}

	// GetAgentTools
	toolsResp, err := srv.GetAgentTools(ctx, connect.NewRequest(&numenv1.GetAgentToolsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(toolsResp.Msg.GetTools()) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(toolsResp.Msg.GetTools()))
	}
	if toolsResp.Msg.GetTools()[0].GetName() != "test_tool" {
		t.Errorf("expected tool test_tool, got %s", toolsResp.Msg.GetTools()[0].GetName())
	}
	if toolsResp.Msg.GetSystemPromptExtension() != "Always be polite." {
		t.Errorf("expected prompt extension 'Always be polite.', got %q", toolsResp.Msg.GetSystemPromptExtension())
	}
}
