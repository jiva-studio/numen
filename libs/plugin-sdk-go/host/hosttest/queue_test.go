package hosttest_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
)

func enqueueBehind(t *testing.T, q *hosttest.Queue, isUserStarted bool) <-chan host.Turn {
	t.Helper()
	turns := make(chan host.Turn, 1)
	queued := make(chan struct{})
	go func() {
		turn, err := q.EnqueueWork(t.Context(), host.Work{IsUserStarted: isUserStarted, OnWaiting: func() { close(queued) }})
		if err == nil {
			turns <- turn
		}
	}()
	select {
	case <-queued:
	case <-time.After(time.Second):
		t.Fatal("the work never waited")
	}
	return turns
}

func TestQueueRunsAPersonsRequestBeforeBackgroundWork(t *testing.T) {
	q := hosttest.NewQueue()
	first, err := q.EnqueueWork(t.Context(), host.Work{})
	if err != nil {
		t.Fatal(err)
	}
	contended := first.WatchWaiting()
	background := enqueueBehind(t, q, false)
	requested := enqueueBehind(t, q, true)
	select {
	case <-contended:
	default:
		t.Fatal("no signal for the work queued behind the turn")
	}

	first.Release()
	select {
	case turn := <-requested:
		select {
		case <-background:
			t.Fatal("background work ran beside the person's request")
		default:
		}
		turn.Release()
	case <-background:
		t.Fatal("background work ran before the person's request")
	case <-time.After(time.Second):
		t.Fatal("nothing ran")
	}
	select {
	case turn := <-background:
		turn.Release()
	case <-time.After(time.Second):
		t.Fatal("background work never ran")
	}
}

func TestQueueListsATaskUntilItIsFinished(t *testing.T) {
	q := hosttest.NewQueue()
	ctx := t.Context()
	for _, task := range []host.Task{{ID: "a", Title: "One"}, {ID: "b", Title: "Two"}, {ID: "a", Title: "One again", Done: 1, Total: 2}} {
		if err := q.ReportTask(ctx, task); err != nil {
			t.Fatal(err)
		}
	}

	got := q.ListTasks()
	if len(got) != 2 || got[0].Title != "One again" || got[1].Title != "Two" {
		t.Fatalf("got %+v, want a replaced in place and b after it", got)
	}
	if err := q.FinishTask(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if got := q.ListTasks(); len(got) != 1 || got[0].ID != "b" {
		t.Fatalf("got %+v after finishing a, want only b", got)
	}
}
