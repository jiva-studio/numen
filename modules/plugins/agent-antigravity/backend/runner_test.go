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
	if bin != "agy" || len(args) != 0 {
		t.Errorf("expected agy and empty args, got %s, %v", bin, args)
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

func TestStartRunKeepsTheSessionAndSaysTheContent(t *testing.T) {
	script := writeScript(t, "agy",
		`echo '{"session_id":"s-1","content":"hello"}'`+"\n"+
			`echo '{"tool_calls":[{"name":"read_note","arguments":{"path":"a.md"}}]}'`+"\n")
	reg := &agentrun.Registry{}
	runner := &AgentRunner{Registry: reg}

	w, err := runner.StartRun(t.Context(), RunParams{Command: []string{script}, Conversation: "c-1"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Stop() }()

	var got []*numenv1.AskAgentResponse
	for resp := range w.Output {
		got = append(got, resp)
	}
	if len(got) != 2 || got[0].GetSaid() != "hello" || got[1].GetToolCall().GetTool() != "read_note" {
		t.Fatalf("the steps are %v", got)
	}
	if session := reg.GetSession("c-1"); session != "s-1" {
		t.Errorf("expected session s-1, got %q", session)
	}
}
