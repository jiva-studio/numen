package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func TestResolveCommandWithEmptyCommand(t *testing.T) {
	withoutAgentsAnywhere(t)

	bin, args := resolveCommand(RunParams{})
	if bin != "codex" || len(args) != 0 {
		t.Errorf("expected codex and empty args, got %s, %v", bin, args)
	}

}

func TestStartRunWorkBufferCapacity(t *testing.T) {
	needsShell(t)
	script := writeScript(t, "runner", "exit 0\n")
	runner := &AgentRunner{Registry: &agentrun.Registry{}}

	w, err := runner.StartRun(t.Context(), RunParams{
		Command: []string{script},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Stop() }()

	if got := cap(w.Output); got != 64 {
		t.Errorf("expected work output channel capacity 64, got %d", got)
	}
}

func TestConfigureCommandTokenEnv(t *testing.T) {
	needsShell(t)
	out := filepath.Join(t.TempDir(), "token")
	script := writeScript(t, "runner", "printf '%s' \"$NUMEN_TOKEN\" > "+quoteShell(out)+"\n")
	runner := &AgentRunner{Registry: &agentrun.Registry{}}

	w, err := runner.StartRun(t.Context(), RunParams{
		Command: []string{script},
		Tools:   Endpoint{Token: "test-token-xyz"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Stop() }()

	// Wait for the script to finish and write out.
	for range w.Output {
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "test-token-xyz" {
		t.Errorf("expected NUMEN_TOKEN 'test-token-xyz', got %q", string(data))
	}
}

func TestStartRunReportsTheToolCallsOfCodex(t *testing.T) {
	script := writeScript(t, "codex",
		`echo '{"type":"item.started","item":{"type":"mcp_tool_call","name":"read_note","arguments":{"path":"a.md"}}}'`+"\n"+
			`echo '{"type":"item.completed","item":{"type":"mcp_tool_call"}}'`+"\n"+
			"echo plain text\n")
	runner := &AgentRunner{Registry: &agentrun.Registry{}}

	w, err := runner.StartRun(t.Context(), RunParams{Command: []string{script}})
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
