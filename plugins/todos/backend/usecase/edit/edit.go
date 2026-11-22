// Package edit adds, rewrites and removes task lines of a note and writes the note over the file
// stat it had when it was read, as the tick of a checkbox does.
package edit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

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
	// ErrNoExpected is a change that does not say the text of the task it targets.
	ErrNoExpected = errors.New("the expected text of the task is required")
	// ErrInvalidText is a task text that is empty or has a line break.
	ErrInvalidText = errors.New("the task text must be one non-empty line")
)

// Editor changes the task lines of the notes the reader and the writer reach. Changes run one
// after the other.
type Editor struct {
	files  ports.FileReader
	writer ports.FileWriter
	store  ports.NoteStore
	lock   *ports.WriteLock
}

// New returns the editor over the notes the reader and the writer reach; the store gets each note
// written and the lock orders its writes with the other use cases that write notes.
func New(files ports.FileReader, writer ports.FileWriter, store ports.NoteStore, lock *ports.WriteLock) *Editor {
	return &Editor{files: files, writer: writer, store: store, lock: lock}
}

// Find returns the task at the line when its text is the expected text, which is required.
func (e *Editor) Find(ctx context.Context, path string, line int, expected string) (todo.Task, error) {
	if expected == "" {
		return todo.Task{}, ErrNoExpected
	}
	raw, _, err := e.files.Read(ctx, path)
	if err != nil {
		return todo.Task{}, convertReadError(err)
	}
	return findAt(raw, path, line, expected)
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
		content = append(bytes.Clone(content), todo.NewLine(text)+"\n"...)
		if _, err := findAt(content, path, line, ""); err != nil {
			return nil, 0, ErrNotTask
		}
		return content, line, nil
	})
}

// Edit replaces the text after the checkbox of the task at the line and returns the task written.
// The expected text is required and must equal the text of the task now.
func (e *Editor) Edit(ctx context.Context, path string, line int, expected, text string) (todo.Task, error) {
	if expected == "" {
		return todo.Task{}, ErrNoExpected
	}
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
	})
}

// Remove deletes the task line, with its line break, and returns the task removed. The expected
// text is required and must equal the text of the task now.
func (e *Editor) Remove(ctx context.Context, path string, line int, expected string) (todo.Task, error) {
	if expected == "" {
		return todo.Task{}, ErrNoExpected
	}
	var removed todo.Task
	err := e.run(ctx, path, func(raw []byte) ([]byte, error) {
		found, err := findAt(raw, path, line, expected)
		if err != nil {
			return nil, err
		}
		removed = found
		return replaceLine(raw, line, ""), nil
	})
	return removed, err
}

func (e *Editor) write(
	ctx context.Context, path string, change func(raw []byte) ([]byte, int, error),
) (todo.Task, error) {
	var line int
	var content []byte
	err := e.run(ctx, path, func(raw []byte) ([]byte, error) {
		var err error
		content, line, err = change(raw)
		return content, err
	})
	if err != nil {
		return todo.Task{}, err
	}
	return findAt(content, path, line, "")
}

func (e *Editor) run(ctx context.Context, path string, change func(raw []byte) ([]byte, error)) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	err := e.attempt(ctx, path, change)
	if errors.Is(err, host.ErrStale) {
		err = e.attempt(ctx, path, change)
	}
	if errors.Is(err, host.ErrStale) {
		return ErrStale
	}
	return err
}

func (e *Editor) attempt(ctx context.Context, path string, change func(raw []byte) ([]byte, error)) error {
	raw, baseline, err := e.files.Read(ctx, path)
	if err != nil {
		return convertReadError(err)
	}
	content, err := change(raw)
	if err != nil {
		return err
	}
	stat, err := e.writer.Write(ctx, path, content, baseline)
	if err != nil {
		return err
	}
	e.store.StoreNote(path, content, stat)
	return nil
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
