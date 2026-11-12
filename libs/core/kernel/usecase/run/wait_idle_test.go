package run

import (
	"context"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func waitIdleIn(work *PluginWork, ctx context.Context) <-chan error {
	finished := make(chan error, 1)
	go func() { finished <- work.WaitIdle(ctx) }()
	return finished
}

func requireWaiting(t *testing.T, finished <-chan error, step string) {
	t.Helper()
	select {
	case err := <-finished:
		t.Fatalf("%s: the wait ended with %v while the work runs", step, err)
	case <-time.After(30 * time.Millisecond):
	}
}

func requireFinished(t *testing.T, finished <-chan error, step string) {
	t.Helper()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatalf("%s: %v", step, err)
		}
	case <-time.After(time.Second):
		t.Fatalf("%s: the wait did not end", step)
	}
}

func TestWaitIdleWaitsForATaskAndTheLeaseOfAPlugin(t *testing.T) {
	work := NewPluginWork(&Lock{}, NewTasks(), nil)
	turn, err := work.EnqueueWork(t.Context(), "org.example.ears", true, nil)
	if err != nil {
		t.Fatal(err)
	}
	work.ReportTask("org.example.ears", plugin.Task{ID: "talk", Title: "Transcribing"})

	finished := waitIdleIn(work, t.Context())
	requireWaiting(t, finished, "a task and a lease")
	work.FinishTask("org.example.ears", "talk")
	requireWaiting(t, finished, "a lease")
	turn.Release()
	requireFinished(t, finished, "nothing")
}

func TestWaitIdleDoesNotWaitForAFailedTask(t *testing.T) {
	work := NewPluginWork(&Lock{}, NewTasks(), nil)
	work.ReportTask("org.example.ears", plugin.Task{ID: "talk", Title: "Transcribing", Error: "no model"})

	requireFinished(t, waitIdleIn(work, t.Context()), "a failed task")
}

func TestWaitIdleEndsWithItsContext(t *testing.T) {
	work := NewPluginWork(&Lock{}, NewTasks(), nil)
	work.ReportTask("org.example.ears", plugin.Task{ID: "talk", Title: "Transcribing"})
	ctx, cancel := context.WithCancel(t.Context())

	finished := waitIdleIn(work, ctx)
	cancel()

	select {
	case err := <-finished:
		if err == nil {
			t.Error("a wait whose context ended reported the work finished")
		}
	case <-time.After(time.Second):
		t.Fatal("the wait outlived its context")
	}
}
