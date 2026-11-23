package tasktools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/todos/backend/filter"
	"github.com/jiva-studio/numen/plugins/todos/backend/todo"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/complete"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/edit"
)

const (
	fieldsSentence   = "A task is a line `- [ ] text` of a note; its fields are written in the text as todo.txt does: `due:YYYY-MM-DD`, `rec:1w`, `pri:A` or a leading `(A)`, `+project` and `@context`."
	lineSentence     = "A task is addressed by its note path and its line from task_list; a write is refused when the line is no longer that task."
	expectedSentence = "expectedText is required: pass the text of the task from task_list, so a line number gone stale never changes another task."
	conflictSentence = "A write goes through the note with the conflict check of a tick in the editor, so a note changed meanwhile is refused or retried once."
)

// TaskListInput selects the tasks task_list returns.
type TaskListInput struct {
	Status   string `json:"status,omitempty" jsonschema:"open (default), done or all"`
	Due      string `json:"due,omitempty" jsonschema:"any (default), none, overdue, today, thisWeek or range; none, overdue, today and thisWeek select open tasks"`
	From     string `json:"from,omitempty" jsonschema:"with due range: the first due date as YYYY-MM-DD, included"`
	To       string `json:"to,omitempty" jsonschema:"with due range: the last due date as YYYY-MM-DD, included"`
	Project  string `json:"project,omitempty" jsonschema:"only tasks of this project, written without the plus sign"`
	Context  string `json:"context,omitempty" jsonschema:"only tasks of this context, written without the at sign"`
	Priority string `json:"priority,omitempty" jsonschema:"only tasks of this priority, one letter A to Z"`
	Path     string `json:"path,omitempty" jsonschema:"only tasks of this note, or of the notes under this folder when it ends in a slash"`
	Text     string `json:"text,omitempty" jsonschema:"only tasks whose text contains this, compared without case"`
	Limit    int    `json:"limit,omitempty" jsonschema:"the most tasks to return, 100 by default and at most 500"`
}

// TaskListOutput is the page of tasks task_list returns.
type TaskListOutput struct {
	Tasks []TaskOutput `json:"tasks" jsonschema:"the tasks ordered by due date (none last), then priority, then path and line"`
	Total int          `json:"total" jsonschema:"how many tasks match before the limit"`
}

func (t *Tools) addTaskList(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:        "task_list",
		Title:       "List tasks",
		Description: "List the tasks written in the notes of the vault, with their path, line, text, status, due date, priority, projects and contexts. The filters combine with AND. " + fieldsSentence,
		Effect:      tools.ReadOnly,
	}, func(ctx context.Context, in TaskListInput) (TaskListOutput, error) {
		scope, err := t.getScope(ctx)
		if err != nil {
			return TaskListOutput{}, err
		}
		f, err := readFilter(in)
		if err != nil {
			return TaskListOutput{}, err
		}
		listing := scope.Tasks.List(t.clock.Now().Format(time.DateOnly), f, filter.ClampLimit(in.Limit, defaultLimit, maxLimit))
		out := TaskListOutput{Tasks: make([]TaskOutput, 0, len(listing.Tasks)), Total: listing.Total}
		for _, task := range listing.Tasks {
			out.Tasks = append(out.Tasks, encodeTask(task))
		}
		return out, nil
	})
}

func readFilter(in TaskListInput) (filter.Filter, error) {
	var f filter.Filter
	if err := readStatusAndDue(&f, in); err != nil {
		return f, err
	}
	if err := readFields(&f, in); err != nil {
		return f, err
	}
	f.Path, f.Text = in.Path, in.Text
	return f, nil
}

func readStatusAndDue(f *filter.Filter, in TaskListInput) error {
	switch in.Status {
	case "", "open":
		f.Status = []todo.Status{todo.StatusOpen}
	case "done":
		f.Status = []todo.Status{todo.StatusDone}
	case "all":
		f.Status = []todo.Status{todo.StatusOpen, todo.StatusDone, todo.StatusOther}
	default:
		return fmt.Errorf("status %q is not open, done or all", in.Status)
	}
	switch due := filter.DueFilter(in.Due); due {
	case "", filter.DueAny, filter.DueNone, filter.DueOverdue, filter.DueToday, filter.DueThisWeek, filter.DueRange:
		f.Due = due
	default:
		return fmt.Errorf("due %q is not any, none, overdue, today, thisWeek or range", in.Due)
	}
	if err := checkRange(in); err != nil {
		return err
	}
	f.From, f.To = in.From, in.To
	return nil
}

func readFields(f *filter.Filter, in TaskListInput) error {
	if in.Project != "" {
		f.Projects = []string{strings.TrimPrefix(in.Project, "+")}
	}
	if in.Context != "" {
		f.Contexts = []string{strings.TrimPrefix(in.Context, "@")}
	}
	if in.Priority != "" {
		letter, isLetter := filter.NormalizePriority(in.Priority)
		if !isLetter {
			return fmt.Errorf("priority %q is not one letter from A to Z", in.Priority)
		}
		f.Priorities = []string{letter}
	}
	return nil
}

func checkRange(in TaskListInput) error {
	if in.Due != string(filter.DueRange) && (in.From != "" || in.To != "") {
		return errors.New("from and to need due range")
	}
	for _, date := range []string{in.From, in.To} {
		if date != "" && !todo.IsDate(date) {
			return fmt.Errorf("%q is not a YYYY-MM-DD date", date)
		}
	}
	return nil
}

// TaskAddInput names the note and the task text to append.
type TaskAddInput struct {
	Path string `json:"path" jsonschema:"the path of an existing note, relative to the vault folder"`
	Text string `json:"text" jsonschema:"the task text with its fields, on one line, without the checkbox"`
}

func (t *Tools) addTaskAdd(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:         "task_add",
		Title:        "Add a task",
		Description:  "Append an open task line `- [ ] text` to the end of an existing note.",
		Effect:       tools.ReadWrite,
		Instructions: []string{fieldsSentence, conflictSentence},
	}, func(ctx context.Context, in TaskAddInput) (TaskOutput, error) {
		scope, err := t.getScope(ctx)
		if err != nil {
			return TaskOutput{}, err
		}
		task, err := scope.Editor.Add(ctx, in.Path, in.Text)
		return encodeTask(task), describe(in.Path, err)
	})
}

// TaskSetDoneInput names the task and the state to set.
type TaskSetDoneInput struct {
	Path         string `json:"path" jsonschema:"the path of the note"`
	Line         int    `json:"line" jsonschema:"the line of the task from task_list"`
	ExpectedText string `json:"expectedText" jsonschema:"required: the text of the task from task_list; the change is refused when the task now has other text"`
	IsDone       bool   `json:"done" jsonschema:"true ticks an open task and adds done: today, false unticks a done task"`
}

func (t *Tools) addTaskSetDone(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:         "task_set_done",
		Title:        "Tick or untick a task",
		Description:  "Tick an open task, which adds done: with today's date, or untick a done one. Only the checkbox and the done: field change.",
		Effect:       tools.ReadWrite,
		Instructions: []string{lineSentence, expectedSentence, conflictSentence},
	}, func(ctx context.Context, in TaskSetDoneInput) (TaskOutput, error) {
		scope, err := t.getScope(ctx)
		if err != nil {
			return TaskOutput{}, err
		}
		found, err := scope.Editor.Find(ctx, in.Path, in.Line, in.ExpectedText)
		if err != nil {
			return TaskOutput{}, describe(in.Path, err)
		}
		result, err := scope.Completer.Execute(ctx, complete.Request{
			Path: in.Path, Line: in.Line, IsProseLine: false, Raw: found.Raw, IsDone: in.IsDone, Today: t.clock.Now().Format(time.DateOnly),
		})
		if err != nil {
			return TaskOutput{}, err
		}
		if result.Refusal != "" {
			return TaskOutput{}, newRefusalError(result.Refusal)
		}
		result.Task.Path = in.Path
		return encodeTask(result.Task), nil
	})
}

// TaskEditInput names the task and its new text.
type TaskEditInput struct {
	Path         string `json:"path" jsonschema:"the path of the note"`
	Line         int    `json:"line" jsonschema:"the line of the task from task_list"`
	Text         string `json:"text" jsonschema:"the new text with its fields, on one line, replacing everything after the checkbox"`
	ExpectedText string `json:"expectedText" jsonschema:"required: the text of the task from task_list; the edit is refused when the task now has other text"`
}

func (t *Tools) addTaskEdit(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:         "task_edit",
		Title:        "Edit a task",
		Description:  "Replace the text of a task, its fields included, and keep its checkbox state, indentation and list marker.",
		Effect:       tools.ReadWrite,
		Instructions: []string{fieldsSentence, lineSentence, expectedSentence, conflictSentence},
	}, func(ctx context.Context, in TaskEditInput) (TaskOutput, error) {
		scope, err := t.getScope(ctx)
		if err != nil {
			return TaskOutput{}, err
		}
		task, err := scope.Editor.Edit(ctx, in.Path, in.Line, in.ExpectedText, in.Text)
		return encodeTask(task), describe(in.Path, err)
	})
}

// TaskRemoveInput names the task to delete and the text it must still have.
type TaskRemoveInput struct {
	Path         string `json:"path" jsonschema:"the path of the note"`
	Line         int    `json:"line" jsonschema:"the line of the task from task_list"`
	ExpectedText string `json:"expectedText" jsonschema:"required: the text of the task from task_list; the removal is refused when the task now has other text"`
}

func (t *Tools) addTaskRemove(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:         "task_remove",
		Title:        "Remove a task",
		Description:  "Delete the line of a task from its note. The line cannot be restored, so list the task first.",
		Effect:       tools.ReadWrite,
		Instructions: []string{lineSentence, expectedSentence, conflictSentence},
	}, func(ctx context.Context, in TaskRemoveInput) (TaskOutput, error) {
		scope, err := t.getScope(ctx)
		if err != nil {
			return TaskOutput{}, err
		}
		task, err := scope.Editor.Remove(ctx, in.Path, in.Line, in.ExpectedText)
		return encodeTask(task), describe(in.Path, err)
	})
}

func describe(path string, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, edit.ErrNoNote):
		return fmt.Errorf("%s is not a note of the vault", path)
	case errors.Is(err, edit.ErrNotTask):
		return errors.New("the line is not a task; list the tasks again")
	case errors.Is(err, edit.ErrNoExpected):
		return errors.New("expectedText is required: pass the text of the task from task_list")
	case errors.Is(err, edit.ErrChanged):
		return errors.New("the task text is not the expected text; list the tasks again")
	}
	return err
}
