package editor

import (
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/note/edit"
)

// Writes is the writes to the vault that the window waits for before it closes, and what a write
// tells: whether the index followed it, and the embedder that a note holds new prose.
type Writes struct {
	writing           inflight
	window            *wire.Window
	NotifyNoteWritten func()
}

// BeginWrite registers a write and reports false once the window is closing, when nothing may be
// written.
func (w *Writes) BeginWrite() bool { return w.writing.begin() }

// FinishWrite ends a write that BeginWrite registered.
func (w *Writes) FinishWrite() { w.writing.finish() }

// Seal refuses new writes and answers with what closes once the writes taken have finished.
func (w *Writes) Seal() <-chan struct{} { return w.writing.seal() }

// Open accepts writes again.
func (w *Writes) Open() { w.writing.Open() }

// IsIndexOutdated reports whether err says the write reached the vault and the index did not
// follow, and shows the task that tells the user so.
func (w *Writes) IsIndexOutdated(err error) bool {
	if err == nil {
		w.window.RemoveTask(updatingIndex)
		return false
	}
	if !errors.Is(err, edit.ErrIndexOutdated) {
		return false
	}
	w.window.AddTask(work.Task{ID: updatingIndex, Title: "Update index", Error: err.Error()})
	return true
}

// NoteWritten tells the embedder that a note holds new prose.
func (w *Writes) NoteWritten() {
	if w.NotifyNoteWritten != nil {
		w.NotifyNoteWritten()
	}
}
