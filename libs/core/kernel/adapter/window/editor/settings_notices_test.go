package editor

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
)

func TestTheNoticesOfTheSettingsReachTheWindowAsNoticesAndNotAsFailedTasks(t *testing.T) {
	const notice = "numen.json: no installed plugin md.numen.proofreading; its settings are ignored"
	state := NewState(run.NewTasks(), nil, nil)

	NewInstallation(state, Lifecycle{}, nil, nil).Report([]string{notice})

	tasks := state.Window.Tasks.List()
	if len(tasks) != 1 || !strings.Contains(tasks[0].About, notice) {
		t.Fatalf("the notice is not among the tasks as text: %+v", tasks)
	}
	if tasks[0].Error != "" {
		t.Errorf("the notice is a failed task: %q", tasks[0].Error)
	}
}
