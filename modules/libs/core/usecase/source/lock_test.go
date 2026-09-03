package source

import (
	"context"
	"testing"
	"time"
)

// Work a person is sitting in front of takes the lock before work the vault set
// itself, whatever order they arrived in.
func TestTheLockGoesToWhatWasAskedFor(t *testing.T) {
	var g lock

	held, err := g.acquire(t.Context(), false, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Both wait for the lock, the unasked one first.
	took := make(chan bool, 2)
	standing := func(asked bool) {
		waiting := make(chan struct{})
		go func() {
			release, err := g.acquire(t.Context(), asked, func() { close(waiting) })
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
				t.Errorf("the lock went to asked=%v, %d in", got, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("nothing took the lock, %d in", want)
		}
	}
}

// A run that gave up while waiting takes no lock, and the one behind it is not
// left standing.
func TestARunThatLeavesHandsTheLockOn(t *testing.T) {
	var g lock

	held, err := g.acquire(context.Background(), true, nil)
	if err != nil {
		t.Fatal(err)
	}

	gone, stop := context.WithCancel(context.Background())
	left := make(chan error, 1)
	waiting := make(chan struct{})
	go func() {
		_, err := g.acquire(gone, true, func() { close(waiting) })
		left <- err
	}()
	<-waiting

	after := make(chan struct{})
	standing := make(chan struct{})
	go func() {
		release, err := g.acquire(context.Background(), true, func() { close(standing) })
		if err != nil {
			return
		}
		close(after)
		release()
	}()
	<-standing

	stop()
	if err := <-left; err == nil {
		t.Error("a run that gave up was handed the lock")
	}
	held()

	select {
	case <-after:
	case <-time.After(time.Second):
		t.Fatal("the lock was dropped when the run before it left")
	}
}
