package task_test

import (
	"context"
	"fmt"
	"slices"
	"sync"
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

func TestWhatAListenerIsToldLastIsTheNewest(t *testing.T) {
	// Several pieces of work report at once, which is what the list is for. Each
	// of them only ever gets further on, so a listener told that one has got less
	// far than it already had has been handed an older list than the one before.
	tasks := task.New()
	ctx, stop := context.WithCancel(t.Context())
	watching := tasks.Watch(ctx)

	ended := make(chan []task.Task, 1)
	go func() {
		far := map[string]int64{}
		var last []task.Task
		for list := range watching {
			for _, at := range list {
				if at.Done < far[at.ID] {
					t.Errorf("%s had got to %d and is now said to be at %d", at.ID, far[at.ID], at.Done)
				}
				far[at.ID] = at.Done
			}
			last = list
		}
		ended <- last
	}()

	var reporting sync.WaitGroup
	for which := range 8 {
		reporting.Add(1)
		go func() {
			defer reporting.Done()
			id := fmt.Sprintf("reading-%d", which)
			for page := int64(1); page <= 200; page++ {
				tasks.Set(task.Task{ID: id, Doing: "Reading a scan", Done: page, Total: 200})
			}
		}()
	}
	reporting.Wait()

	// The stream ends with its context after the last list is in it, so what
	// comes out of it last is what a person is left looking at.
	doing := tasks.List()
	stop()
	if last := <-ended; !slices.Equal(last, doing) {
		t.Errorf("left showing %+v, want what is being done, %+v", last, doing)
	}
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
