// Package complete ticks or unticks one task line of a note and writes the note over the file
// stat it had when it was read. It is the one write path of the plugin.
package complete

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/todos/backend/ports"
	"github.com/jiva-studio/numen/plugins/todos/backend/todo"
)

// Refusal is why a change was not written: the person can fix it by looking at the note.
type Refusal string

const (
	// RefusalNotFound is a note with no line of the text the request names.
	RefusalNotFound Refusal = "notFound"
	// RefusalChanged is a note whose frontmatter can no longer be read.
	RefusalChanged Refusal = "changed"
	// RefusalStale is a note that changed between the read and the write, twice.
	RefusalStale Refusal = "stale"
	// RefusalInvalidRecurrence is a tick of a task whose rec: cannot be read.
	RefusalInvalidRecurrence Refusal = "invalidRecurrence"
	// RefusalNotOpen is a tick of a task that is not open.
	RefusalNotOpen Refusal = "notOpen"
	// RefusalNotDone is an untick of a task that is open.
	RefusalNotDone Refusal = "notDone"
)

// Request names the task line to change: the note, the line number and the text it had when it
// was listed, the change and the day.
type Request struct {
	Path   string
	Line   int
	Raw    string
	IsDone bool
	Today  string
}

// Result is the task after the change, the bytes written and the file stat after the write. A
// refused change has a Refusal and nothing else.
type Result struct {
	Task    todo.Task
	Change  todo.Change
	Refusal Refusal
	Content []byte
	Stat    domain.FileStat
}

// SetTaskDone changes one task line of a note. Two changes of one note run one after the other.
type SetTaskDone struct {
	files  ports.FileReader
	writer ports.FileWriter
}

// New returns the use case over the notes the reader and the writer reach.
func New(files ports.FileReader, writer ports.FileWriter) *SetTaskDone {
	return &SetTaskDone{files: files, writer: writer}
}

// Execute reads the note, finds the line, changes it and writes the note over the read baseline.
// A host failure is the error; a change the note does not allow is the Refusal of the result.
func (u *SetTaskDone) Execute(ctx context.Context, req Request) (Result, error) {
	return Result{}, errors.New("not implemented")
}
