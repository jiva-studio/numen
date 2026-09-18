package editor

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// A listener that fell behind is told to read everything again.
func TestALaggingListenerIsToldToReadEverythingAgain(t *testing.T) {
	following := newChangeAudience()
	line, done := following.listen()
	defer done()

	// Fill what it may be owed, then one more it cannot take.
	for range cap(line) {
		following.tell(change{paths: []string{"notes/one.md"}})
	}
	following.tell(change{paths: []string{"notes/missed.md"}})

	for range cap(line) {
		<-line
	}
	following.tell(change{paths: []string{"notes/two.md"}})

	if last := <-line; !last.shouldReload {
		t.Errorf("a listener that missed one was handed %+v", last)
	}
}

// What matters is the last place asked for: one asked for while a listener is
// busy replaces the one it has not read.
func TestTheLastPlaceAskedForIsTheOneWaiting(t *testing.T) {
	focusing := newPlaceAudience()
	line, done := focusing.listen()
	defer done()

	focusing.tell(domain.Place{Path: "notes/one.md"})
	focusing.tell(domain.Place{Path: "notes/two.md"})
	focusing.tell(domain.Place{Path: "library/A Book.epub", Spans: []domain.Span{{From: 1200, To: 1280}}})

	want := domain.Place{Path: "library/A Book.epub", Spans: []domain.Span{{From: 1200, To: 1280}}}
	if waiting := <-line; !reflect.DeepEqual(waiting, want) {
		t.Errorf("the listener was handed %+v", waiting)
	}
	if len(line) != 0 {
		t.Errorf("%d places are queued behind it", len(line))
	}
}

func TestNobodyIsToldAfterTheyStopListening(t *testing.T) {
	focusing := newPlaceAudience()
	line, done := focusing.listen()
	done()

	focusing.tell(domain.Place{Path: "notes/one.md"})

	if _, open := <-line; open {
		t.Error("a closed line was written to")
	}
}
