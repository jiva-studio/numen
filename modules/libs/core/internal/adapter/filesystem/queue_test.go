package filesystem

import (
	"fmt"
	"testing"
)

// TestTheQueueDropsAtItsBoundAndSaysSo. A bound with an honest overflow is what
// keeps a checkout of a large repository from becoming memory the machine does
// not have.
func TestTheQueueDropsAtItsBoundAndSaysSo(t *testing.T) {
	waiting := newQueue(2)
	for i := range 5 {
		waiting.put(event{path: fmt.Sprintf("Note%d.md", i)})
	}

	events, over, done := waiting.take()
	if !over {
		t.Fatal("the queue was filled past its bound and did not say so")
	}
	if done {
		t.Error("the queue said the watcher had stopped")
	}
	if len(events) > 2 {
		t.Errorf("the queue held %d events, past its bound of 2", len(events))
	}

	if _, over, _ := waiting.take(); over {
		t.Error("one overflow was reported twice")
	}
}
