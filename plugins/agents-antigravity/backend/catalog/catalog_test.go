package catalog

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/model"
)

type fakeProgram struct {
	installed bool
	items     []model.Model
	err       error
	commands  [][]string
}

func (f *fakeProgram) IsInstalled(string) bool { return f.installed }

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

func TestProgramCommandIsTheOneOfAntigravity(t *testing.T) {
	cmds := []agentrun.ProgramCommand{
		{Program: "other", Command: []string{"other-bin"}},
		{Program: "antigravity", Command: []string{"agy-bin", "--arg"}},
	}
	if got := getProgramCommand(cmds); !slices.Equal(got, []string{"agy-bin", "--arg"}) {
		t.Errorf("getProgramCommand(antigravity) = %v", got)
	}
	if got := getProgramCommand(cmds[:1]); got != nil {
		t.Errorf("getProgramCommand(none) = %v, want nil", got)
	}
}

func TestTheStateNamesAMissingProgram(t *testing.T) {
	missing := New(&fakeProgram{}, &fakeProgram{}, false)
	if got := missing.GetState(""); got.Unreachable == "" {
		t.Errorf("state of a missing program = %+v", got)
	}
	installed := New(&fakeProgram{}, &fakeProgram{installed: true}, false)
	if got := installed.GetState(""); got.Unreachable != "" {
		t.Errorf("state of an installed program = %+v", got)
	}
}

func TestTheProgramListsTheModelsOfTheExecutablePath(t *testing.T) {
	program := &fakeProgram{installed: true, items: []model.Model{{Name: "m-custom", Title: "Custom"}}}

	got := New(program, program, false).GetPrograms(t.Context(), "/opt/agy", nil)

	if len(got) != 1 || !got[0].IsInstalled || len(got[0].Models) != 1 {
		t.Fatalf("the programs are %+v", got)
	}
	if len(program.commands) != 1 || !slices.Equal(program.commands[0], []string{"/opt/agy"}) {
		t.Errorf("the models were items with %v", program.commands)
	}
}

func TestAProgramWhoseListFailsHasNoModels(t *testing.T) {
	program := &fakeProgram{installed: true, err: errors.New("no list")}

	got := New(program, program, false).GetPrograms(t.Context(), "", nil)

	if len(got) != 1 || !got[0].IsInstalled || len(got[0].Models) != 0 {
		t.Errorf("the programs are %+v", got)
	}
}

func TestADefaultModelIsNotResolved(t *testing.T) {
	program := &fakeProgram{installed: true}
	req := &agentrun.Request{Model: model.DefaultName, Reasoning: "high"}

	New(program, program, false).ResolveModel(t.Context(), req)

	if req.Model != model.DefaultName || len(program.commands) != 0 {
		t.Errorf("the model is %q after listing with %v", req.Model, program.commands)
	}
}
