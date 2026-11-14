// Package filter selects and orders tasks. It is the one engine the chip, the list, the agenda and
// the calendar share, and the one a later tasks block parses its text into.
package filter

import (
	"slices"
	"strings"

	"github.com/jiva-studio/numen/plugins/views-todos/backend/todo"
)

// DueFilter says which due dates a filter accepts. The empty value accepts any.
type DueFilter string

const (
	// DueAny accepts every task.
	DueAny DueFilter = "any"
	// DueNone accepts open tasks with no due date.
	DueNone DueFilter = "none"
	// DueOverdue accepts open tasks due before today.
	DueOverdue DueFilter = "overdue"
	// DueToday accepts open tasks due today.
	DueToday DueFilter = "today"
	// DueThisWeek accepts open tasks due in the seven days from today.
	DueThisWeek DueFilter = "thisWeek"
	// DueRange accepts tasks due from From to To, both included.
	DueRange DueFilter = "range"
)

// daysOfWeek is the length of the thisWeek range.
const daysOfWeek = 7

// Filter is a value, not a query language. Its fields combine with AND. An empty Status list means
// open. Path is a note path, or a folder prefix that ends in a slash. Text is a case-insensitive
// substring of the display text.
type Filter struct {
	Status     []todo.Status
	Due        DueFilter
	From, To   string
	Projects   []string
	Contexts   []string
	Priorities []string
	Path       string
	Text       string
}

// Apply returns the tasks the filter accepts, ordered by due date (tasks with none last), then
// priority A to Z (none last), then path, then line. Today is a YYYY-MM-DD date.
func Apply(tasks []todo.Task, f Filter, today string) []todo.Task {
	accepted := []todo.Task{}
	for _, task := range tasks {
		if f.accepts(task, today) {
			accepted = append(accepted, task)
		}
	}
	slices.SortStableFunc(accepted, compare)
	return accepted
}

func (f Filter) accepts(task todo.Task, today string) bool {
	return f.isStatusAccepted(task) &&
		f.isDueAccepted(task, today) &&
		isAnyOf(f.Projects, task.Projects) &&
		isAnyOf(f.Contexts, task.Contexts) &&
		isAnyOf(f.Priorities, []string{task.Priority}) &&
		f.isPathAccepted(task) &&
		strings.Contains(strings.ToLower(task.Text), strings.ToLower(f.Text))
}

func (f Filter) isStatusAccepted(task todo.Task) bool {
	if len(f.Status) == 0 {
		return task.Status == todo.StatusOpen
	}
	return slices.Contains(f.Status, task.Status)
}

func (f Filter) isDueAccepted(task todo.Task, today string) bool {
	switch f.Due {
	case DueNone, DueOverdue, DueToday, DueThisWeek:
		return task.Status == todo.StatusOpen && isDueMatched(f.Due, task.Due, today)
	case DueRange:
		return task.Due != "" && (f.From == "" || task.Due >= f.From) && (f.To == "" || task.Due <= f.To)
	case DueAny:
	}
	return true
}

// isDueMatched reports whether a due date, empty for none, is the one a filter of an open task asks for.
func isDueMatched(kind DueFilter, due, today string) bool {
	switch kind {
	case DueNone:
		return due == ""
	case DueOverdue:
		return due != "" && due < today
	case DueToday:
		return due == today
	case DueThisWeek:
		return due >= today && due <= todo.AddDays(today, daysOfWeek-1)
	case DueAny, DueRange:
	}
	return false
}

func (f Filter) isPathAccepted(task todo.Task) bool {
	switch {
	case f.Path == "":
		return true
	case strings.HasSuffix(f.Path, "/"):
		return strings.HasPrefix(task.Path, f.Path)
	}
	return task.Path == f.Path
}

// isAnyOf reports whether no value is wanted or one of the wanted values is among the values.
func isAnyOf(wanted, values []string) bool {
	if len(wanted) == 0 {
		return true
	}
	return slices.ContainsFunc(values, func(value string) bool { return slices.Contains(wanted, value) })
}

// compare orders by due date, priority, path and line; an empty due date or priority is last.
func compare(a, b todo.Task) int {
	if order := compareFilled(a.Due, b.Due); order != 0 {
		return order
	}
	if order := compareFilled(a.Priority, b.Priority); order != 0 {
		return order
	}
	if order := strings.Compare(a.Path, b.Path); order != 0 {
		return order
	}
	return a.Line - b.Line
}

func compareFilled(a, b string) int {
	switch {
	case a == b:
		return 0
	case a == "":
		return 1
	case b == "":
		return -1
	}
	return strings.Compare(a, b)
}
