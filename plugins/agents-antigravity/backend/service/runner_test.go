package service

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun/agenttest"
)

func TestTheSessionAndTheContentOfAntigravityAreReported(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		`echo '{"session_id":"s-1","content":"hello"}'`+"\n"+
			`echo '{"tool_calls":[{"name":"read_note","arguments":{"path":"a.md"}}]}'`+"\n")
	agent := newAgentServer(false)
	reg := &agent.Registry

	w, err := agent.runner.Start(t.Context(), &agentrun.Request{Command: []string{script}, Conversation: "c-1"}, "")
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
	if len(got) != 2 || got[0].GetText() != "hello" || got[1].GetToolCall().Tool != "read_note" {
		t.Fatalf("the steps are %v", got)
	}
	if session := reg.GetSession("c-1"); session != "s-1" {
		t.Errorf("expected session s-1, got %q", session)
	}
}

func TestAFailedHomeSetupEndsTheRunBeforeItStarts(t *testing.T) {
	setUnusableTempDir(t)
	agent := newAgentServer(false)

	_, err := agent.runner.Start(t.Context(), &agentrun.Request{Command: []string{"agy"}, ToolsURL: "http://127.0.0.1:1/mcp"}, "")

	if err == nil || !strings.Contains(err.Error(), "set up the Antigravity home") {
		t.Errorf("got %v", err)
	}
}

func TestTheToolCallsOfAntigravityAreCountedAgainstTheStepLimit(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		`while true; do echo '{"tool_calls":[{"name":"read_note","arguments":{}}]}'; done`+"\n")
	agent := newAgentServer(false)

	w, err := agent.runner.Start(t.Context(), &agentrun.Request{Command: []string{script}, MaxSteps: 2}, "")
	if err != nil {
		t.Fatal(err)
	}
	var got []*agentrun.Response
	for resp := range w.Output {
		got = append(got, resp)
	}
	if last := got[len(got)-1]; last.GetStopped() != "reached maximum limit of 2 steps" {
		t.Errorf("the last step is %v", last)
	}
}

func TestAStreamedToolCallIsCountedAgainstTheStepLimit(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		`while true; do
echo '{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"tool_use","name":"read_note"}}}'
echo '{"type":"stream_event","event":{"type":"content_block_stop"}}'
done`+"\n")
	agent := newAgentServer(false)

	w, err := agent.runner.Start(t.Context(), &agentrun.Request{Command: []string{script}, MaxSteps: 1}, "")
	if err != nil {
		t.Fatal(err)
	}
	var got []*agentrun.Response
	for resp := range w.Output {
		got = append(got, resp)
	}
	if last := got[len(got)-1]; last.GetStopped() != "reached maximum limit of 1 steps" {
		t.Errorf("the last step is %v", last)
	}
}

func setUnusableTempDir(t *testing.T) {
	t.Helper()
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, filepath.Join(t.TempDir(), "absent"))
	}
}

func collect(t *testing.T, script string) ([]*agentrun.Response, *agentrun.Registry) {
	t.Helper()
	agent := newAgentServer(false)
	reg := &agent.Registry
	w, err := agent.runner.Start(t.Context(), &agentrun.Request{Command: []string{script}, Conversation: "c-1", Prompt: "q"}, "")
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
	return got, reg
}

// The CLI answer arrives as the text deltas of its response steps. Its final
// result repeats them, so it is not emitted again.
func TestRunner_CLIAnswerEmittedOnce(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		`echo '{"event":"init","conversation_id":"s-1","init":{"model":"m"}}'`+"\n"+
			`echo '{"event":"step_update","step_update":{"conversation_id":"s-1","step_index":0,"state":"DONE","step_type":"user_input"}}'`+"\n"+
			`echo '{"event":"step_update","step_update":{"conversation_id":"s-1","step_index":1,"state":"ACTIVE","step_type":"agent_response","text_delta":"ok"}}'`+"\n"+
			`printf '%s\n' '{"event":"step_update","step_update":{"conversation_id":"s-1","step_index":1,"state":"DONE","step_type":"agent_response","text_delta":"\n"}}'`+"\n"+
			`printf '%s\n' '{"event":"result","result":{"conversation_id":"s-1","status":"SUCCESS","response":"ok\n"}}'`+"\n")

	got, reg := collect(t, script)

	var response strings.Builder
	for _, resp := range got {
		if resp.GetStopped() != "" {
			t.Fatalf("stopped: %v", resp)
		}
		response.WriteString(resp.GetText())
	}
	if response.String() != "ok\n" {
		t.Errorf("said %q, want %q", response.String(), "ok\n")
	}
	if session := reg.GetSession("c-1"); session != "s-1" {
		t.Errorf("session %q, want s-1", session)
	}
}

// A result with a response and no streamed text emits the response.
func TestRunner_CLIResponseEmittedWhenNothingStreamed(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		`echo '{"event":"result","result":{"status":"SUCCESS","response":"fine"}}'`+"\n")

	got, _ := collect(t, script)

	if len(got) != 1 || got[0].GetText() != "fine" {
		t.Errorf("the steps are %v", got)
	}
}

// A result event ends the run even when a child process keeps the output pipe open.
func TestResultEventEndsRunWithOpenChildOutput(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		"(sleep 60) &\n"+
			`echo '{"event":"step_update","step_update":{"step_type":"agent_response","text_delta":"done"}}'`+"\n"+
			`echo '{"event":"result","result":{"status":"SUCCESS","response":"done"}}'`+"\n"+
			"wait\n")

	got, _ := collect(t, script)

	if len(got) != 1 || got[0].GetText() != "done" {
		t.Errorf("the steps are %v", got)
	}
}

func TestAnErrorResultOfTheCliStopsTheRunWithItsError(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		`echo '{"event":"result","result":{"status":"ERROR","response":"","error":"Eligibility check failed: 503"}}'`+"\n"+
			`echo 'error: Eligibility check failed: 503' >&2`+"\nexit 1\n")

	got, _ := collect(t, script)

	if len(got) == 0 || got[0].GetStopped() != "Eligibility check failed: 503" {
		t.Errorf("the steps are %v", got)
	}
}

// A model the CLI does not know ends it with the error and the list of the models on its error
// output: the run stops with the error line.
func TestAModelTheCliDoesNotKnowStopsTheRunWithTheErrorLine(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		`echo 'error: invalid model selection (--model "m-old"): not recognized' >&2`+"\n"+
			`printf 'Available models:\n  Model One\n  Model Two\n' >&2`+"\nexit 1\n")

	got, _ := collect(t, script)

	want := `invalid model selection (--model "m-old"): not recognized`
	if len(got) != 1 || got[0].GetStopped() != want {
		t.Errorf("the steps are %v, want a stop of %q", got, want)
	}
}
