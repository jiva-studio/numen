package broadcast

import (
	"testing"
)

// A listener reading while tell runs on another goroutine receives numbers
// in the order they were told, never out of order and never twice.
func TestConcurrentTellsReachAConcurrentReaderInOrder(t *testing.T) {
	bcast := Broadcaster[int]{Capacity: 16}
	line, isDone := bcast.Subscribe()

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
		bcast.Publish(n)
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
}

func TestInflightBeginsAndFinishes(t *testing.T) {
	var inf Inflight
	if !inf.Begin() {
		t.Fatal("expected begin to succeed")
	}
	if inf.Count() != 1 {
		t.Fatalf("expected count 1, got %d", inf.Count())
	}
	inf.Finish()
	if inf.Count() != 0 {
		t.Fatalf("expected count 0, got %d", inf.Count())
	}
	idle := inf.Seal()
	select {
	case <-idle:
	default:
		t.Fatal("expected idle to be closed immediately")
	}
	if inf.Begin() {
		t.Fatal("expected begin to fail after seal")
	}
}
