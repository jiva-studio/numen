package letgo

import (
	"slices"
	"sync"
	"testing"
	"time"
)

func TestTheStepsRunInTheOrderTheyWereGiven(t *testing.T) {
	var ran []string
	mark := func(what string) func() { return func() { ran = append(ran, what) } }

	InOrder(mark("page"), mark("agents"), mark("vault"), mark("under")).Go()

	if want := []string{"page", "agents", "vault", "under"}; !slices.Equal(ran, want) {
		t.Errorf("what the window let go of was %v", ran)
	}
}

// The application's shutdown and the return of the run it was made in both ask,
// and the window is let go of once.
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

// An ask that returned while a step was still running would be a run going on
// to close what that step has open.
func TestAskingWhileTheStepsRunWaitsForThem(t *testing.T) {
	running := make(chan struct{})
	held := make(chan struct{})
	steps := InOrder(func() {
		close(running)
		<-held
	})

	go steps.Go()
	<-running

	returned := make(chan struct{})
	go func() {
		steps.Go()
		close(returned)
	}()

	select {
	case <-returned:
		t.Fatal("an ask arriving while the steps ran was answered before they were over")
	case <-time.After(300 * time.Millisecond):
	}

	close(held)
	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		t.Fatal("the ask was never answered")
	}
}

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

func TestAWindowHoldingNothingIsLetGoOfWithoutTrouble(t *testing.T) {
	InOrder().Go()
}
