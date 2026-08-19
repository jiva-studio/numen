package task_test

import (
	"context"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/task"
)

func TestWhatIsBeingDoneIsWhatIsListed(t *testing.T) {
	tasks := task.New()
	tasks.Set(task.Task{ID: "reading", Doing: "Reading a scan", About: "book.pdf"})
	tasks.Set(task.Task{ID: "indexing", Doing: "Indexing"})

	list := tasks.List()
	if len(list) != 2 {
		t.Fatalf("listed %d, want 2", len(list))
	}
	// The order they were first seen in, so a list does not rearrange itself
	// under somebody reading it.
	if list[0].ID != "reading" || list[1].ID != "indexing" {
		t.Errorf("listed %s then %s", list[0].ID, list[1].ID)
	}

	tasks.Done("reading")
	if list := tasks.List(); len(list) != 1 || list[0].ID != "indexing" {
		t.Errorf("after finishing one: %+v", list)
	}
}

func TestWorkReportedAgainReplacesItself(t *testing.T) {
	tasks := task.New()
	tasks.Set(task.Task{ID: "reading", Doing: "Reading a scan", Done: 1, Total: 9})
	tasks.Set(task.Task{ID: "reading", Doing: "Reading a scan", Done: 4, Total: 9})

	list := tasks.List()
	if len(list) != 1 {
		t.Fatalf("listed %d, want the one piece of work", len(list))
	}
	if list[0].Done != 4 {
		t.Errorf("it has got to %d, want 4", list[0].Done)
	}
}

func TestFinishingWhatIsNotThereIsTheOutcomeAskedFor(t *testing.T) {
	tasks := task.New()
	tasks.Done("nothing")
	if list := tasks.List(); len(list) != 0 {
		t.Errorf("listed %+v", list)
	}
}

func TestWatchingSaysWhatIsBeingDoneAndThenWhatChanges(t *testing.T) {
	tasks := task.New()
	tasks.Set(task.Task{ID: "reading", Doing: "Reading a scan"})

	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	watching := tasks.Watch(ctx)

	// What is already being done, before anything changes: a window that opened
	// while work was running has to be told about it.
	if list := next(t, watching); len(list) != 1 {
		t.Fatalf("first said %+v", list)
	}

	tasks.Set(task.Task{ID: "fetching", Doing: "Fetching"})
	if list := next(t, watching); len(list) != 2 {
		t.Errorf("after one more: %+v", list)
	}

	tasks.Done("reading")
	if list := next(t, watching); len(list) != 1 || list[0].ID != "fetching" {
		t.Errorf("after one finished: %+v", list)
	}
}

func TestAListenerBehindIsGivenTheNewestAndNotAQueue(t *testing.T) {
	// What the work was doing a second ago is of no interest to anybody, and a
	// listener that fell behind must not be made to walk through it.
	tasks := task.New()
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	watching := tasks.Watch(ctx)

	for i := int64(1); i <= 5; i++ {
		tasks.Set(task.Task{ID: "reading", Doing: "Reading a scan", Done: i, Total: 5})
	}
	if list := next(t, watching); len(list) != 1 || list[0].Done != 5 {
		t.Errorf("said %+v, want the newest", list)
	}
}

func TestWatchingEndsWithItsContext(t *testing.T) {
	tasks := task.New()
	ctx, stop := context.WithCancel(t.Context())
	watching := tasks.Watch(ctx)
	stop()

	// The first is what was being done when it began; the channel closes after.
	for range 2 {
		select {
		case _, open := <-watching:
			if !open {
				return
			}
		case <-time.After(time.Second):
			t.Fatal("the stream did not end with its context")
		}
	}
	t.Fatal("the stream did not end with its context")
}

func next(t *testing.T, watching <-chan []task.Task) []task.Task {
	t.Helper()
	select {
	case list := <-watching:
		return list
	case <-time.After(time.Second):
		t.Fatal("nothing was said")
	}
	return nil
}
