// Package filter selects and orders tasks. It is the one engine the chip, the list, the agenda and
// the calendar share, and the one a later tasks block parses its text into.
package filter

import "github.com/jiva-studio/numen/plugins/views-todos/backend/todo"

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
	// DueThisWeek accepts open tasks due in the seven days from the week start.
	DueThisWeek DueFilter = "thisWeek"
	// DueRange accepts tasks due from From to To, both included.
	DueRange DueFilter = "range"
)

// Filter is a value, not a query language. Its fields combine with AND.
type Filter struct {
	// Status lists open, done and other; an empty list means open.
	Status     []todo.Status
	Due        DueFilter
	From, To   string
	Projects   []string
	Contexts   []string
	Priorities []string
	// Path is a note path, or a folder prefix that ends in a slash.
	Path string
	// Text is a case-insensitive substring of the display text.
	Text string
}

// Apply returns the tasks the filter accepts, ordered by due date (tasks with none last), then
// priority A to Z (none last), then path, then line. Today is a YYYY-MM-DD date.
func Apply(tasks []todo.Task, f Filter, today string) []todo.Task {
	return nil
}
