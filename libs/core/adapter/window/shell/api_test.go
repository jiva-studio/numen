package flashcards

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/run"
)

// A window built over no list of work keeps one of its own, and one built over
// a list keeps that list.
func TestAWindowKeepsTheListOfWorkItWasBuiltOver(t *testing.T) {
	if NewWindow(nil).Tasking == nil {
		t.Error("a window built over no list keeps none")
	}

	tasks := run.NewTasks()
	if NewWindow(tasks).Tasking != tasks {
		t.Error("a window built over a list keeps another")
	}
}
