package container

import (
	"context"
	"testing"
	"time"
)

// Work a person is sitting in front of takes the turn before work the vault set
// itself, whatever order they arrived in.
func TestTheTurnGoesToWhatWasAskedFor(t *testing.T) {
	var g turn

	held, err := g.take(t.Context(), false, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Both stand behind the turn, the unasked one first.
	took := make(chan bool, 2)
	standing := func(asked bool) {
		waiting := make(chan struct{})
		go func() {
			release, err := g.take(t.Context(), asked, func() { close(waiting) })
			if err != nil {
				return
			}
			took <- asked
			release()
		}()
		<-waiting
	}
	standing(false)
	standing(true)

	held()
	for want := range []bool{true, false} {
		select {
		case got := <-took:
			if got != (want == 0) {
				t.Errorf("the turn %d went to asked=%v", want, got)
			}
		case <-time.After(time.Second):
			t.Fatalf("nothing took turn %d", want)
		}
	}
}

// A run that gave up while waiting takes no turn, and the one behind it is not
// left standing.
func TestARunThatLeavesHandsTheTurnOn(t *testing.T) {
	var g turn

	held, err := g.take(context.Background(), true, nil)
	if err != nil {
		t.Fatal(err)
	}

	gone, stop := context.WithCancel(context.Background())
	left := make(chan error, 1)
	waiting := make(chan struct{})
	go func() {
		_, err := g.take(gone, true, func() { close(waiting) })
		left <- err
	}()
	<-waiting

	after := make(chan struct{})
	standing := make(chan struct{})
	go func() {
		release, err := g.take(context.Background(), true, func() { close(standing) })
		if err != nil {
			return
		}
		close(after)
		release()
	}()
	<-standing

	stop()
	if err := <-left; err == nil {
		t.Error("a run that gave up was handed the turn")
	}
	held()

	select {
	case <-after:
	case <-time.After(time.Second):
		t.Fatal("the turn was dropped when the run before it left")
	}
}
