package shell

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
)

func TestAWindowKeepsTheListOfWorkItWasBuiltOver(t *testing.T) {
	if NewWindow(testWindow, nil).Tasks == nil {
		t.Error("a window built over no list keeps none")
	}

	tasks := run.NewTasks()
	if NewWindow(testWindow, tasks).Tasks != tasks {
		t.Error("a window built over a list keeps another")
	}
}
