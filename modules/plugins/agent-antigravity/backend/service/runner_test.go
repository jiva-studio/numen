package service

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun/agenttest"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func TestResolveCommandWithEmptyCommand(t *testing.T) {
	withoutAgentsAnywhere(t)

	bin, args := resolveCommand(nil)
	if bin != "agy" || len(args) != 0 {
		t.Errorf("expected agy and empty args, got %s, %v", bin, args)
	}
}

func TestTheSessionOfTheConversationIsHandedToTheNextRun(t *testing.T) {
	var reg agentrun.Registry
	reg.SetSession("c-1", "s-1")

	cmd, _, _ := makeAntigravityCommand(t.Context(), &numenv1.AskAgentRequest{Conversation: "c-1", Prompt: "q"}, &reg)

	if !slices.Contains(cmd.Args, "--conversation") || !slices.Contains(cmd.Args, "s-1") {
		t.Errorf("the arguments are %v", cmd.Args)
	}
}

func TestTheSessionAndTheContentOfAntigravityAreReported(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		`echo '{"session_id":"s-1","content":"hello"}'`+"\n"+
			`echo '{"tool_calls":[{"name":"read_note","arguments":{"path":"a.md"}}]}'`+"\n")
	var reg agentrun.Registry

	w, err := agentrun.Start(t.Context(), &reg, newSpec(
		&numenv1.AskAgentRequest{Command: []string{script}, Conversation: "c-1"}, &reg))
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

func TestAFailedHomeSetupEndsTheRunBeforeItStarts(t *testing.T) {
	setUnusableTempDir(t)
	var reg agentrun.Registry

	_, err := agentrun.Start(t.Context(), &reg, newSpec(
		&numenv1.AskAgentRequest{Command: []string{"agy"}, ToolsUrl: "http://127.0.0.1:1/mcp"}, &reg))

	if err == nil || !strings.Contains(err.Error(), "set up the Antigravity home") {
		t.Errorf("got %v", err)
	}
}

func TestTheToolCallsOfAntigravityAreCountedAgainstTheStepLimit(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		`while true; do echo '{"tool_calls":[{"name":"read_note","arguments":{}}]}'; done`+"\n")
	var reg agentrun.Registry

	w, err := agentrun.Start(t.Context(), &reg, newSpec(
		&numenv1.AskAgentRequest{Command: []string{script}, MaxSteps: 2}, &reg))
	if err != nil {
		t.Fatal(err)
	}
	var got []*numenv1.AskAgentResponse
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
	var reg agentrun.Registry

	w, err := agentrun.Start(t.Context(), &reg, newSpec(
		&numenv1.AskAgentRequest{Command: []string{script}, MaxSteps: 1}, &reg))
	if err != nil {
		t.Fatal(err)
	}
	var got []*numenv1.AskAgentResponse
	for resp := range w.Output {
		got = append(got, resp)
	}
	if last := got[len(got)-1]; last.GetStopped() != "reached maximum limit of 1 steps" {
		t.Errorf("the last step is %v", last)
	}
}

// setUnusableTempDir points every variable the temporary folder is read from at a folder that does
// not exist.
func setUnusableTempDir(t *testing.T) {
	t.Helper()
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, filepath.Join(t.TempDir(), "absent"))
	}
}

// collect reads every response of the run of the script a request with no tools asks for.
func collect(t *testing.T, script string) ([]*numenv1.AskAgentResponse, *agentrun.Registry) {
	t.Helper()
	var reg agentrun.Registry
	w, err := agentrun.Start(t.Context(), &reg, newSpec(
		&numenv1.AskAgentRequest{Command: []string{script}, Conversation: "c-1", Prompt: "q"}, &reg))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Stop() }()
	var got []*numenv1.AskAgentResponse
	for resp := range w.Output {
		got = append(got, resp)
	}
	return got, &reg
}

// The answer of the CLI arrives as the text deltas of its response steps, and its result, which
// repeats them, is not said again.
func TestTheAnswerOfTheCliIsSaidOnce(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		`echo '{"event":"init","conversation_id":"s-1","init":{"model":"m"}}'`+"\n"+
			`echo '{"event":"step_update","step_update":{"conversation_id":"s-1","step_index":0,"state":"DONE","step_type":"user_input"}}'`+"\n"+
			`echo '{"event":"step_update","step_update":{"conversation_id":"s-1","step_index":1,"state":"ACTIVE","step_type":"agent_response","text_delta":"ok"}}'`+"\n"+
			`echo '{"event":"step_update","step_update":{"conversation_id":"s-1","step_index":1,"state":"DONE","step_type":"agent_response","text_delta":"\n"}}'`+"\n"+
			`echo '{"event":"result","result":{"conversation_id":"s-1","status":"SUCCESS","response":"ok\n"}}'`+"\n")

	got, reg := collect(t, script)

	var said strings.Builder
	for _, resp := range got {
		if resp.GetStopped() != "" {
			t.Fatalf("stopped: %v", resp)
		}
		said.WriteString(resp.GetSaid())
	}
	if said.String() != "ok\n" {
		t.Errorf("said %q, want %q", said.String(), "ok\n")
	}
	if session := reg.GetSession("c-1"); session != "s-1" {
		t.Errorf("session %q, want s-1", session)
	}
}

// A result with a response and no streamed text says the response.
func TestTheResponseOfTheCliIsSaidWhereNoTextWasStreamed(t *testing.T) {
	script := agenttest.WriteScript(t, "agy",
		`echo '{"event":"result","result":{"status":"SUCCESS","response":"fine"}}'`+"\n")

	got, _ := collect(t, script)

	if len(got) != 1 || got[0].GetSaid() != "fine" {
		t.Errorf("the steps are %v", got)
	}
}

// A result in error stops the run with the error the CLI gave.
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
