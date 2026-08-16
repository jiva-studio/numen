package webui

import "testing"

// A listener that fell behind is told to read everything again.
func TestALaggingListenerIsToldToReadEverythingAgain(t *testing.T) {
	following := following()
	line, done := following.listen()
	defer done()

	// Fill what it may be owed, then one more it cannot take.
	for range cap(line) {
		following.tell(changed{paths: []string{"notes/one.md"}})
	}
	following.tell(changed{paths: []string{"notes/missed.md"}})

	for range cap(line) {
		<-line
	}
	following.tell(changed{paths: []string{"notes/two.md"}})

	if last := <-line; !last.reload {
		t.Errorf("a listener that missed one was handed %+v", last)
	}
}

// What matters is the last note asked for: one asked for while a listener is
// busy replaces the one it has not read.
func TestTheLastNoteAskedForIsTheOneWaiting(t *testing.T) {
	focusing := focusing()
	line, done := focusing.listen()
	defer done()

	focusing.tell("notes/one.md")
	focusing.tell("notes/two.md")
	focusing.tell("notes/three.md")

	if waiting := <-line; waiting != "notes/three.md" {
		t.Errorf("the listener was handed %q", waiting)
	}
	if len(line) != 0 {
		t.Errorf("%d notes are queued behind it", len(line))
	}
}

func TestNobodyIsToldAfterTheyStopListening(t *testing.T) {
	focusing := focusing()
	line, done := focusing.listen()
	done()

	focusing.tell("notes/one.md")

	if _, open := <-line; open {
		t.Error("a closed line was written to")
	}
}
