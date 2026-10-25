package run

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// enqueueBehind starts the plugin's work while the lock is held and returns once
// it waits, with the channel its turn arrives on.
func enqueueBehind(
	t *testing.T, ctx context.Context, work *PluginWork, pluginID string, isUserStarted bool,
) (<-chan pluginport.PluginTurn, <-chan error) {
	t.Helper()
	turns := make(chan pluginport.PluginTurn, 1)
	errs := make(chan error, 1)
	queued := make(chan struct{})
	go func() {
		turn, err := work.EnqueueWork(ctx, pluginID, isUserStarted, func() { close(queued) })
		if err != nil {
			errs <- err
			return
		}
		turns <- turn
	}()
	select {
	case <-queued:
	case err := <-errs:
		t.Fatalf("%s did not wait: %v", pluginID, err)
	case <-time.After(time.Second):
		t.Fatalf("%s never waited", pluginID)
	}
	return turns, errs
}

func receiveTurn(t *testing.T, turns <-chan pluginport.PluginTurn, pluginID string) pluginport.PluginTurn {
	t.Helper()
	select {
	case turn := <-turns:
		return turn
	case <-time.After(time.Second):
		t.Fatalf("%s got no turn", pluginID)
	}
	return nil
}

func TestPluginWorkRunsAPersonsRequestBeforeBackgroundWork(t *testing.T) {
	lock := &Lock{}
	work := NewPluginWork(lock, NewTasks())

	release, err := lock.Acquire(t.Context(), true, nil)
	if err != nil {
		t.Fatal(err)
	}
	background, _ := enqueueBehind(t, t.Context(), work, "indexer", false)
	requested, _ := enqueueBehind(t, t.Context(), work, "transcriber", true)

	release()
	first := receiveTurn(t, requested, "transcriber")
	select {
	case <-background:
		t.Fatal("background work ran beside the person's request")
	default:
	}
	first.Release()
	receiveTurn(t, background, "indexer").Release()
}

func TestPluginTurnReportsContention(t *testing.T) {
	lock := &Lock{}
	work := NewPluginWork(lock, NewTasks())

	turn, err := work.EnqueueWork(t.Context(), "transcriber", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	contended := turn.WatchWaiting()
	select {
	case <-contended:
		t.Fatal("signalled work behind the turn while none is queued")
	default:
	}
	next, _ := enqueueBehind(t, t.Context(), work, "recogniser", true)
	select {
	case <-contended:
	case <-time.After(time.Second):
		t.Fatal("no signal for the work queued behind the turn")
	}
	turn.Release()
	turn.Release()
	select {
	case <-turn.WatchEnd():
	default:
		t.Fatal("a released turn is not over")
	}
	receiveTurn(t, next, "recogniser").Release()
}

func TestPluginWorkListsAPluginsTaskAndRemovesIt(t *testing.T) {
	tasks := NewTasks()
	work := NewPluginWork(&Lock{}, tasks)

	work.ReportTask("transcriber", pluginport.PluginTask{
		ID: "a.mp3", Title: "Transcribing", About: "a.mp3", Done: 3, Total: 10,
		Unit: pluginport.TaskUnitSeconds, IsUserStarted: true,
	})
	work.ReportTask("recogniser", pluginport.PluginTask{ID: "a.mp3", Title: "Recognising", Error: "no models"})

	got := tasks.List()
	if len(got) != 2 {
		t.Fatalf("got %d tasks, want 2: %+v", len(got), got)
	}
	want := Task{
		ID: "plugin:transcriber:a.mp3", Title: "Transcribing", About: "a.mp3", Count: 3, Total: 10,
		Unit: Seconds, IsUserStarted: true,
	}
	if got[0] != want {
		t.Errorf("got %+v, want %+v", got[0], want)
	}
	if got[1].ID == got[0].ID || got[1].Error != "no models" {
		t.Errorf("the second plugin's task is %+v", got[1])
	}

	work.FinishTask("transcriber", "a.mp3")
	got = tasks.List()
	if len(got) != 1 || got[0].Title != "Recognising" {
		t.Fatalf("after finishing, got %+v, want only the recogniser's task", got)
	}
}

func TestPluginWorkClearsTheTasksAndTurnsOfAPluginThatStopped(t *testing.T) {
	lock := &Lock{}
	tasks := NewTasks()
	work := NewPluginWork(lock, tasks)

	work.ReportTask("transcriber", pluginport.PluginTask{ID: "a", Title: "Transcribing"})
	work.ReportTask("transcriber", pluginport.PluginTask{ID: "b", Title: "Transcribing", Error: "failed"})
	work.ReportTask("recogniser", pluginport.PluginTask{ID: "a", Title: "Recognising"})

	held, err := work.EnqueueWork(t.Context(), "transcriber", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, cleared := enqueueBehind(t, t.Context(), work, "transcriber", true)
	other, _ := enqueueBehind(t, t.Context(), work, "recogniser", false)

	work.ClearPlugin("transcriber")

	select {
	case err := <-cleared:
		if !errors.Is(err, errPluginCleared) {
			t.Errorf("the stopped plugin's wait ended with %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("the stopped plugin still waits")
	}
	select {
	case <-held.WatchEnd():
	default:
		t.Fatal("the stopped plugin's turn is not over")
	}
	receiveTurn(t, other, "recogniser").Release()

	got := tasks.List()
	if len(got) != 1 || got[0].ID != "plugin:recogniser:a" {
		t.Fatalf("got %+v, want only the recogniser's task", got)
	}
}
