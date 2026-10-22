package editor

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// A listener that fell behind is told to read everything again.
func TestALaggingListenerIsReportedToReadEverythingAgain(t *testing.T) {
	following := newChangeBroadcaster()
	line, isDone := following.subscribe()
	defer isDone()

	// Fill what it may be owed, then one more it cannot take.
	room := following.getCapacity()
	for range room {
		following.publish(change{paths: []string{"notes/one.md"}})
	}
	following.publish(change{paths: []string{"notes/missed.md"}})

	for range room {
		<-line
	}
	following.publish(change{paths: []string{"notes/two.md"}})

	if last := <-line; !last.shouldReload {
		t.Errorf("a listener that missed one was handed %+v", last)
	}
}

// What matters is the last place asked for: one asked for while a listener is
// busy replaces the one it has not read.
func TestTheLastPlaceAskedForIsTheOneWaiting(t *testing.T) {
	focusing := newPlaceBroadcaster()
	line, isDone := focusing.subscribe()
	defer isDone()

	focusing.publish(domain.Place{Path: "notes/one.md"})
	focusing.publish(domain.Place{Path: "notes/two.md"})
	focusing.publish(domain.Place{Path: "library/A Book.epub", Spans: []domain.ByteSpan{{From: 1200, To: 1280}}})

	want := domain.Place{Path: "library/A Book.epub", Spans: []domain.ByteSpan{{From: 1200, To: 1280}}}
	if received := <-line; !reflect.DeepEqual(received, want) {
		t.Errorf("the listener was handed %+v", received)
	}
	select {
	case extra := <-line:
		t.Errorf("a place was queued behind it: %+v", extra)
	default:
	}
}

func TestNobodyIsReportedAfterTheyStopListening(t *testing.T) {
	focusing := newPlaceBroadcaster()
	line, isDone := focusing.subscribe()
	isDone()

	focusing.publish(domain.Place{Path: "notes/one.md"})

	if _, open := <-line; open {
		t.Error("a closed line was written to")
	}
}

// A listener reading while tell runs on another goroutine is handed numbers
// in the order they were told, never out of order and never twice.
func TestConcurrentTellsReachAConcurrentReaderInOrder(t *testing.T) {
	bcast := broadcaster[int]{capacity: 16}
	line, isDone := bcast.subscribe()

	const total = 20000

	var received []int
	read := make(chan struct{})
	receivedFirst := make(chan struct{})
	go func() {
		defer close(read)
		for v := range line {
			if len(received) == 0 {
				close(receivedFirst)
			}
			received = append(received, v)
		}
	}()

	for n := range total {
		bcast.publish(n)
	}
	<-receivedFirst
	isDone()
	<-read

	last := -1
	for _, v := range received {
		if v <= last {
			t.Fatalf("got %d after %d: out of order or repeated", v, last)
		}
		last = v
	}
	if len(received) == 0 {
		t.Fatal("the reader was handed nothing")
	}
}
