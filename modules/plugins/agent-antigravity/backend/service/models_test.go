package service

import (
	"slices"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun/agenttest"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func TestTheModelsAreTheLinesThatHoldANameAndATitle(t *testing.T) {
	got := parseAntigravityModels([]byte(
		"Fetching available models...\n" +
			"m-high\tModel (High)\n" +
			"bare\t\n" +
			"\tno name\n" +
			"m-low\tModel (Low)\n"))
	want := []antigravityModel{{"m-high", "Model (High)"}, {"bare", "bare"}, {"m-low", "Model (Low)"}}
	if !slices.Equal(got, want) {
		t.Errorf("the models are %v, want %v", got, want)
	}
}

// The program list holds the models the installed program lists with "models", whatever the
// settings once named.
func TestTheProgramListHoldsTheModelsTheProgramListsFoldedByLevel(t *testing.T) {
	agenttest.NeedsShell(t)
	t.Setenv("HOME", t.TempDir())
	script := agenttest.WriteScript(t, "agy",
		`if [ "$1" = models ]; then printf 'Fetching available models...\nm-high\tModel (High)\nm-low\tModel (Low)\n'; fi`+"\n")
	client := agenttest.Serve(t, &agentServer{})

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{
		Commands: []*numenv1.AgentProgramCommand{{Program: "antigravity", Command: []string{script}}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	models := got.Msg.GetPrograms()[0].GetModels()
	if len(models) != 1 || models[0].GetName() != "m" || models[0].GetTitle() != "Model" {
		t.Fatalf("the models are %v", models)
	}
	var levels []string
	for _, one := range models[0].GetReasoningLevels() {
		levels = append(levels, one.GetName())
	}
	if !slices.Equal(levels, []string{"low", "high"}) || models[0].GetDefaultReasoning() != "low" {
		t.Errorf("the levels are %v, the default %q", levels, models[0].GetDefaultReasoning())
	}
}

func TestTheProgramListHoldsNoModelWhereTheProgramCannotListThem(t *testing.T) {
	agenttest.NeedsShell(t)
	t.Setenv("HOME", t.TempDir())
	script := agenttest.WriteScript(t, "agy", "exit 1\n")
	client := agenttest.Serve(t, &agentServer{})

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{
		Commands: []*numenv1.AgentProgramCommand{{Program: "antigravity", Command: []string{script}}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	program := got.Msg.GetPrograms()[0]
	if !program.GetIsInstalled() || len(program.GetModels()) != 0 {
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
		req := &numenv1.AskAgentRequest{Model: one.model, Reasoning: one.reasoning, Command: []string{script}}
		(&agentServer{}).resolveModel(t.Context(), req)
		if req.GetModel() != one.want {
			t.Errorf("model %q at %q became %q, want %q", one.model, one.reasoning, req.GetModel(), one.want)
		}
	}
}
