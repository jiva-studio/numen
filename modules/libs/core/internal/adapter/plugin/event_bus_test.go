package plugin

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEventBusDebounceTyping(t *testing.T) {
	bus := NewEventBus(nil, 50*time.Millisecond, 200*time.Millisecond)
	defer bus.Close(t.Context())

	var count atomic.Int32
	unsub := bus.Subscribe("onTyping", func(_ context.Context, _ string) error {
		count.Add(1)
		return nil
	})
	defer unsub()

	for i := 0; i < 10; i++ {
		bus.Publish("onTyping")
		time.Sleep(5 * time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)

	if got := count.Load(); got != 1 {
		t.Fatalf("expected 1 debounced call, got %d", got)
	}
}

func TestEventBusThrottle(t *testing.T) {
	bus := NewEventBus(nil, 40*time.Millisecond, 100*time.Millisecond)
	defer bus.Close(t.Context())

	var count atomic.Int32
	unsub := bus.Subscribe("onVaultModify", func(_ context.Context, _ string) error {
		count.Add(1)
		return nil
	})
	defer unsub()

	start := time.Now()
	for time.Since(start) < 250*time.Millisecond {
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
	bus := NewEventBus(nil, 50*time.Millisecond, 200*time.Millisecond)
	defer bus.Close(t.Context())

	start := time.Now()
	for i := 0; i < 100; i++ {
		bus.Publish("onTyping")
	}
	elapsed := time.Since(start)

	if elapsed > 50*time.Millisecond {
		t.Fatalf("Publish blocked the caller for %v", elapsed)
	}
}

func TestEventBusFlush(t *testing.T) {
	bus := NewEventBus(nil, 500*time.Millisecond, 1000*time.Millisecond)
	defer bus.Close(t.Context())

	var wg sync.WaitGroup
	wg.Add(1)

	var received string
	unsub := bus.Subscribe("onFile", func(_ context.Context, event string) error {
		received = event
		wg.Done()
		return nil
	})
	defer unsub()

	bus.Publish("onFile:notes/test.md")
	bus.Flush(t.Context())

	wg.Wait()

	if received != "onFile:notes/test.md" {
		t.Fatalf("got %q, want onFile:notes/test.md", received)
	}
}
