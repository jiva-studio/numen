package eventbus

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func getRealClock() Clock { return testsupport.GetMachine() }

func TestEventBusDebounceTyping(t *testing.T) {
	bus := New(nil, getRealClock(), 50*time.Millisecond, 200*time.Millisecond, nil)
	defer func() {
		if err := bus.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()

	var count atomic.Int32
	unsub := bus.Subscribe("onTyping", func(_ context.Context, _ string) error {
		count.Add(1)
		return nil
	})
	defer unsub()

	for range 10 {
		bus.Publish("onTyping")
		time.Sleep(5 * time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)

	if got := count.Load(); got != 1 {
		t.Fatalf("expected 1 debounced call, got %d", got)
	}
}

func TestEventBusThrottle(t *testing.T) {
	bus := New(nil, getRealClock(), 40*time.Millisecond, 100*time.Millisecond, nil)
	defer func() {
		if err := bus.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()

	var count atomic.Int32
	unsub := bus.Subscribe("onVaultModify", func(_ context.Context, _ string) error {
		count.Add(1)
		return nil
	})
	defer unsub()

	start := getRealClock().Now()
	for getRealClock().Now().Sub(start) < 250*time.Millisecond {
		bus.Publish("onVaultModify")
		time.Sleep(10 * time.Millisecond)
	}

	time.Sleep(80 * time.Millisecond)

	// In 250ms with 100ms throttle, we should receive at least 2 events
	if got := count.Load(); got < 2 {
		t.Fatalf("expected at least 2 throttled calls, got %d", got)
	}
}

func TestEventBusNonBlocking(t *testing.T) {
	bus := New(nil, getRealClock(), 50*time.Millisecond, 200*time.Millisecond, nil)
	defer func() {
		if err := bus.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()

	start := getRealClock().Now()
	for range 100 {
		bus.Publish("onTyping")
	}
	elapsed := getRealClock().Now().Sub(start)

	if elapsed > 50*time.Millisecond {
		t.Fatalf("Publish blocked the caller for %v", elapsed)
	}
}

func TestEventBusFlush(t *testing.T) {
	bus := New(nil, getRealClock(), 500*time.Millisecond, 1000*time.Millisecond, nil)
	defer func() {
		if err := bus.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)

	var received string
	unsub := bus.Subscribe("onIndexChanged", func(_ context.Context, event string) error {
		received = event
		wg.Done()
		return nil
	})
	defer unsub()

	bus.Publish("onIndexChanged:v-1")
	if err := bus.Flush(t.Context()); err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	if received != "onIndexChanged:v-1" {
		t.Fatalf("got %q, want onIndexChanged:v-1", received)
	}
}

func TestEventBusUnsubscribe(t *testing.T) {
	bus := New(nil, getRealClock(), 10*time.Millisecond, 50*time.Millisecond, nil)
	defer func() {
		if err := bus.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()

	var count atomic.Int32
	unsub := bus.Subscribe("testEvent", func(_ context.Context, _ string) error {
		count.Add(1)
		return nil
	})

	bus.Publish("testEvent:1")
	if err := bus.Flush(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := count.Load(); got != 1 {
		t.Fatalf("expected 1 call before unsubscribe, got %d", got)
	}

	unsub()

	bus.Publish("testEvent:2")
	if err := bus.Flush(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := count.Load(); got != 1 {
		t.Fatalf("expected 1 call after unsubscribe, got %d", got)
	}
}

// Typing events all fall under one key, so only the last of them is delivered.
func TestEventBusDebouncesTypingEvents(t *testing.T) {
	bus := New(nil, getRealClock(), 500*time.Millisecond, 1000*time.Millisecond, nil)
	defer func() {
		if err := bus.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()

	var received []string
	var mu sync.Mutex
	unsub := bus.Subscribe("onTyping", func(_ context.Context, event string) error {
		mu.Lock()
		received = append(received, event)
		mu.Unlock()
		return nil
	})
	defer unsub()

	events := []string{"onTyping:a.md", "onTyping:b.md"}
	for _, event := range events {
		bus.Publish(event)
	}
	if err := bus.Flush(t.Context()); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if last := events[len(events)-1]; len(received) != 1 || received[0] != last {
		t.Fatalf("expected last event %s, got %v", last, received)
	}
}

// A delivery that fails after Publish returned is reported, and Flush returns the failure of the delivery it runs.
func TestEventBusReportsTheFailureOfADelivery(t *testing.T) {
	failure := errors.New("the plugin refused the event")
	reported := make(chan error, 1)
	bus := New(nil, getRealClock(), 10*time.Millisecond, 50*time.Millisecond, func(err error) {
		select {
		case reported <- err:
		default:
		}
	})
	defer func() {
		if err := bus.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()
	unsub := bus.Subscribe("failing", func(context.Context, string) error { return failure })
	defer unsub()

	bus.Publish("failing:first")
	select {
	case err := <-reported:
		if !errors.Is(err, failure) {
			t.Errorf("reported %v, want %v", err, failure)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the failed delivery was not reported")
	}

	bus.Publish("failing:second")
	if err := bus.Flush(t.Context()); !errors.Is(err, failure) {
		t.Errorf("Flush returned %v, want %v", err, failure)
	}
}
