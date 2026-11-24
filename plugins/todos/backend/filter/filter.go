// Package filter selects and orders tasks. It is the one engine the chip, the list, the agenda and
// the calendar share, and the one a later tasks block parses its text into.
package filter

import (
	"slices"
	"strings"

	"github.com/jiva-studio/numen/plugins/todos/backend/todo"
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
	// DueNextSevenDays accepts open tasks due in the six days after today.
	DueNextSevenDays DueFilter = "nextSevenDays"
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
		if f.isAccepted(task, today) {
			accepted = append(accepted, task)
		}
	}
	slices.SortStableFunc(accepted, compare)
	return accepted
}

// IsMatch reports whether the filter accepts at least one of the tasks, without ordering them.
func IsMatch(tasks []todo.Task, f Filter, today string) bool {
	return slices.ContainsFunc(tasks, func(task todo.Task) bool { return f.isAccepted(task, today) })
}

// ClampLimit returns the count a caller asked for, the default when it asked for none and at most max.
func ClampLimit(asked, defaultLimit, maxLimit int) int {
	switch {
	case asked <= 0:
		return defaultLimit
	case asked > maxLimit:
		return maxLimit
	}
	return asked
}

// NormalizePriority returns the priority letter in upper case, or false when it is not one letter A to Z.
func NormalizePriority(letter string) (string, bool) {
	letter = strings.ToUpper(strings.TrimSpace(letter))
	if len(letter) != 1 || letter[0] < 'A' || letter[0] > 'Z' {
		return "", false
	}
	return letter, true
}

func (f Filter) isAccepted(task todo.Task, today string) bool {
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
	case DueNone, DueOverdue, DueToday, DueThisWeek, DueNextSevenDays:
		return task.Status == todo.StatusOpen && IsDueMatched(f.Due, task.Due, today)
	case DueRange:
		return task.Due != "" && (f.From == "" || task.Due >= f.From) && (f.To == "" || task.Due <= f.To)
	case DueAny:
	}
	return true
}

// IsDueMatched reports whether a due date, empty for none, falls in the group kind names at today.
func IsDueMatched(kind DueFilter, due, today string) bool {
	switch kind {
	case DueNone:
		return due == ""
	case DueOverdue:
		return due != "" && due < today
	case DueToday:
		return due == today
	case DueThisWeek:
		return due >= today && due <= todo.AddDays(today, daysOfWeek-1)
	case DueNextSevenDays:
		return due > today && due <= todo.AddDays(today, daysOfWeek-1)
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

func isAnyOf(wanted, values []string) bool {
	if len(wanted) == 0 {
		return true
	}
	return slices.ContainsFunc(values, func(value string) bool { return slices.Contains(wanted, value) })
}

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
