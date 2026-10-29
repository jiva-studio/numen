package service

import (
	"context"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
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

func getLevelNames(levels []*agentrun.ReasoningLevel) []string {
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
	byName := map[string]*agentrun.AgentModel{}
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
func TestModels_QueriesProgramForModels(t *testing.T) {
	request := t.TempDir() + "/asked"
	answer := strings.Replace(readFixture(t), `"r1"`, `"numen-models"`, 1)
	answer = strings.ReplaceAll(answer, "'", "")
	script := writeScript(t, "claude", "read line\nprintf '%s\\n' \"$line\" > "+request+"\ncat <<'EOF'\n"+answer+"EOF\nsleep 30\n")
	t.Setenv("NUMEN_CLAUDE_COMMAND", script)
	client := startAgent(t, &agentServer{})

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&agentrun.GetAgentProgramsRequest{}))
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

	got, err := client.GetAgentPrograms(t.Context(), connect.NewRequest(&agentrun.GetAgentProgramsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	for place, one := range got.Msg.GetPrograms()[0].GetModels() {
		if one.GetName() != fallbackModels[place].Value || len(one.GetReasoningLevels()) != 0 {
			t.Errorf("the model %v is not a fallback alias", one)
		}
	}
}

func TestModels_DepthLimitedToModelLevels(t *testing.T) {
	s := &agentServer{}
	models, err := parseModels(strings.NewReader(readFixture(t)), "r1")
	if err != nil {
		t.Fatal(err)
	}
	s.getModelCache().Put("claude", models)
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

func TestModels_CachedListIsNotRequeried(t *testing.T) {
	s := &agentServer{}
	s.getModelCache().Put("x", []claudeModel{{Value: "m"}})
	got, err := s.getModels(context.Background(), []string{"x"}, nil)
	if err != nil || len(got) != 1 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestAnAsyncCacheReturnsEmptyWhileInFlightAndFallbackAfterFailure(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	gate := t.TempDir() + "/gate"
	script := writeScript(t, "claude", "while [ ! -f "+gate+" ]; do sleep 0.01; done\nexit 1\n")
	s := &agentServer{models: newModelCache(true)}
	got, err := s.getModels(context.Background(), []string{script}, nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("in-flight async cache returned %v, %v", got, err)
	}
	if err := os.WriteFile(gate, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && !s.getModelCache().HasFailed(script) {
		time.Sleep(5 * time.Millisecond)
	}
	got, err = s.getModels(context.Background(), []string{script}, nil)
	if err != nil || len(got) != len(fallbackModels) {
		t.Fatalf("failed async cache returned %v, %v", got, err)
	}
}
