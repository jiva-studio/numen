package toolset

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
)

func TestRegister_KeepsOrder(t *testing.T) {
	var set Toolset
	set.Register(&agentrun.ToolDefinition{Name: "search_tool", Description: "Search notes"})
	set.Register(&agentrun.ToolDefinition{Name: "new_tool", Description: "New tool"})

	got := set.Get()
	if len(got.Tools) != 2 || got.Tools[0].Name != "search_tool" || got.Tools[1].Name != "new_tool" {
		t.Errorf("expected [search_tool, new_tool], got %+v", got.Tools)
	}
}

func TestAnExtensionIsAppendedToThePrompt(t *testing.T) {
	var set Toolset
	set.ExtendPrompt("Base prompt.")

	if got := set.ExtendPrompt(" Extra instructions."); got != "Base prompt. Extra instructions." {
		t.Errorf("unexpected full prompt: %q", got)
	}
	if got := set.GetPrompt(); got != "Base prompt. Extra instructions." {
		t.Errorf("unexpected prompt: %q", got)
	}
	if got := set.Get().SystemPromptExtension; got != "Base prompt. Extra instructions." {
		t.Errorf("unexpected extension: %q", got)
	}
}

func TestTheToolsReturnedAreACopy(t *testing.T) {
	var set Toolset
	set.Register(&agentrun.ToolDefinition{Name: "one"})

	got := set.Get()
	got.Tools[0] = &agentrun.ToolDefinition{Name: "changed"}

	if name := set.Get().Tools[0].Name; name != "one" {
		t.Errorf("the registered tool was changed to %q", name)
	}
}
