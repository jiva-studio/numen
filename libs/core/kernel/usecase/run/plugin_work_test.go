package run

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	taskwork "github.com/jiva-studio/numen/libs/core/kernel/domain/work"
)

func enqueueAfter(
	t *testing.T, ctx context.Context, work *PluginWork, pluginID string, isUserStarted bool,
) (<-chan *Lease, <-chan error) {
	t.Helper()
	turns := make(chan *Lease, 1)
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

func receiveLease(t *testing.T, turns <-chan *Lease, pluginID string) *Lease {
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
	work := NewPluginWork(lock, NewTasks(), nil)

	release, err := lock.Acquire(t.Context(), true, nil)
	if err != nil {
		t.Fatal(err)
	}
	background, _ := enqueueAfter(t, t.Context(), work, "indexer", false)
	requested, _ := enqueueAfter(t, t.Context(), work, "transcriber", true)

	release()
	first := receiveLease(t, requested, "transcriber")
	select {
	case <-background:
		t.Fatal("background work ran beside the person's request")
	default:
	}
	first.Release()
	receiveLease(t, background, "indexer").Release()
}

func TestLeaseReportsContention(t *testing.T) {
	lock := &Lock{}
	work := NewPluginWork(lock, NewTasks(), nil)

	turn, err := work.EnqueueWork(t.Context(), "transcriber", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	contended := turn.WatchWaiting()
	select {
	case <-contended:
		t.Fatal("signaled work behind the turn while none is queued")
	default:
	}
	next, _ := enqueueAfter(t, t.Context(), work, "recognizer", true)
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
	receiveLease(t, next, "recognizer").Release()
}

func TestPluginWorkListsAPluginsTaskAndRemovesIt(t *testing.T) {
	tasks := NewTasks()
	work := NewPluginWork(&Lock{}, tasks, nil)

	work.ReportTask("transcriber", plugin.Task{
		ID: "a.mp3", Title: "Transcribing", About: "a.mp3", Done: 3, Total: 10,
		Unit: plugin.TaskUnitSeconds, IsUserStarted: true,
	})
	work.ReportTask("recognizer", plugin.Task{ID: "a.mp3", Title: "Recognizing", Error: "no models"})

	got := tasks.List()
	if len(got) != 2 {
		t.Fatalf("got %d tasks, want 2: %+v", len(got), got)
	}
	want := taskwork.Task{
		ID: "plugin:transcriber:a.mp3", Title: "Transcribing", About: "a.mp3", Count: 3, Total: 10,
		Unit: taskwork.Seconds, IsUserStarted: true, PluginID: "transcriber",
	}
	if !reflect.DeepEqual(got[0], want) {
		t.Errorf("got %+v, want %+v", got[0], want)
	}
	if got[1].ID == got[0].ID || got[1].Error != "no models" {
		t.Errorf("the second plugin's task is %+v", got[1])
	}

	work.FinishTask("transcriber", "a.mp3")
	got = tasks.List()
	if len(got) != 1 || got[0].Title != "Recognizing" {
		t.Fatalf("after finishing, got %+v, want only the recognizer's task", got)
	}
}

func TestPluginWorkClearsTheTasksAndTurnsOfAPluginThatStopped(t *testing.T) {
	lock := &Lock{}
	tasks := NewTasks()
	work := NewPluginWork(lock, tasks, nil)

	work.ReportTask("transcriber", plugin.Task{ID: "a", Title: "Transcribing"})
	work.ReportTask("transcriber", plugin.Task{ID: "b", Title: "Transcribing", Error: "failed"})
	work.ReportTask("recognizer", plugin.Task{ID: "a", Title: "Recognizing"})

	held, err := work.EnqueueWork(t.Context(), "transcriber", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, cleared := enqueueAfter(t, t.Context(), work, "transcriber", true)
	other, _ := enqueueAfter(t, t.Context(), work, "recognizer", false)

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
	receiveLease(t, other, "recognizer").Release()

	got := tasks.List()
	if len(got) != 1 || got[0].ID != "plugin:recognizer:a" {
		t.Fatalf("got %+v, want only the recognizer's task", got)
	}
}

func TestReportTaskSetsPluginIDAndName(t *testing.T) {
	tasks := NewTasks()
	names := map[string]string{"md.example.engine": "Example Engine"}
	work := NewPluginWork(&Lock{}, tasks, func(id string) string { return names[id] })

	work.ReportTask("md.example.engine", plugin.Task{ID: "one", Title: "Reading"})

	got := tasks.List()
	if len(got) != 1 || got[0].PluginID != "md.example.engine" || got[0].PluginName != "Example Engine" {
		t.Fatalf("got %+v, want the task under md.example.engine named Example Engine", got)
	}
}

func TestReportTaskWithNilNamesSetsEmptyName(t *testing.T) {
	tasks := NewTasks()
	work := NewPluginWork(&Lock{}, tasks, nil)

	work.ReportTask("md.example.engine", plugin.Task{ID: "one", Title: "Reading"})

	got := tasks.List()
	if len(got) != 1 || got[0].PluginID != "md.example.engine" || got[0].PluginName != "" {
		t.Fatalf("got %+v, want the task under md.example.engine with no name", got)
	}
}
