package agentrun_test

import (
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func TestToolboxKeepsTheToolsAndExtendsThePrompt(t *testing.T) {
	var box agentrun.Toolbox
	ctx := t.Context()

	reg, err := box.RegisterTool(ctx, connect.NewRequest(&numenv1.RegisterToolRequest{
		Tool: &numenv1.AgentToolDefinition{Name: "read_note"},
	}))
	if err != nil || !reg.Msg.GetSuccess() {
		t.Fatalf("registration failed: %v", err)
	}
	if _, err := box.RegisterTool(ctx, connect.NewRequest(&numenv1.RegisterToolRequest{})); err != nil {
		t.Fatal(err)
	}
	first, _ := box.ExtendSystemPrompt(ctx, connect.NewRequest(&numenv1.ExtendSystemPromptRequest{Extension: "Be polite."}))
	second, _ := box.ExtendSystemPrompt(ctx, connect.NewRequest(&numenv1.ExtendSystemPromptRequest{Extension: "Be brief."}))
	same, _ := box.ExtendSystemPrompt(ctx, connect.NewRequest(&numenv1.ExtendSystemPromptRequest{}))

	if first.Msg.GetFullPrompt() != "Be polite." || second.Msg.GetFullPrompt() != "Be polite.\nBe brief." ||
		same.Msg.GetFullPrompt() != "Be polite.\nBe brief." {
		t.Errorf("the prompts are %q, %q, %q", first.Msg.GetFullPrompt(), second.Msg.GetFullPrompt(), same.Msg.GetFullPrompt())
	}
	got, err := box.GetAgentTools(ctx, connect.NewRequest(&numenv1.GetAgentToolsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if tools := got.Msg.GetTools(); len(tools) != 1 || tools[0].GetName() != "read_note" {
		t.Errorf("the tools are %v", tools)
	}
	if got.Msg.GetSystemPromptExtension() != "Be polite.\nBe brief." {
		t.Errorf("the extension is %q", got.Msg.GetSystemPromptExtension())
	}
}
