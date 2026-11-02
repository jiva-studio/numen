package serialqueue

import (
	"testing"
	"time"
)

func TestQueueRunsInTheOrderGiven(t *testing.T) {
	updates := New()
	var order []int
	signal := make(chan struct{})

	updates.Go(func() {
		time.Sleep(50 * time.Millisecond)
		order = append(order, 1)
	})
	updates.Go(func() { order = append(order, 2) })
	updates.Go(func() {
		order = append(order, 3)
		close(signal)
	})

	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal("the updates did not run")
	}
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Fatalf("order = %v, want [1 2 3]", order)
	}
}
