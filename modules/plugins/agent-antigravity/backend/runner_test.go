package main

import (
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
	t.Setenv("TMPDIR", "/nonexistent/tmp")
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
