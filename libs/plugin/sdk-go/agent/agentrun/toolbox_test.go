package agentrun_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
)

func TestToolboxPreservesTheToolsAndExtendsThePrompt(t *testing.T) {
	var box agentrun.Toolbox
	ctx := t.Context()

	if err := box.RegisterTool(ctx, &agentrun.ToolDefinition{Name: "read_note"}); err != nil {
		t.Fatalf("registration failed: %v", err)
	}
	if err := box.RegisterTool(ctx, nil); err != nil {
		t.Fatal(err)
	}
	first, err := box.ExtendSystemPrompt(ctx, "Be polite.")
	if err != nil {
		t.Fatal(err)
	}
	second, err := box.ExtendSystemPrompt(ctx, "Be brief.")
	if err != nil {
		t.Fatal(err)
	}
	same, err := box.ExtendSystemPrompt(ctx, "")
	if err != nil {
		t.Fatal(err)
	}

	if first != "Be polite." || second != "Be polite.\nBe brief." || same != "Be polite.\nBe brief." {
		t.Errorf("the prompts are %q, %q, %q", first, second, same)
	}
	got, err := box.GetTools(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Tools) != 1 || got.Tools[0].Name != "read_note" {
		t.Errorf("the tools are %v", got.Tools)
	}
	if got.SystemPromptExtension != "Be polite.\nBe brief." {
		t.Errorf("the extension is %q", got.SystemPromptExtension)
	}
}
