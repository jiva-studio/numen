// Package queryfns answers the query functions of the todos manifest: each is a condition on a
// note that holds when the note has a task the filter engine accepts.
package queryfns

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/searchfunction"
	"github.com/jiva-studio/numen/plugins/todos/backend/filter"
	"github.com/jiva-studio/numen/plugins/todos/backend/ports"
	"github.com/jiva-studio/numen/plugins/todos/backend/todo"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/index"
)

const notReadyWarning = "the task index is still being built, so the answer may be incomplete"

// Vaults opens the tasks of the vault a call names.
type Vaults interface {
	GetTasks(ctx context.Context, vaultID string) (Tasks, error)
}

// Tasks lists the notes that hold a task a filter accepts.
type Tasks interface {
	ListPaths(today string, f filter.Filter) index.PathListing
	MatchPaths(today string, f filter.Filter, asked []string) index.PathListing
}

// buildFilter returns the filter a call stands for, at today.
type buildFilter func(call searchfunction.Call, today string) (filter.Filter, error)

// New returns the registry of the query functions over the tasks of the vault each call names and the day of the clock.
func New(vaults Vaults, clock ports.Clock) (*searchfunction.Registry, error) {
	registry := searchfunction.New()
	for name, build := range createBuilders() {
		if err := registry.Add(name, newFunction(build, vaults, clock)); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func createBuilders() map[string]buildFilter {
	return map[string]buildFilter{
		"hasOpenTask":       createFixedBuilder(todo.StatusOpen, filter.DueAny),
		"hasDoneTask":       createFixedBuilder(todo.StatusDone, filter.DueAny),
		"hasTaskOverdue":    createFixedBuilder(todo.StatusOpen, filter.DueOverdue),
		"hasTaskDueToday":   createFixedBuilder(todo.StatusOpen, filter.DueToday),
		"hasTaskWithoutDue": createFixedBuilder(todo.StatusOpen, filter.DueNone),
		"hasTaskDueWithin":  buildDueWithin,
		"hasTaskDueBefore":  buildDueBefore,
		"hasTaskProject":    buildField("hasTaskProject", "name", func(f *filter.Filter, name string) { f.Projects = []string{name} }),
		"hasTaskContext":    buildField("hasTaskContext", "name", func(f *filter.Filter, name string) { f.Contexts = []string{name} }),
		"hasTaskPriority":   buildPriority,
		"hasTaskText":       buildText,
	}
}

func createFixedBuilder(status todo.Status, due filter.DueFilter) buildFilter {
	return func(_ searchfunction.Call, _ string) (filter.Filter, error) {
		var f filter.Filter
		f.Status = []todo.Status{status}
		f.Due = due
		return f, nil
	}
}

func buildDueWithin(call searchfunction.Call, today string) (filter.Filter, error) {
	days, isGiven := call.Arguments.GetInt("days")
	if !isGiven || days < 0 {
		return filter.Filter{}, newInvalidError("hasTaskDueWithin needs days, a number of days from 0")
	}
	var f filter.Filter
	f.Status = []todo.Status{todo.StatusOpen}
	f.Due = filter.DueRange
	f.From = today
	f.To = todo.AddDays(today, int(min(days, maxDays)))
	return f, nil
}

// maxDays keeps today plus the days inside the calendar the date arithmetic reads.
const maxDays = 365 * 1000

func buildDueBefore(call searchfunction.Call, _ string) (filter.Filter, error) {
	date, _ := call.Arguments.GetString("date")
	date = strings.TrimSpace(date)
	if !todo.IsDate(date) {
		return filter.Filter{}, newInvalidError("hasTaskDueBefore needs date as YYYY-MM-DD, got %q", date)
	}
	var f filter.Filter
	f.Status = []todo.Status{todo.StatusOpen, todo.StatusDone}
	f.Due = filter.DueRange
	f.To = date
	return f, nil
}

func buildField(function, argument string, set func(*filter.Filter, string)) buildFilter {
	return func(call searchfunction.Call, _ string) (filter.Filter, error) {
		value, _ := call.Arguments.GetString(argument)
		value = strings.TrimLeft(strings.TrimSpace(value), "+@")
		if value == "" {
			return filter.Filter{}, newInvalidError("%s needs a %s", function, argument)
		}
		var f filter.Filter
		set(&f, value)
		return f, nil
	}
}

func buildPriority(call searchfunction.Call, _ string) (filter.Filter, error) {
	letter, _ := call.Arguments.GetString("letter")
	normal, isLetter := filter.NormalizePriority(letter)
	if !isLetter {
		return filter.Filter{}, newInvalidError("hasTaskPriority needs letter, one letter from A to Z, got %q", strings.TrimSpace(letter))
	}
	var f filter.Filter
	f.Priorities = []string{normal}
	return f, nil
}

func buildText(call searchfunction.Call, _ string) (filter.Filter, error) {
	text, _ := call.Arguments.GetString("text")
	if strings.TrimSpace(text) == "" {
		return filter.Filter{}, newInvalidError("hasTaskText needs text")
	}
	var f filter.Filter
	f.Text = strings.TrimSpace(text)
	return f, nil
}

func newInvalidError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", searchfunction.ErrInvalidArguments, fmt.Sprintf(format, args...))
}

func newFunction(build buildFilter, vaults Vaults, clock ports.Clock) searchfunction.Function {
	list := func(ctx context.Context, call searchfunction.Call, query func(tasks Tasks, today string, f filter.Filter) index.PathListing) ([]string, string, error) {
		tasks, err := vaults.GetTasks(ctx, call.VaultID)
		if err != nil {
			return nil, "", err
		}
		today := clock.Now().Format(time.DateOnly)
		f, err := build(call, today)
		if err != nil {
			return nil, "", err
		}
		listing := query(tasks, today, f)
		if !listing.IsReady {
			return listing.Paths, notReadyWarning, nil
		}
		return listing.Paths, "", nil
	}
	return searchfunction.Function{
		Match: func(ctx context.Context, call searchfunction.Call, asked []string) (searchfunction.Matched, error) {
			paths, warning, err := list(ctx, call, func(tasks Tasks, today string, f filter.Filter) index.PathListing {
				return tasks.MatchPaths(today, f, asked)
			})
			if err != nil {
				return searchfunction.Matched{}, err
			}
			return searchfunction.Matched{Paths: paths, Warning: warning}, nil
		},
		Find: func(ctx context.Context, call searchfunction.Call, limit int) (searchfunction.Found, error) {
			paths, warning, err := list(ctx, call, func(tasks Tasks, today string, f filter.Filter) index.PathListing {
				return tasks.ListPaths(today, f)
			})
			if err != nil {
				return searchfunction.Found{}, err
			}
			return searchfunction.Found{Paths: paths[:min(limit, len(paths))], EstimatedCount: int64(len(paths)), Warning: warning}, nil
		},
	}
}
