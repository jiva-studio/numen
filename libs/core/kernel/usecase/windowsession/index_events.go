package windowsession

import (
	"fmt"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// IndexEvents shows what the indexing worker reports in the window.
type IndexEvents struct {
	rt    *Runtime
	tasks windowport.Tasks
	out   io.Writer
}

// NewIndexEvents makes the events of the indexing worker of the window over rt. Diagnostics go to out.
func NewIndexEvents(rt *Runtime, tasks windowport.Tasks, out io.Writer) IndexEvents {
	return IndexEvents{rt: rt, tasks: tasks, out: out}
}

// ReportTask puts one piece of work in the list of what is being done.
func (e IndexEvents) ReportTask(pos work.Task) { e.tasks.AddTask(pos) }

// FinishTask takes one piece of work out of the list.
func (e IndexEvents) FinishTask(id string) { e.tasks.RemoveTask(id) }

// ReportError records why a pass could not run.
func (e IndexEvents) ReportError(err error) { e.rt.SetError(err.Error()) }

// ReportNotes records that the notes of a vault were read.
func (e IndexEvents) ReportNotes(v domain.Vault, notes int) {
	e.say("%s: %d notes\n", v.Name, notes)
	e.rt.SetReady(true)
}

// ReportTexts records how many file texts were cut.
func (e IndexEvents) ReportTexts(v domain.Vault, res vaultreading.IndexResult) {
	e.say("%s: %d file texts, %d chunks\n", v.Name, res.Extracted, res.Chunks)
}

func (e IndexEvents) say(format string, a ...any) {
	if _, err := fmt.Fprintf(e.out, format, a...); err != nil {
		e.tasks.AddTask(work.Task{ID: writingDiagnostics, Title: "Writing diagnostics", Error: err.Error()})
	}
}
