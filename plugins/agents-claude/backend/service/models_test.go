package service

import (
	"context"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

// initializeFixture is the answer of Claude Code 2.1.303 to the initialize request, cut to its models.
const initializeFixture = "testdata/initialize-response.json"

func readFixture(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(initializeFixture)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func getLevelNames(levels []*numenv1.ReasoningLevel) []string {
	names := []string{}
	for _, one := range levels {
		names = append(names, one.GetName())
	}
	return names
}

func TestTheModelsAreReadFromTheAnswerOfTheProgram(t *testing.T) {
	models, err := parseModels(strings.NewReader(
		`{"type":"system","subtype":"hook_started"}`+"\n"+readFixture(t)), "r1")
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]*numenv1.AgentModel{}
	for _, one := range models {
		byName[one.Value] = one.toProto()
	}
	if got := getLevelNames(byName["opus"].GetReasoningLevels()); !slices.Equal(got, []string{"low", "medium", "high", "xhigh", "max"}) {
		t.Errorf("opus offers %v", got)
	}
	if got := getLevelNames(byName["claude-sonnet-4-6"].GetReasoningLevels()); !slices.Equal(got, []string{"low", "medium", "high", "max"}) {
		t.Errorf("sonnet 4.6 offers %v", got)
	}
	if got := byName["claude-haiku-4-5-20251001"]; len(got.GetReasoningLevels()) != 0 || got.GetTitle() != "Haiku 4.5" {
		t.Errorf("a model of no effort offers a level: %v", got)
	}
	if byName[defaultModelName] == nil {
		t.Error("the default model is not listed")
	}
}

func TestAnswerToAnotherRequestOrWithoutModelsIsAnError(t *testing.T) {
	if _, err := parseModels(strings.NewReader(readFixture(t)), "other"); err == nil {
		t.Error("an answer to another request was read")
	}
	errorResp := `{"type":"control_response","response":{"subtype":"error","request_id":"r1","error":"no"}}`
	if _, err := parseModels(strings.NewReader(errorResp), "r1"); err == nil {
		t.Error("an error was read as models")
	}
	empty := `{"type":"control_response","response":{"subtype":"success","request_id":"r1","response":{"models":[]}}}`
	if _, err := parseModels(strings.NewReader(empty), "r1"); err == nil {
		t.Error("an empty list was read as models")
	}
}

// The program is asked over stdin and its answer carries the id of the request.
func TestTheProgramIsAskedForItsModels(t *testing.T) {
	request := t.TempDir() + "/asked"
	answer := strings.Replace(readFixture(t), `"r1"`, `"numen-models"`, 1)
	answer = strings.ReplaceAll(answer, "'", "")
	script := writeScript(t, "claude", "read line\nprintf '%s\\n' \"$line\" > "+request+"\ncat <<'EOF'\n"+answer+"EOF\nsleep 30\n")
	t.Setenv("NUMEN_CLAUDE_COMMAND", script)
	client := startAgent(t, &agentServer{})

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	models := got.Msg.GetPrograms()[0].GetModels()
	if len(models) != 13 {
		t.Fatalf("the models are %v", models)
	}
	raw, err := os.ReadFile(request)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"subtype":"initialize"`) {
		t.Errorf("the program was asked %s", raw)
	}
}

func TestAProgramThatCannotListItsModelsOffersTheFallback(t *testing.T) {
	t.Setenv("NUMEN_CLAUDE_COMMAND", writeScript(t, "claude", "exit 1\n"))
	client := startAgent(t, &agentServer{})

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	for place, one := range got.Msg.GetPrograms()[0].GetModels() {
		if one.GetName() != fallbackModels[place].Value || len(one.GetReasoningLevels()) != 0 {
			t.Errorf("the model %v is not a fallback alias", one)
		}
	}
}

func TestTheDepthAskedIsHeldToTheLevelsOfTheModel(t *testing.T) {
	s := &agentServer{}
	models, err := parseModels(strings.NewReader(readFixture(t)), "r1")
	if err != nil {
		t.Fatal(err)
	}
	s.models.lists = map[string]modelList{"claude": {models: models}}
	cases := []struct{ model, requestedLevel, want string }{
		{"claude-sonnet-4-6", "xhigh", ""},
		{"claude-sonnet-4-6", "max", "max"},
		{"claude-haiku-4-5-20251001", "high", ""},
		{"", "high", "high"},
		{"unlisted", "high", "high"},
		{"unlisted", "bogus", ""},
	}
	for _, one := range cases {
		if got := s.getReasoning(one.model, one.requestedLevel); got != one.want {
			t.Errorf("%q requested %q got %q, want %q", one.model, one.requestedLevel, got, one.want)
		}
	}
}

func TestACachedListIsNotAskedAgain(t *testing.T) {
	var cache modelCache
	cache.lists = map[string]modelList{"x": {models: []claudeModel{{Value: "m"}}, fetchedAt: time.Now()}}
	got, err := cache.get(context.Background(), []string{"x"}, nil)
	if err != nil || len(got) != 1 {
		t.Fatalf("got %v, %v", got, err)
	}
}
