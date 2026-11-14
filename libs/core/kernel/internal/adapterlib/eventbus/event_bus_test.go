package eventbus

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func getRealClock() Clock { return testsupport.GetMachine() }

func TestEventBusDebounceTyping(t *testing.T) {
	bus := New(nil, getRealClock(), 50*time.Millisecond, 200*time.Millisecond)
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
	bus := New(nil, getRealClock(), 40*time.Millisecond, 100*time.Millisecond)
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
	bus := New(nil, getRealClock(), 50*time.Millisecond, 200*time.Millisecond)
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
	bus := New(nil, getRealClock(), 500*time.Millisecond, 1000*time.Millisecond)
	defer func() {
		if err := bus.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)

	var received string
	unsub := bus.Subscribe("onSearchSourceReady", func(_ context.Context, event string) error {
		received = event
		wg.Done()
		return nil
	})
	defer unsub()

	bus.Publish("onSearchSourceReady:v-1")
	if err := bus.Flush(t.Context()); err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	if received != "onSearchSourceReady:v-1" {
		t.Fatalf("got %q, want onSearchSourceReady:v-1", received)
	}
}

func TestEventBusUnsubscribe(t *testing.T) {
	bus := New(nil, getRealClock(), 10*time.Millisecond, 50*time.Millisecond)
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

// TestEventBusDebouncesEventsOfOneKey. A key extractor and a debounce rule
// each put several events under one key, and only the last of them is delivered.
func TestEventBusDebouncesEventsOfOneKey(t *testing.T) {
	cases := []struct {
		name   string
		option Option
		prefix string
		events []string
	}{
		{
			name: "key extractor",
			option: WithKeyExtractor(func(event string) string {
				if strings.HasPrefix(event, "customGroup:") {
					return "customGroup"
				}
				return event
			}),
			prefix: "customGroup:",
			events: []string{"customGroup:item1", "customGroup:item2"},
		},
		{
			name: "debounce rule",
			option: WithDebounceRule(func(event string) (string, bool) {
				if strings.HasPrefix(event, "ruleA:") {
					return "ruleA-shared", true
				}
				return "", false
			}),
			prefix: "ruleA:",
			events: []string{"ruleA:1", "ruleA:2"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bus := New(nil, getRealClock(), 500*time.Millisecond, 1000*time.Millisecond, c.option)
			defer func() {
				if err := bus.Close(t.Context()); err != nil {
					t.Error(err)
				}
			}()

			var received []string
			var mu sync.Mutex
			unsub := bus.Subscribe(c.prefix, func(_ context.Context, event string) error {
				mu.Lock()
				received = append(received, event)
				mu.Unlock()
				return nil
			})
			defer unsub()

			for _, event := range c.events {
				bus.Publish(event)
			}
			if err := bus.Flush(t.Context()); err != nil {
				t.Fatalf("flush failed: %v", err)
			}

			mu.Lock()
			defer mu.Unlock()
			if last := c.events[len(c.events)-1]; len(received) != 1 || received[0] != last {
				t.Fatalf("expected last event %s, got %v", last, received)
			}
		})
	}
}

func TestEventBusErrorHandling(t *testing.T) {
	expectedErr := context.DeadlineExceeded
	var asyncErr atomic.Pointer[error]

	bus := New(nil, getRealClock(), 10*time.Millisecond, 50*time.Millisecond,
		WithErrorHandler(func(err error) {
			asyncErr.Store(&err)
		}),
	)
	defer func() {
		if err := bus.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()

	unsub := bus.Subscribe("failing", func(_ context.Context, _ string) error {
		return expectedErr
	})
	defer unsub()

	bus.Publish("failing:event")
	err := bus.Flush(t.Context())
	if err == nil {
		t.Fatal("expected Flush to return error, got nil")
	}
	if !strings.Contains(err.Error(), expectedErr.Error()) {
		t.Fatalf("expected error containing %v, got %v", expectedErr, err)
	}
}
