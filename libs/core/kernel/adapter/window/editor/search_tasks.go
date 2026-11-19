package editor

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
)

// SearchTasks reports the availability of search on the window's task list.
type SearchTasks struct{ Window *wire.Window }

// ReportWordsAlone records that the search runs on words only, with the reason. It is a task because
// a window opened from a desktop entry has no terminal.
func (s SearchTasks) ReportWordsAlone(err error) {
	s.Window.AddTask(work.Task{ID: work.WordsAlone, Title: "Search by words only", Error: err.Error()})
}

// ClearWordsAlone removes the words-only task once a search source answers again.
func (s SearchTasks) ClearWordsAlone() {
	s.Window.RemoveTask(work.WordsAlone)
}

// ReportFloorError records a similarity floor that is stored and cannot be read.
func (s SearchTasks) ReportFloorError(err error) {
	s.Window.AddTask(work.Task{ID: readingTheFloor, Title: "Read search floor", Error: err.Error()})
}
