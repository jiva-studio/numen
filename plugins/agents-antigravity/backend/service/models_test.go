package service

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun/agenttest"
)

// The program list holds the models the installed program lists with "models", whatever the
// settings once named.
func TestGetAgentPrograms_ModelListGroupedByLevel(t *testing.T) {
	agenttest.NeedsShell(t)
	t.Setenv("HOME", t.TempDir())
	script := agenttest.WriteScript(t, "agy",
		`if [ "$1" = models ]; then printf 'Fetching available models...\nm-high\tModel (High)\nm-low\tModel (Low)\n'; fi`+"\n")
	client := serveAgent(t, newAgentServer(false))

	got, err := client.GetPrograms(t.Context(), []agentrun.ProgramCommand{{Program: "antigravity", Command: []string{script}}})
	if err != nil {
		t.Fatal(err)
	}
	models := got[0].Models
	if len(models) != 1 || models[0].Name != "m" || models[0].Title != "Model" {
		t.Fatalf("the models are %v", models)
	}
	var levels []string
	for _, one := range models[0].ReasoningLevels {
		levels = append(levels, one.Name)
	}
	if !slices.Equal(levels, []string{"low", "high"}) || models[0].DefaultReasoning != "low" {
		t.Errorf("the levels are %v, the default %q", levels, models[0].DefaultReasoning)
	}
}

func TestGetAgentPrograms_ModelListFailure(t *testing.T) {
	agenttest.NeedsShell(t)
	t.Setenv("HOME", t.TempDir())
	script := agenttest.WriteScript(t, "agy", "exit 1\n")
	client := serveAgent(t, newAgentServer(false))

	got, err := client.GetPrograms(t.Context(), []agentrun.ProgramCommand{{Program: "antigravity", Command: []string{script}}})
	if err != nil {
		t.Fatal(err)
	}
	program := got[0]
	if !program.IsInstalled || len(program.Models) != 0 {
		t.Errorf("the program is %v", program)
	}
}

func TestTheLevelOfARequestChoosesTheListedModel(t *testing.T) {
	agenttest.NeedsShell(t)
	t.Setenv("HOME", t.TempDir())
	script := agenttest.WriteScript(t, "agy",
		`if [ "$1" = models ]; then printf 'm-high\tModel (High)\nm-low\tModel (Low)\nsolo\tSolo\n'; fi`+"\n")
	cases := []struct{ model, reasoning, want string }{
		{"m", "high", "m-high"},
		{"m", "low", "m-low"},
		{"m", "", "m-low"},
		{"m-high", "low", "m-high"},
		{"solo", "high", "solo"},
		{"", "high", ""},
		{"default", "high", "default"},
	}
	for _, one := range cases {
		req := &agentrun.Request{Model: one.model, Reasoning: one.reasoning, Command: []string{script}}
		newAgentServer(false).catalog.ResolveModel(t.Context(), req)
		if req.Model != one.want {
			t.Errorf("model %q at %q became %q, want %q", one.model, one.reasoning, req.Model, one.want)
		}
	}
}

func stubCli(t *testing.T, listing string) string {
	t.Helper()
	return agenttest.WriteScript(t, "agy",
		`if [ "$1" = models ]; then printf '`+listing+`'; exit 0; fi
model=
while [ $# -gt 0 ]; do if [ "$1" = --model ]; then model=$2; fi; shift; done
case "$model" in
  m-medium|m-low|m-high|solo)
    echo '{"event":"step_update","step_update":{"step_type":"agent_response","text_delta":"ran '$model'"}}' ;;
  *)
    echo 'error: invalid model selection (--model "'$model'"): not recognized' >&2
    exit 1 ;;
esac
`)
}

func askStub(t *testing.T, script, model, level string) (said, stopReason string) {
	t.Helper()
	stream, err := serveAgent(t, newAgentServer(false)).Ask(t.Context(), &agentrun.Request{
		Command: []string{script}, Model: model, Reasoning: level, Prompt: "q",
	})
	if err != nil {
		t.Fatal(err)
	}
	for stream.Receive() {
		said += stream.Msg().GetText()
		stopReason += stream.Msg().GetStopped()
	}
	return said, stopReason
}

func TestModels_FamilyWithoutLevelUsesDefaultLevelModel(t *testing.T) {
	agenttest.NeedsShell(t)
	t.Setenv("HOME", t.TempDir())
	cases := []struct{ name, listing, level, want string }{
		{"medium is the default", "m-high\\tModel (High)\\nm-medium\\tModel (Medium)\\nm-low\\tModel (Low)\\n", "", "ran m-medium"},
		{"a level that is not listed gives the default", "m-high\\tModel (High)\\nm-medium\\tModel (Medium)\\n", "ultra", "ran m-medium"},
		{"the lowest where there is no medium", "m-high\\tModel (High)\\nm-low\\tModel (Low)\\n", "", "ran m-low"},
		{"the chosen level is kept", "m-high\\tModel (High)\\nm-medium\\tModel (Medium)\\n", "high", "ran m-high"},
	}
	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			response, stopReason := askStub(t, stubCli(t, one.listing), "m", one.level)
			if response != one.want || stopReason != "" {
				t.Errorf("said %q and stopReason %q, want %q", response, stopReason, one.want)
			}
		})
	}
}

func TestAModelTheCliDoesNotListStopsTheRunWithTheReasonOfTheCli(t *testing.T) {
	agenttest.NeedsShell(t)
	t.Setenv("HOME", t.TempDir())

	_, stopReason := askStub(t, stubCli(t, "m-medium\\tModel (Medium)\\n"), "gone", "")

	if want := `invalid model selection (--model "gone"): not recognized`; stopReason != want {
		t.Errorf("stopReason with %q, want %q", stopReason, want)
	}
}

func TestAFamilyOfOneListedModelRunsThatModel(t *testing.T) {
	agenttest.NeedsShell(t)
	t.Setenv("HOME", t.TempDir())

	response, stopReason := askStub(t, stubCli(t, "solo\\tSolo\\n"), "solo", "")

	if response != "ran solo" || stopReason != "" {
		t.Errorf("said %q and stopReason %q", response, stopReason)
	}
}
