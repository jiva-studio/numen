package catalog

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/model"
)

// fakeProgram is the outer edge of the catalog: the program that lists the models.
type fakeProgram struct {
	items    []model.Model
	err      error
	commands [][]string
}

func (f *fakeProgram) List(_ context.Context, command []string) ([]model.Model, error) {
	f.commands = append(f.commands, command)
	return f.items, f.err
}

func TestTheCacheKeepsTheListFor15SecondsAndTheFailureFor30(t *testing.T) {
	if modelListLimit != 15*time.Second {
		t.Errorf("modelListLimit = %v, want 15s", modelListLimit)
	}
	if failureBackoff != 30*time.Second {
		t.Errorf("failureBackoff = %v, want 30s", failureBackoff)
	}
}

func TestTheModelsOfTheProgramAreListedFoldedByLevel(t *testing.T) {
	program := &fakeProgram{items: []model.Model{
		{Name: "m-high", Title: "Model (High)"}, {Name: "m-low", Title: "Model (Low)"}, {Name: "m-custom", Title: "Custom"},
	}}

	got := New(program, false).List(t.Context(), "/opt/agy")

	if len(got) != 2 || got[0].Name != "m" || len(got[0].Levels) != 2 || got[1].Name != "m-custom" {
		t.Fatalf("the models are %+v", got)
	}
	if len(program.commands) != 1 || !slices.Equal(program.commands[0], []string{"/opt/agy"}) {
		t.Errorf("the models were listed with %v", program.commands)
	}
}

func TestAProgramWhoseListFailsHasNoModels(t *testing.T) {
	program := &fakeProgram{err: errors.New("no list")}

	if got := New(program, false).List(t.Context(), "/opt/agy"); len(got) != 0 {
		t.Errorf("the models are %+v", got)
	}
}

func TestAModelAndALevelResolveToTheListedModel(t *testing.T) {
	program := &fakeProgram{items: []model.Model{{Name: "m-high", Title: "Model (High)"}, {Name: "m-low", Title: "Model (Low)"}}}

	if got := New(program, false).ResolveModel(t.Context(), "/opt/agy", "m", "low"); got != "m-low" {
		t.Errorf("resolved to %q, want m-low", got)
	}
}

func TestADefaultModelIsNotResolved(t *testing.T) {
	program := &fakeProgram{}

	got := New(program, false).ResolveModel(t.Context(), "/opt/agy", model.DefaultName, "high")

	if got != model.DefaultName || len(program.commands) != 0 {
		t.Errorf("the model is %q after listing with %v", got, program.commands)
	}
}
