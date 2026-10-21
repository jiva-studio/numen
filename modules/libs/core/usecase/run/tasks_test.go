package run

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestWhatIsBeingDoneIsWhatIsListed(t *testing.T) {
	tasks := NewTasks()
	tasks.Set(Task{ID: "recognition", Title: "Recognising a scan", About: "book.pdf"})
	tasks.Set(Task{ID: "indexing", Title: "Indexing"})

	list := tasks.List()
	if len(list) != 2 {
		t.Fatalf("listed %d, want 2", len(list))
	}
	// The order they were first seen in, so a list does not rearrange itself
	// under somebody reading it.
	if list[0].ID != "recognition" || list[1].ID != "indexing" {
		t.Errorf("listed %s then %s", list[0].ID, list[1].ID)
	}

	tasks.Remove("recognition")
	if list := tasks.List(); len(list) != 1 || list[0].ID != "indexing" {
		t.Errorf("after finishing one: %+v", list)
	}
}

func TestWorkReportedAgainReplacesItself(t *testing.T) {
	tasks := NewTasks()
	tasks.Set(Task{ID: "recognition", Title: "Recognising a scan", Count: 1, Total: 9})
	tasks.Set(Task{ID: "recognition", Title: "Recognising a scan", Count: 4, Total: 9})

	list := tasks.List()
	if len(list) != 1 {
		t.Fatalf("listed %d, want the one piece of work", len(list))
	}
	if list[0].Count != 4 {
		t.Errorf("it has got to %d, want 4", list[0].Count)
	}
}

func TestFinishingWhatIsNotThereIsTheOutcomeAskedFor(t *testing.T) {
	tasks := NewTasks()
	tasks.Remove("nothing")
	if list := tasks.List(); len(list) != 0 {
		t.Errorf("listed %+v", list)
	}
}

func TestClearTakesAllTasksOut(t *testing.T) {
	tasks := NewTasks()
	tasks.Set(Task{ID: "recognition", Title: "Recognising a scan"})
	tasks.Set(Task{ID: "indexing", Title: "Indexing"})

	if len(tasks.List()) != 2 {
		t.Fatalf("listed %d, want 2", len(tasks.List()))
	}

	tasks.Clear()
	if list := tasks.List(); len(list) != 0 {
		t.Errorf("after clear: %+v", list)
	}
}

func TestWatchingSaysWhatIsBeingDoneAndThenWhatChanges(t *testing.T) {
	tasks := NewTasks()
	tasks.Set(Task{ID: "recognition", Title: "Recognising a scan"})

	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	watching := tasks.Watch(ctx)

	// What is already being done, before anything changes: a window that opened
	// while work was running has to be told about it.
	if list := next(t, watching); len(list) != 1 {
		t.Fatalf("first said %+v", list)
	}

	tasks.Set(Task{ID: "fetching", Title: "Fetching"})
	if list := next(t, watching); len(list) != 2 {
		t.Errorf("after one more: %+v", list)
	}

	tasks.Remove("recognition")
	if list := next(t, watching); len(list) != 1 || list[0].ID != "fetching" {
		t.Errorf("after one finished: %+v", list)
	}
}

func TestAListenerBehindIsGivenTheNewestAndNotAQueue(t *testing.T) {
	// What the work was doing a second ago is of no interest to anybody, and a
	// listener that fell behind must not be made to walk through it.
	tasks := NewTasks()
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	watching := tasks.Watch(ctx)

	for i := int64(1); i <= 5; i++ {
		tasks.Set(Task{ID: "recognition", Title: "Recognising a scan", Count: i, Total: 5})
	}
	if list := next(t, watching); len(list) != 1 || list[0].Count != 5 {
		t.Errorf("said %+v, want the newest", list)
	}
}

func TestWatchingEndsWithItsContext(t *testing.T) {
	tasks := NewTasks()
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
	tasks := NewTasks()
	ctx, stop := context.WithCancel(t.Context())
	watching := tasks.Watch(ctx)

	ended := make(chan []Task, 1)
	go func() {
		far := map[string]int64{}
		var last []Task
		for list := range watching {
			for _, at := range list {
				if at.Count < far[at.ID] {
					t.Errorf("%s had got to %d and is now said to be at %d", at.ID, far[at.ID], at.Count)
				}
				far[at.ID] = at.Count
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
			id := fmt.Sprintf("recognition-%d", which)
			for page := int64(1); page <= 200; page++ {
				tasks.Set(Task{ID: id, Title: "Recognising a scan", Count: page, Total: 200})
			}
		}()
	}
	reporting.Wait()

	// The stream ends with its context after the last list is in it, so what
	// comes out of it last is what a person is left looking at.
	executing := tasks.List()
	stop()
	if last := <-ended; !slices.Equal(last, executing) {
		t.Errorf("left showing %+v, want what is being done, %+v", last, executing)
	}
}

func next(t *testing.T, watching <-chan []Task) []Task {
	t.Helper()
	select {
	case list := <-watching:
		return list
	case <-time.After(time.Second):
		t.Fatal("nothing was said")
	}
	return nil
}

// quiet fails when a listener has been told of anything.
func quiet(t *testing.T, watching <-chan []Task, why string) {
	t.Helper()
	select {
	case list := <-watching:
		t.Errorf("%s: a listener was told %+v", why, list)
	default:
	}
}

func TestWorkWithNoNameIsNotListed(t *testing.T) {
	tasks := NewTasks()
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	watching := tasks.Watch(ctx)
	next(t, watching)

	tasks.Set(Task{Title: "Recognising a scan"})
	if list := tasks.List(); len(list) != 0 {
		t.Errorf("listed %+v", list)
	}
	quiet(t, watching, "work with no name was reported")
}

func TestFinishingWhatIsNotThereTellsNobody(t *testing.T) {
	tasks := NewTasks()
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	watching := tasks.Watch(ctx)
	next(t, watching)

	tasks.Remove("recognition")
	quiet(t, watching, "a task that was not there was finished")
}

func TestWorkTakenOutCanBeReportedAgain(t *testing.T) {
	tasks := NewTasks()
	tasks.Set(Task{ID: "a"})
	tasks.Set(Task{ID: "b"})
	tasks.Remove("a")
	tasks.Set(Task{ID: "a", Title: "again"})

	list := tasks.List()
	if len(list) != 2 || list[0].ID != "b" || list[1].ID != "a" || list[1].Title != "again" {
		t.Errorf("listed %+v, want b and then a", list)
	}
}

func TestClearingTakesAllTasksOutAndTellsListeners(t *testing.T) {
	tasks := NewTasks()
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	watching := tasks.Watch(ctx)
	next(t, watching)

	tasks.Set(Task{ID: "a"})
	next(t, watching)
	tasks.Clear()
	if list := next(t, watching); len(list) != 0 {
		t.Errorf("told %+v after the list was cleared", list)
	}

	tasks.Set(Task{ID: "a", Title: "again"})
	if list := tasks.List(); len(list) != 1 || list[0].Title != "again" {
		t.Errorf("listed %+v after reporting the work again", list)
	}
}

func TestClearingWhatIsEmptyTellsNobody(t *testing.T) {
	tasks := NewTasks()
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	watching := tasks.Watch(ctx)
	next(t, watching)

	tasks.Clear()
	quiet(t, watching, "an empty list was cleared")
}

func TestClearingASingleTaskTakesItOut(t *testing.T) {
	tasks := NewTasks()
	tasks.Set(Task{ID: "a"})
	tasks.Clear()
	if list := tasks.List(); len(list) != 0 {
		t.Errorf("listed %+v", list)
	}
}

func TestAListenerThatEndedDoesNotStopTheOthersBeingTold(t *testing.T) {
	tasks := NewTasks()
	endingCtx, end := context.WithCancel(t.Context())
	ending := tasks.Watch(endingCtx)
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	staying := tasks.Watch(ctx)
	next(t, staying)

	end()
	for open := true; open; {
		select {
		case _, open = <-ending:
		case <-time.After(time.Second):
			t.Fatal("the stream did not end with its context")
		}
	}

	// Telling the ended listener would send on a closed channel.
	tasks.Set(Task{ID: "a"})
	if list := next(t, staying); len(list) != 1 {
		t.Errorf("the listener that stayed was told %+v", list)
	}
}
