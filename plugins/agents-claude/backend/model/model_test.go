package model

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/jiva-studio/numen/plugins/agents-claude/backend/domain/reasoning"
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
	list, err := Parse(strings.NewReader(
		`{"type":"system","subtype":"hook_started"}`+"\n"+readFixture(t)), "r1")
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Model{}
	for _, one := range list {
		byName[one.Value] = one
	}
	if got := getLevelNames(byName["opus"].GetLevels()); !slices.Equal(got, []string{"low", "medium", "high", "xhigh", "max"}) {
		t.Errorf("opus offers %v", got)
	}
	if got := getLevelNames(byName["claude-sonnet-4-6"].GetLevels()); !slices.Equal(got, []string{"low", "medium", "high", "max"}) {
		t.Errorf("sonnet 4.6 offers %v", got)
	}
	if got := byName["claude-haiku-4-5-20251001"]; len(got.GetLevels()) != 0 || got.DisplayName != "Haiku 4.5" {
		t.Errorf("a model of no effort offers a level: %v", got)
	}
	if _, isListed := byName[DefaultName]; !isListed {
		t.Error("the default model is not listed")
	}
}

func TestAnswerToAnotherRequestOrWithoutModelsIsAnError(t *testing.T) {
	if _, err := Parse(strings.NewReader(readFixture(t)), "other"); err == nil {
		t.Error("an answer to another request was read")
	}
	errorResp := `{"type":"control_response","response":{"subtype":"error","request_id":"r1","error":"no"}}`
	if _, err := Parse(strings.NewReader(errorResp), "r1"); err == nil {
		t.Error("an error was read as models")
	}
	empty := `{"type":"control_response","response":{"subtype":"success","request_id":"r1","response":{"models":[]}}}`
	if _, err := Parse(strings.NewReader(empty), "r1"); err == nil {
		t.Error("an empty list was read as models")
	}
}

func TestModelConstantsLevelsAndParseEdgeCases(t *testing.T) {
	if Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", Timeout)
	}
	if Lifetime != 10*time.Minute {
		t.Errorf("Lifetime = %v, want 10m", Lifetime)
	}

	noEffort := Model{Value: "m", SupportsEffort: false, EffortLevels: []string{"high"}}
	if got := noEffort.GetLevels(); got != nil {
		t.Errorf("expected nil levels when SupportsEffort is false, got %v", got)
	}

	unsorted := Model{Value: "m", SupportsEffort: true, EffortLevels: []string{"max", "low", "high"}}
	if got := getLevelNames(unsorted.GetLevels()); !slices.Equal(got, []string{"low", "high", "max"}) {
		t.Errorf("expected sorted levels [low high max], got %v", got)
	}

	// Large line (>64KB), non-control_response with matching request_id, and broken JSON with control_response.
	largeLine := `{"type":"system","padding":"` + strings.Repeat("x", 70*1024) + `"}`
	wrongType := `{"type":"other","response":{"subtype":"success","request_id":"r1","response":{"models":[{"value":"bad"}]}}}`
	brokenJSON := `{"type":"control_response","response":bad}`
	list, err := Parse(strings.NewReader(strings.Join([]string{
		largeLine,
		wrongType,
		brokenJSON,
		readFixture(t),
	}, "\n")), "r1")
	if err != nil || len(list) != 13 {
		t.Fatalf("Parse with prefix lines = (%d, %v), want (13, nil)", len(list), err)
	}

	// Scanner read error.
	readErr := errors.New("read failed")
	if _, err := Parse(iotest.ErrReader(readErr), "r1"); !errors.Is(err, readErr) {
		t.Errorf("Parse ErrReader = %v, want %v", err, readErr)
	}
}
