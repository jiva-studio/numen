package agentrun

import "testing"

func TestARequestNamingNoToolHasNone(t *testing.T) {
	if HasTools(&Request{ToolsURL: "http://127.0.0.1:1/mcp", ToolsToken: "t"}) {
		t.Error("a request with an endpoint and no tool name has tools")
	}
	if !HasTools(&Request{Tools: []string{"note_read"}}) {
		t.Error("a request naming a tool has none")
	}
}

func TestThePromptExistsAfterTheInstructions(t *testing.T) {
	if got := WritePrompt("be brief", "why?"); got != "be brief\n\nwhy?" {
		t.Errorf("got %q", got)
	}
	if got := WritePrompt(" \n", "why?"); got != "why?" {
		t.Errorf("got %q", got)
	}
}
