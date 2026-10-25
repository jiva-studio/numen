package service

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun/agenttest"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

func TestResolveCommandWithEmptyCommand(t *testing.T) {
	withoutAgentsAnywhere(t)

	bin, args := resolveCommand(nil)
	if bin != "codex" || len(args) != 0 {
		t.Errorf("expected codex and empty args, got %s, %v", bin, args)
	}
}

func TestTheToolCallsOfCodexAreReported(t *testing.T) {
	script := agenttest.WriteScript(t, "codex",
		`echo '{"type":"item.started","item":{"type":"mcp_tool_call","name":"read_note","arguments":{"path":"a.md"}}}'`+"\n"+
			`echo '{"type":"item.completed","item":{"type":"mcp_tool_call"}}'`+"\n"+
			`echo '{"type":"item.completed","item":{"type":"agent_message"}}'`+"\n"+
			`echo '{"type":"turn.completed"}'`+"\n"+
			"echo plain text\n")
	var reg agentrun.Registry

	w, err := start(t.Context(), &reg, newSpec(&numenv1.AskAgentRequest{Command: []string{script}}))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Stop() }()

	var got []*numenv1.AskAgentResponse
	for resp := range w.Output {
		got = append(got, resp)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 steps, got %v", got)
	}
	if call := got[0].GetToolCall(); call.GetTool() != "read_note" || call.GetAbout() != `{"path":"a.md"}` {
		t.Errorf("tool call is %v", got[0])
	}
	if got[1].GetToolFinished() == nil {
		t.Errorf("expected a finished tool, got %v", got[1])
	}
	if got[2].GetText() != "plain text\n" {
		t.Errorf("expected the plain line as text, got %v", got[2])
	}
}

func TestTheMessagesOfCodexAreSaid(t *testing.T) {
	script := agenttest.WriteScript(t, "codex",
		`echo '{"type":"item.completed","item":{"id":"i0","type":"agent_message","text":"The answer."}}'`+"\n"+
			`echo '{"type":"item.completed","item":{"type":"agent_message","text":""}}'`+"\n"+
			`echo '{"type":"item.completed","item":{"type":"reasoning","text":"hidden"}}'`+"\n")
	var reg agentrun.Registry

	w, err := start(t.Context(), &reg, newSpec(&numenv1.AskAgentRequest{Command: []string{script}}))
	if err != nil {
		t.Fatal(err)
	}
	var got []*numenv1.AskAgentResponse
	for resp := range w.Output {
		got = append(got, resp)
	}
	if len(got) != 1 || got[0].GetText() != "The answer." {
		t.Errorf("got %v", got)
	}
}

func TestACodexToolCallNamesItsToolByTheToolField(t *testing.T) {
	script := agenttest.WriteScript(t, "codex",
		`echo '{"type":"item.started","item":{"type":"mcp_tool_call","server":"numen","tool":"search_notes","arguments":{"q":"x"}}}'`+"\n")
	var reg agentrun.Registry

	w, err := start(t.Context(), &reg, newSpec(&numenv1.AskAgentRequest{Command: []string{script}}))
	if err != nil {
		t.Fatal(err)
	}
	var got []*numenv1.AskAgentResponse
	for resp := range w.Output {
		got = append(got, resp)
	}
	if len(got) != 1 || got[0].GetToolCall().GetTool() != "search_notes" {
		t.Errorf("got %v", got)
	}
}
