package vaultwatch

import (
	"fmt"
	"testing"
)

// TestTheQueueDropsAtItsBoundAndSaysSo. A bound with an honest overflow is what
// keeps a checkout of a large repository from becoming memory the machine does
// not have.
func TestTheQueueDropsAtItsBoundAndReturnsSo(t *testing.T) {
	queue := newQueue(2)
	for i := range 5 {
		queue.put(event{path: fmt.Sprintf("Note%d.md", i)})
	}

	events, finished, isDone := queue.take()
	if !finished {
		t.Fatal("the queue was filled past its bound and did not say so")
	}
	if isDone {
		t.Error("the queue said the watcher had stopped")
	}
	if len(events) > 2 {
		t.Errorf("the queue held %d events, past its bound of 2", len(events))
	}

	if _, doneChan, _ := queue.take(); doneChan {
		t.Error("one overflow was reported twice")
	}
}
