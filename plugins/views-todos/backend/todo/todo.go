// Package todo reads the tasks written in a note: the lines that start with a checkbox, and the
// fields todo.txt puts in them. It takes bytes and returns values; it reads no file and no clock.
package todo

import "errors"

// ErrUnreadable is what a note gets when its frontmatter cannot be read or is never closed. Such a
// note has no tasks.
var ErrUnreadable = errors.New("the note cannot be read")

// Status is what the character of a checkbox says.
type Status string

const (
	// StatusOpen is a checkbox holding a space.
	StatusOpen Status = "open"
	// StatusDone is a checkbox holding x or X.
	StatusDone Status = "done"
	// StatusOther is a checkbox holding any other character, such as - or /.
	StatusOther Status = "other"
)

// Issue is a field of a task line that is written but cannot be read.
type Issue string

const (
	// IssueInvalidDue is a due: that is not a calendar date.
	IssueInvalidDue Issue = "invalidDue"
	// IssueInvalidDone is a done: that is not a calendar date.
	IssueInvalidDone Issue = "invalidDone"
	// IssueInvalidRecurrence is a rec: that is not a count and a unit.
	IssueInvalidRecurrence Issue = "invalidRecurrence"
	// IssueInvalidPriority is a pri: that is not a letter from A to Z.
	IssueInvalidPriority Issue = "invalidPriority"
)

// Task is one task line of a note. ParseNote leaves Path empty and the index sets it. Line is the
// 1-based number of the line in the file, frontmatter included. Raw is the whole line without its
// line ending, which a write compares. Text is the text after the checkbox without the fields, the
// leading priority, the projects and the contexts. Depth is the indentation width divided by 2,
// rounded down.
type Task struct {
	Path       string
	Line       int
	Raw        string
	Text       string
	Status     Status
	Priority   string
	Due        string
	DoneOn     string
	Recurrence string
	Projects   []string
	Contexts   []string
	Issues     []Issue
	Depth      int
}

// ParseNote returns the tasks of a note in the order of its lines.
func ParseNote(_ []byte) ([]Task, error) {
	return nil, nil
}
