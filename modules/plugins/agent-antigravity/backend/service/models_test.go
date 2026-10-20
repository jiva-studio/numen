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
func TestTheProgramListHoldsTheModelsTheProgramLists(t *testing.T) {
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
	var names, titles []string
	for _, one := range got.Msg.GetPrograms()[0].GetModels() {
		names, titles = append(names, one.GetName()), append(titles, one.GetTitle())
	}
	if !slices.Equal(names, []string{"m-high", "m-low"}) || !slices.Equal(titles, []string{"Model (High)", "Model (Low)"}) {
		t.Errorf("the models are %v %v", names, titles)
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
