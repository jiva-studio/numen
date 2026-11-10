package container

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
)

func TestTheIndexOfAPluginCommandClosesOnceThePluginWorkFinished(t *testing.T) {
	work := run.NewPluginWork(&run.Lock{}, run.NewTasks(), nil)
	work.ReportTask("org.example.ears", plugin.Task{ID: "talk", Title: "Transcribing"})
	closed := make(chan struct{})
	closeIndex := closeAfterWork(t.Context(), work, func() error {
		close(closed)
		return nil
	})

	finished := make(chan error, 1)
	go func() { finished <- closeIndex() }()
	select {
	case <-closed:
		t.Fatal("the index closed while a plugin task ran")
	case <-time.After(30 * time.Millisecond):
	}
	work.FinishTask("org.example.ears", "talk")

	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("the index did not close once the task finished")
	}
}
