package service

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cache"
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

func getLevelNames(levels []reasoning.Level) []string {
	names := []string{}
	for _, one := range levels {
		names = append(names, one.Name)
	}
	return names
}

func TestTheModelsAreReadFromTheAnswerOfTheProgram(t *testing.T) {
	models, err := parseModels(strings.NewReader(
		`{"type":"system","subtype":"hook_started"}`+"\n"+readFixture(t)), "r1")
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]*reasoning.Model{}
	for _, one := range models {
		byName[one.Value] = one.toProto()
	}
	if got := getLevelNames(byName["opus"].ReasoningLevels); !slices.Equal(got, []string{"low", "medium", "high", "xhigh", "max"}) {
		t.Errorf("opus offers %v", got)
	}
	if got := getLevelNames(byName["claude-sonnet-4-6"].ReasoningLevels); !slices.Equal(got, []string{"low", "medium", "high", "max"}) {
		t.Errorf("sonnet 4.6 offers %v", got)
	}
	if got := byName["claude-haiku-4-5-20251001"]; len(got.ReasoningLevels) != 0 || got.Title != "Haiku 4.5" {
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

// The program receives the request over stdin and its answer carries the request id.
func TestModels_QueriesProgramForModels(t *testing.T) {
	request := t.TempDir() + "/asked"
	answer := strings.Replace(readFixture(t), `"r1"`, `"numen-models"`, 1)
	answer = strings.ReplaceAll(answer, "'", "")
	script := writeScript(t, "claude", "read line\nprintf '%s\\n' \"$line\" > "+request+"\ncat <<'EOF'\n"+answer+"EOF\nsleep 30\n")
	t.Setenv("NUMEN_CLAUDE_COMMAND", script)
	client := startAgent(t, &agentServer{})

	got, err := client.GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	models := got[0].Models
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

// A program that exits as soon as it has answered lists its models and not the fallback: on macOS the
// kill of its exited process group fails.
func TestAProgramThatExitsAfterAnsweringListsItsModels(t *testing.T) {
	answer := strings.Replace(readFixture(t), `"r1"`, `"numen-models"`, 1)
	answer = strings.ReplaceAll(answer, "'", "")
	script := writeScript(t, "claude", "read -r _\ncat <<'EOF'\n"+answer+"EOF\nexit 0\n")

	models, err := fetchModels(t.Context(), []string{script}, os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 13 {
		t.Errorf("the models are %v", models)
	}
}

func TestGetModels_FallbackWhenListFails(t *testing.T) {
	t.Setenv("NUMEN_CLAUDE_COMMAND", writeScript(t, "claude", "exit 1\n"))
	client := startAgent(t, &agentServer{})

	got, err := client.GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for place, one := range got[0].Models {
		if one.Name != getFallbackModels()[place].Value || len(one.ReasoningLevels) != 0 {
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
	got, err := s.getModels(t.Context(), []string{"x"}, nil)
	if err != nil || len(got) != 1 {
		t.Fatalf("got %v, %v", got, err)
	}
}

// The load runs in the background and fails when the test lets it. The cache reports the failure
// to its error handler after it has recorded it, so waiting for the report is waiting for the
// failure.
func TestModelCache_AsyncReturnsEmptyThenFallbackOnFailure(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	fail := make(chan struct{})
	reported := make(chan error, 2)
	s := &agentServer{
		models: newModelCache(true, cache.WithErrorHandler(func(err error) { reported <- err })),
		fetch: func(ctx context.Context, _, _ []string) ([]claudeModel, error) {
			select {
			case <-fail:
				return nil, errors.New("the program lists no model")
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}
	command := []string{"claude"}
	got, err := s.getModels(t.Context(), command, nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("in-flight async cache returned %v, %v", got, err)
	}
	close(fail)
	select {
	case <-reported:
	case <-t.Context().Done():
		t.Fatal("the load did not report its failure")
	}
	got, err = s.getModels(t.Context(), command, nil)
	if err != nil || len(got) != len(getFallbackModels()) {
		t.Fatalf("failed async cache returned %v, %v", got, err)
	}
}
