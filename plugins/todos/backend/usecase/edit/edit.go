// Package edit adds, rewrites and removes task lines of a note and writes the note over the file
// stat it had when it was read, as the tick of a checkbox does.
package edit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/todos/backend/ports"
	"github.com/jiva-studio/numen/plugins/todos/backend/todo"
)

var (
	// ErrNoNote is a path that is no note of the vault.
	ErrNoNote = errors.New("the note does not exist")
	// ErrUnreadable is a note whose frontmatter cannot be read.
	ErrUnreadable = errors.New("the note cannot be read")
	// ErrNotTask is a line that is not a task line, or beyond the end of the note.
	ErrNotTask = errors.New("the line is not a task")
	// ErrChanged is a task whose text is not the text the caller expected.
	ErrChanged = errors.New("the task changed since it was listed")
	// ErrStale is a note that changed between the read and the write, twice.
	ErrStale = errors.New("the note changed while the task was written")
	// ErrInvalidText is a task text that is empty or has a line break.
	ErrInvalidText = errors.New("the task text must be one non-empty line")
)

// Editor changes the task lines of the notes the reader and the writer reach. Changes run one
// after the other.
type Editor struct {
	files  ports.FileReader
	writer ports.FileWriter
	store  ports.NoteStore

	mu sync.Mutex
}

// New returns the editor over the notes the reader and the writer reach; the store gets each note
// written.
func New(files ports.FileReader, writer ports.FileWriter, store ports.NoteStore) *Editor {
	return &Editor{files: files, writer: writer, store: store, mu: sync.Mutex{}}
}

func (e *Editor) Find(ctx context.Context, path string, line int) (todo.Task, error) {
	raw, _, err := e.files.Read(ctx, path)
	if err != nil {
		return todo.Task{}, convertReadError(err)
	}
	task, err := findAt(raw, path, line, "")
	return task, err
}

// Add appends an open task line to the end of the note and returns the task written.
func (e *Editor) Add(ctx context.Context, path, text string) (todo.Task, error) {
	text = strings.TrimSpace(text)
	if err := checkText(text); err != nil {
		return todo.Task{}, err
	}
	return e.write(ctx, path, func(raw []byte) ([]byte, int, error) {
		if _, err := todo.ParseNote(raw); err != nil {
			return nil, 0, ErrUnreadable
		}
		content := raw
		if len(content) > 0 && content[len(content)-1] != '\n' {
			content = append(bytes.Clone(content), '\n')
		}
		line := bytes.Count(content, []byte("\n")) + 1
		return append(bytes.Clone(content), todo.NewLine(text)+"\n"...), line, nil
	}, true)
}

// Edit replaces the text after the checkbox of the task at the line and returns the task written.
// A non-empty expected text must equal the text of the task now.
func (e *Editor) Edit(ctx context.Context, path string, line int, expected, text string) (todo.Task, error) {
	text = strings.TrimSpace(text)
	if err := checkText(text); err != nil {
		return todo.Task{}, err
	}
	return e.write(ctx, path, func(raw []byte) ([]byte, int, error) {
		found, err := findAt(raw, path, line, expected)
		if err != nil {
			return nil, 0, err
		}
		changed, err := todo.ReplaceText(found.Raw, text)
		if err != nil {
			return nil, 0, fmt.Errorf("%w: %w", ErrInvalidText, err)
		}
		return replaceLine(raw, line, changed), line, nil
	}, true)
}

// Remove deletes the task line, with its line break, and returns the task removed. A non-empty
// expected text must equal the text of the task now.
func (e *Editor) Remove(ctx context.Context, path string, line int, expected string) (todo.Task, error) {
	var removed todo.Task
	_, err := e.write(ctx, path, func(raw []byte) ([]byte, int, error) {
		found, err := findAt(raw, path, line, expected)
		if err != nil {
			return nil, 0, err
		}
		removed = found
		return replaceLine(raw, line, ""), 0, nil
	}, false)
	return removed, err
}

func (e *Editor) write(
	ctx context.Context, path string, change func(raw []byte) ([]byte, int, error), isReturned bool,
) (todo.Task, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	task, err := e.attempt(ctx, path, change, isReturned)
	if errors.Is(err, host.ErrStale) {
		task, err = e.attempt(ctx, path, change, isReturned)
	}
	if errors.Is(err, host.ErrStale) {
		return todo.Task{}, ErrStale
	}
	return task, err
}

func (e *Editor) attempt(
	ctx context.Context, path string, change func(raw []byte) ([]byte, int, error), isReturned bool,
) (todo.Task, error) {
	raw, baseline, err := e.files.Read(ctx, path)
	if err != nil {
		return todo.Task{}, convertReadError(err)
	}
	content, line, err := change(raw)
	if err != nil {
		return todo.Task{}, err
	}
	stat, err := e.writer.Write(ctx, path, content, baseline)
	if err != nil {
		return todo.Task{}, err
	}
	e.store.StoreNote(path, content, stat)
	if !isReturned {
		return todo.Task{}, nil
	}
	return findAt(content, path, line, "")
}

func convertReadError(err error) error {
	if errors.Is(err, host.ErrNotExist) || errors.Is(err, host.ErrNotANote) {
		return ErrNoNote
	}
	return err
}

func checkText(text string) error {
	if text == "" || strings.ContainsAny(text, "\r\n") {
		return ErrInvalidText
	}
	return nil
}

func findAt(raw []byte, path string, line int, expected string) (todo.Task, error) {
	tasks, err := todo.ParseNote(raw)
	if err != nil {
		return todo.Task{}, ErrUnreadable
	}
	for _, task := range tasks {
		if task.Line != line {
			continue
		}
		if expected != "" && task.Text != expected {
			return todo.Task{}, ErrChanged
		}
		task.Path = path
		return task, nil
	}
	return todo.Task{}, ErrNotTask
}

func replaceLine(raw []byte, line int, text string) []byte {
	lines := bytes.SplitAfter(raw, []byte("\n"))
	old := lines[line-1]
	if text == "" {
		return bytes.Join(append(lines[:line-1:line-1], lines[line:]...), nil)
	}
	ending := old[len(bytes.TrimRight(old, "\r\n")):]
	lines[line-1] = append([]byte(text), ending...)
	return bytes.Join(lines, nil)
}
