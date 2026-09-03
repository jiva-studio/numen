package container

import (
	"errors"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// A document named at the instant the line empties is read. The run that finds
// the line empty gives the turn up under the same lock, so the ask that follows
// is told it began and reads the document itself.
//
// Nothing else reads this line: a document told it was queued with no run to
// read it is a document never read, and a person left waiting on a count that
// never falls.
func TestADocumentNamedAsTheLineEmptiesIsRead(t *testing.T) {
	w := recognising(t, errors.New("no models on this machine"))

	var (
		once  sync.Once
		taken port.Taking
	)
	w.Recognising.idle = func() {
		once.Do(func() { taken = w.Start(somewhere, "b.pdf") })
	}

	if got := w.Start(somewhere, "a.pdf"); got != port.Began {
		t.Fatalf("the first document was not read: %v", got)
	}
	w.settled(t)
	w.Wait()

	if taken != port.Began {
		t.Errorf("the document named as the line emptied was told %v", taken)
	}
	if w.opened() != 2 {
		t.Errorf("a recogniser was opened %d times", w.opened())
	}
	if w.Waiting() != 0 {
		t.Errorf("%d documents were left in line", w.Waiting())
	}
}
