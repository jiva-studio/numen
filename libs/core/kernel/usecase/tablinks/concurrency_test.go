package tablinks

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"
)

// scheduleAfterDelay runs f on its own goroutine after d, the way the application schedules.
func scheduleAfterDelay(d time.Duration, f func()) func() bool {
	return time.AfterFunc(d, f).Stop
}

// busyWindow drives one window the way a window does: it watches, reports its tabs, links them, moves a tab
// to the next window, detaches another, and closes its watch, over and over.
func busyWindow(r *registry, index, windows, rounds int) {
	id := "window-" + strconv.Itoa(index)
	next := "window-" + strconv.Itoa((index+1)%windows)
	mine := func(n int) string { return "tab:" + strconv.Itoa(index) + ":" + strconv.Itoa(n) }
	for round := range rounds {
		watch := r.Watch(id)
		r.WriteTabs(Window{ID: id, IsMain: index == 0, TabCount: 3}, []tablink.Tab{
			{ID: mine(0), Kind: "files"}, {ID: mine(1), Kind: "plex"}, {ID: mine(2), Kind: "files"},
		})
		r.Link(mine(0), mine(1))
		r.Link(mine(1), mine(2))
		r.SyncPath(mine(0), "notes/"+strconv.Itoa(round)+".md", "Note")
		r.Highlight(next, round%2 == 0)
		_ = r.GetSnapshot()

		moved := r.Move(id, next, tablink.SavedTab{ID: mine(2), Kind: "files"})
		saved := tablink.SavedTab{ID: mine(1), Kind: "plex"}
		detached := r.Detach(saved)
		if _, moveID, ok := r.ReadDetached(id, saved.ID); ok {
			r.Accept(id, saved.ID, moveID)
		}
		r.CancelMove(mine(2), moved)
		r.CancelDetach(mine(1))
		r.Unlink(mine(0))
		<-detached
		<-moved

		if round%3 == 0 {
			r.Cancel(id)
		}
		watch.Close()
	}
}

func TestWindowsWorkingAtOnceLeaveNothingBehind(t *testing.T) {
	const windows, rounds = 6, 60
	tabs, err := New(time.Millisecond, scheduleAfterDelay, &testEvents{})
	if err != nil {
		t.Fatal(err)
	}
	r := &registry{Links: tabs.Links, Windows: tabs.Windows, Transfers: tabs.Transfers, state: tabs.Links.state.peek(), watches: map[string]*Watch{}}

	var working sync.WaitGroup
	for index := range windows {
		working.Go(func() { busyWindow(r, index, windows, rounds) })
	}
	done := make(chan struct{})
	go func() {
		working.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(60 * time.Second):
		t.Fatal("the windows did not finish: the use cases hold each other")
	}

	deadline := time.After(10 * time.Second)
	for {
		snapshot := r.GetSnapshot()
		if len(snapshot.Windows) == 0 && len(snapshot.Tabs) == 0 && len(snapshot.Groups) == 0 {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("after every window closed, the state keeps %d windows, %d tabs and %d groups",
				len(snapshot.Windows), len(snapshot.Tabs), len(snapshot.Groups))
		case <-time.After(5 * time.Millisecond):
		}
	}
}
