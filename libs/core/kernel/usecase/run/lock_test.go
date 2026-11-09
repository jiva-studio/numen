package run

import (
	"context"
	"testing"
	"time"
)

// Work the user is waiting on takes the lock before background work of the
// vault, regardless of arrival order.
func TestLock_GrantsToRequestedHolderFirst(t *testing.T) {
	var g Lock

	target, err := g.Acquire(t.Context(), false, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Both wait for the lock, the unasked one first.
	took := make(chan bool, 2)
	queue := func(requested bool) {
		waitCh := make(chan struct{})
		go func() {
			release, err := g.Acquire(t.Context(), requested, func() { close(waitCh) })
			if err != nil {
				return
			}
			took <- requested
			release()
		}()
		<-waitCh
	}
	queue(false)
	queue(true)

	target()
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
func TestARunThatPreservesPassesTheLockOn(t *testing.T) {
	var g Lock

	target, err := g.Acquire(t.Context(), true, nil)
	if err != nil {
		t.Fatal(err)
	}

	removed, stop := context.WithCancel(t.Context())
	remaining := make(chan error, 1)
	waitCh := make(chan struct{})
	go func() {
		_, err := g.Acquire(removed, true, func() { close(waitCh) })
		remaining <- err
	}()
	<-waitCh

	after := make(chan struct{})
	queued := make(chan struct{})
	go func() {
		release, err := g.Acquire(t.Context(), true, func() { close(queued) })
		if err != nil {
			return
		}
		close(after)
		release()
	}()
	<-queued

	stop()
	if err := <-remaining; err == nil {
		t.Error("a run that gave up was handed the lock")
	}
	target()

	select {
	case <-after:
	case <-time.After(time.Second):
		t.Fatal("the lock was dropped when the run before it left")
	}
}
