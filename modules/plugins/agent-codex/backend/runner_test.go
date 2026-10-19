package main

import (
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun/agenttest"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
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

	w, err := agentrun.Start(t.Context(), &reg, newSpec(&numenv1.AskAgentRequest{Command: []string{script}}))
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
	if got[2].GetSaid() != "plain text\n" {
		t.Errorf("expected the plain line said, got %v", got[2])
	}
}
