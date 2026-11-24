// Package tasktools provides the MCP tools that list, add, tick, edit and remove the tasks of notes.
package tasktools

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/todos/backend/filter"
	"github.com/jiva-studio/numen/plugins/todos/backend/ports"
	"github.com/jiva-studio/numen/plugins/todos/backend/todo"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/complete"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/index"
)

const (
	defaultLimit = 100
	maxLimit     = 500
)

// Tasks lists the tasks of the index.
type Tasks interface {
	List(today string, f filter.Filter, limit int) index.Listing
}

// Editor changes the task lines of a note.
type Editor interface {
	Find(ctx context.Context, path string, line int, expected string) (todo.Task, error)
	Add(ctx context.Context, path, text string) (todo.Task, error)
	Edit(ctx context.Context, path string, line int, expected, text string) (todo.Task, error)
	Remove(ctx context.Context, path string, line int, expected string) (todo.Task, error)
}

// Completer ticks and unticks a task line.
type Completer interface {
	Execute(ctx context.Context, req complete.Request) (complete.Result, error)
}

// Scope is what the tools use of one vault: its tasks, its editor and the tick of its checkboxes.
type Scope struct {
	Tasks     Tasks
	Editor    Editor
	Completer Completer
}

// Vaults opens the scope of the vault a call names. It refuses an empty identity.
type Vaults interface {
	GetScope(ctx context.Context, vaultID string) (Scope, error)
}

// Tools provides the task tools over the vault each call names.
type Tools struct {
	vaults Vaults
	clock  ports.Clock
}

// New returns the tools; the clock gives the day the due filters compare with.
func New(vaults Vaults, clock ports.Clock) *Tools {
	return &Tools{vaults: vaults, clock: clock}
}

func (t *Tools) getScope(ctx context.Context) (Scope, error) {
	return t.vaults.GetScope(ctx, tools.GetVault(ctx))
}

func (t *Tools) BuildRegistry() (*tools.Registry, error) {
	registry := tools.New()
	for _, add := range []func(*tools.Registry) error{t.addTaskList, t.addTaskAdd, t.addTaskSetDone, t.addTaskEdit, t.addTaskRemove} {
		if err := add(registry); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

// TaskOutput is one task as the tools return it.
type TaskOutput struct {
	Path     string   `json:"path" jsonschema:"the path of the note"`
	Line     int      `json:"line" jsonschema:"the 1-based line of the task in the note, frontmatter included"`
	Text     string   `json:"text" jsonschema:"the text of the task without its fields, projects and contexts"`
	Status   string   `json:"status" jsonschema:"open, done or other, from the character of the checkbox"`
	Due      string   `json:"due" jsonschema:"the due date as YYYY-MM-DD, empty when the task has none"`
	Priority string   `json:"priority" jsonschema:"the priority letter A to Z, empty when the task has none"`
	Projects []string `json:"projects" jsonschema:"the projects of the task without their plus sign"`
	Contexts []string `json:"contexts" jsonschema:"the contexts of the task without their at sign"`
}

func encodeTask(task todo.Task) TaskOutput {
	return TaskOutput{
		Path: task.Path, Line: task.Line, Text: task.Text, Status: string(task.Status), Due: task.Due,
		Priority: task.Priority, Projects: append([]string{}, task.Projects...), Contexts: append([]string{}, task.Contexts...),
	}
}

func newRefusalError(refusal complete.Refusal) error {
	switch refusal {
	case complete.RefusalNotFound:
		return errors.New("the line is no longer that task; list the tasks again")
	case complete.RefusalUnreadable:
		return errors.New("the note frontmatter cannot be read")
	case complete.RefusalStale:
		return errors.New("the note changed while the task was written; try again")
	case complete.RefusalInvalidRecurrence:
		return errors.New("the task has a recurrence that cannot be ticked yet")
	case complete.RefusalNotOpen:
		return errors.New("the task is not open")
	case complete.RefusalNotDone:
		return errors.New("the task is not done")
	}
	return fmt.Errorf("the change was refused: %s", refusal)
}
