// Package complete ticks or unticks one task line of a note and writes the note over the file
// stat it had when it was read. It is the one write path of the plugin.
package complete

import (
	"bytes"
	"context"
	"errors"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
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
	// RefusalInvalidRecurrence is a tick of a task whose rec: cannot be read, and for now of any task
	// with a rec:, until a tick moves the due date of a recurring task.
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

// SetTaskDone changes one task line of a note. Two changes run one after the other, so the second
// finds its line by its text in the note the first wrote.
type SetTaskDone struct {
	files  ports.FileReader
	writer ports.FileWriter

	mu sync.Mutex
}

// New returns the use case over the notes the reader and the writer reach.
func New(files ports.FileReader, writer ports.FileWriter) *SetTaskDone {
	return &SetTaskDone{files: files, writer: writer, mu: sync.Mutex{}}
}

// Execute reads the note, finds the line, changes it and writes the note over the read baseline.
// A host failure is the error; a change the note does not allow is the Refusal of the result.
func (u *SetTaskDone) Execute(ctx context.Context, req Request) (Result, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	result, err := u.write(ctx, req)
	if errors.Is(err, host.ErrStale) {
		result, err = u.write(ctx, req)
	}
	if errors.Is(err, host.ErrStale) {
		return refuse(RefusalStale), nil
	}
	return result, err
}

func (u *SetTaskDone) write(ctx context.Context, req Request) (Result, error) {
	raw, baseline, err := u.files.Read(ctx, req.Path)
	if errors.Is(err, host.ErrNotExist) || errors.Is(err, host.ErrNotANote) {
		return refuse(RefusalNotFound), nil
	}
	if err != nil {
		return Result{}, err
	}
	tasks, err := todo.ParseNote(raw)
	if errors.Is(err, todo.ErrUnreadable) {
		return refuse(RefusalChanged), nil
	}
	if err != nil {
		return Result{}, err
	}
	found, ok := findTask(tasks, req.Line, req.Raw)
	if !ok {
		return refuse(RefusalNotFound), nil
	}
	changed, change, err := todo.ChangeLine(found.Raw, req.IsDone, req.Today)
	if err != nil {
		return refuse(getRefusal(err)), nil
	}
	start := getLineStart(raw, found.Line)
	content := make([]byte, 0, len(raw)+len(changed)-len(found.Raw))
	content = append(content, raw[:start]...)
	content = append(content, changed...)
	content = append(content, raw[start+len(found.Raw):]...)
	stat, err := u.writer.Write(ctx, req.Path, content, baseline)
	if err != nil {
		return Result{}, err
	}
	return Result{Task: getTaskAt(content, req.Path, found.Line), Change: change, Refusal: "", Content: content, Stat: stat}, nil
}

func refuse(refusal Refusal) Result {
	return Result{Task: todo.Task{}, Change: "", Refusal: refusal, Content: nil, Stat: domain.FileStat{}}
}

func findTask(tasks []todo.Task, line int, raw string) (todo.Task, bool) {
	// The task at the line wins; else the task of the same text nearest to it, the earlier on a tie.
	var nearest todo.Task
	distance := -1
	for _, task := range tasks {
		if task.Raw != raw {
			continue
		}
		away := task.Line - line
		if away < 0 {
			away = -away
		}
		if distance < 0 || away < distance {
			nearest, distance = task, away
		}
	}
	return nearest, distance >= 0
}

func getLineStart(raw []byte, line int) int {
	start := 0
	for range line - 1 {
		start += bytes.IndexByte(raw[start:], '\n') + 1
	}
	return start
}

func getTaskAt(content []byte, path string, line int) todo.Task {
	tasks, err := todo.ParseNote(content)
	if err != nil {
		return todo.Task{}
	}
	for _, task := range tasks {
		if task.Line == line {
			task.Path = path
			return task
		}
	}
	return todo.Task{}
}

func getRefusal(err error) Refusal {
	switch {
	case errors.Is(err, todo.ErrNotOpen):
		return RefusalNotOpen
	case errors.Is(err, todo.ErrNotDone):
		return RefusalNotDone
	case errors.Is(err, todo.ErrInvalidRecurrence):
		return RefusalInvalidRecurrence
	}
	return RefusalNotFound
}
