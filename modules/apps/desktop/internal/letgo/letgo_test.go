package letgo

import (
	"slices"
	"sync"
	"testing"
	"time"
)

// TestTheStepsRunInTheOrderTheyWereGiven.
func TestTheStepsRunInTheOrderTheyWereGiven(t *testing.T) {
	var ran []string
	mark := func(what string) func() { return func() { ran = append(ran, what) } }

	InOrder(mark("page"), mark("agents"), mark("vault"), mark("under")).Go()

	if want := []string{"page", "agents", "vault", "under"}; !slices.Equal(ran, want) {
		t.Errorf("what the window let go of was %v", ran)
	}
}

// TestTheStepsRunOnceHoweverOftenTheyAreAskedFor. The application's shutdown
// and the return of the run it was made in both ask, and the window is let go
// of once.
func TestTheStepsRunOnceHoweverOftenTheyAreAskedFor(t *testing.T) {
	times := 0
	steps := InOrder(func() { times++ })

	steps.Go()
	steps.Go()
	steps.Go()

	if times != 1 {
		t.Errorf("the step ran %d times", times)
	}
}

// TestAskingWhileTheStepsRunWaitsForThem. A second ask does not go on to close
// what a step still has open.
func TestAskingWhileTheStepsRunWaitsForThem(t *testing.T) {
	held := make(chan struct{})
	over := false
	steps := InOrder(func() {
		<-held
		over = true
	})

	go steps.Go()

	waited := make(chan struct{})
	go func() {
		defer close(waited)
		time.Sleep(50 * time.Millisecond)
		steps.Go()
		if !over {
			t.Error("an ask arriving while the steps ran did not wait for them")
		}
	}()

	close(held)
	select {
	case <-waited:
	case <-time.After(5 * time.Second):
		t.Fatal("the second ask was never answered")
	}
}

// TestEveryoneAskingAtOnceLetsGoOnce.
func TestEveryoneAskingAtOnceLetsGoOnce(t *testing.T) {
	var mu sync.Mutex
	times := 0
	steps := InOrder(func() {
		mu.Lock()
		defer mu.Unlock()
		times++
	})

	var asking sync.WaitGroup
	for range 8 {
		asking.Add(1)
		go func() {
			defer asking.Done()
			steps.Go()
		}()
	}
	asking.Wait()

	mu.Lock()
	defer mu.Unlock()
	if times != 1 {
		t.Errorf("the step ran %d times", times)
	}
}

// TestAWindowHoldingNothingIsLetGoOfWithoutTrouble.
func TestAWindowHoldingNothingIsLetGoOfWithoutTrouble(t *testing.T) {
	InOrder().Go()
}
