package vault

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// importingFiles is what a drop is called in the list of what is being done.
const importingFiles = "import files"

// WriteGate admits the writes a window waits for before it closes.
type WriteGate interface {
	// BeginWrite registers a write and reports false once the window is closing, when nothing may be written.
	BeginWrite() bool
	// FinishWrite ends a write that BeginWrite registered.
	FinishWrite()
}

// Changes tells the views drawing the vault what changed.
type Changes interface {
	Publish(change window.Change)
}

// ImportFiles copies files dropped onto the window into a folder of the vault, shows the import as work
// while it runs, and tells the views what arrived.
type ImportFiles struct {
	imports Import
	gate    WriteGate
	tasks   windowport.Tasks
	changes Changes
}

// NewImportFiles returns the use case over the import of files, the writes the window waits for, the list
// of work and the feed of changes.
func NewImportFiles(imports Import, gate WriteGate, tasks windowport.Tasks, changes Changes) ImportFiles {
	return ImportFiles{imports: imports, gate: gate, tasks: tasks, changes: changes}
}

// Execute copies the paths into the folder into of the vault, the root being the empty path. It runs where
// the drop reached the application and returns no value: the outcome is the task, which keeps the failures
// it names, and the change of the files that landed in the folder. A window showing no vault, a drop of no
// path and a window that is closing import nothing.
// The watcher does not report images or archives, so the change names what arrived itself.
func (u ImportFiles) Execute(ctx context.Context, v domain.Vault, into string, paths []string) {
	if len(paths) == 0 || v.ID == "" || !u.gate.BeginWrite() {
		return
	}
	defer u.gate.FinishWrite()

	task := work.Task{ID: importingFiles + " " + into, Title: "Import files", About: into}
	u.tasks.AddTask(task)

	imported, err := u.imports.Execute(ctx, v, into, paths)
	if direct := imported.ListDirectChildren(into); len(direct) > 0 {
		u.changes.Publish(window.Change{Paths: direct})
	}
	if err != nil {
		task.Error = err.Error()
		u.tasks.AddTask(task)
		return
	}
	if failureText := imported.DescribeFailures(); failureText != "" {
		task.Error = failureText
		u.tasks.AddTask(task)
		return
	}
	u.tasks.RemoveTask(task.ID)
}
