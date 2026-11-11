package run_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun/agenttest"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/internal/adapter/codexcli"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/usecase/run"
)

func TestTheToolCallsOfCodexAreReported(t *testing.T) {
	script := agenttest.WriteScript(t, "codex",
		`echo '{"type":"item.started","item":{"type":"mcp_tool_call","name":"read_note","arguments":{"path":"a.md"}}}'`+"\n"+
			`echo '{"type":"item.completed","item":{"type":"mcp_tool_call"}}'`+"\n"+
			`echo '{"type":"item.completed","item":{"type":"agent_message"}}'`+"\n"+
			`echo '{"type":"turn.completed"}'`+"\n"+
			"echo plain text\n")
	var reg agentrun.Registry

	w, err := newTestRunner().Start(t.Context(), &reg, &agentrun.Request{Command: []string{script}}, "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := w.Stop(); err != nil {
			t.Error(err)
		}
	}()

	var got []*agentrun.Response
	for resp := range w.Output {
		got = append(got, resp)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 steps, got %v", got)
	}
	if call := got[0].GetToolCall(); call.Tool != "read_note" || call.About != `{"path":"a.md"}` {
		t.Errorf("tool call is %v", got[0])
	}
	if !got[1].IsToolFinished() {
		t.Errorf("expected a finished tool, got %v", got[1])
	}
	if got[2].GetText() != "plain text\n" {
		t.Errorf("expected the plain line as text, got %v", got[2])
	}
}

func TestRunner_EmitsCodexMessages(t *testing.T) {
	script := agenttest.WriteScript(t, "codex",
		`echo '{"type":"item.completed","item":{"id":"i0","type":"agent_message","text":"The answer."}}'`+"\n"+
			`echo '{"type":"item.completed","item":{"type":"agent_message","text":""}}'`+"\n"+
			`echo '{"type":"item.completed","item":{"type":"reasoning","text":"hidden"}}'`+"\n")
	var reg agentrun.Registry

	w, err := newTestRunner().Start(t.Context(), &reg, &agentrun.Request{Command: []string{script}}, "")
	if err != nil {
		t.Fatal(err)
	}
	var got []*agentrun.Response
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

	w, err := newTestRunner().Start(t.Context(), &reg, &agentrun.Request{Command: []string{script}}, "")
	if err != nil {
		t.Fatal(err)
	}
	var got []*agentrun.Response
	for resp := range w.Output {
		got = append(got, resp)
	}
	if len(got) != 1 || got[0].GetToolCall().Tool != "search_notes" {
		t.Errorf("got %v", got)
	}
}

func newTestRunner() *run.Runner {
	return run.NewRunner(codexcli.Commands{}, process.Launcher{})
}
