package editor

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
)

func TestSuccessfulMeaningSearchRemovesTheWordsOnlyTask(t *testing.T) {
	tasks := run.NewTasks()
	state := SearchTasks{Tasks: &wire.Window{Tasks: tasks}}

	state.ReportWordsAlone(errors.New("dial tcp: network is unreachable"))
	if !hasTask(tasks, work.WordsAlone) {
		t.Fatal("a failed search by meaning left no task")
	}
	state.ClearWordsAlone()
	if hasTask(tasks, work.WordsAlone) {
		t.Fatalf("the words-only task stayed after the source answered: %v", tasks.List())
	}
}

func hasTask(tasks *run.Tasks, id string) bool {
	for _, task := range tasks.List() {
		if task.ID == id {
			return true
		}
	}
	return false
}

type testWindow struct {
	*Session
	*PageAccess
	Events *Events
}

func newTestWindow(tasks *run.Tasks, frame Frame, emit func(string)) *testWindow {
	session := NewSession(frame)
	return &testWindow{Session: session, PageAccess: NewPageAccess(tasks, session), Events: &Events{Emit: emit}}
}
