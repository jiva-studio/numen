package eventbus

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/clock"
)

var realClock clock.System

func TestEventBusDebounceTyping(t *testing.T) {
	bus := New(nil, realClock, 50*time.Millisecond, 200*time.Millisecond)
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
	bus := New(nil, realClock, 40*time.Millisecond, 100*time.Millisecond)
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

	start := realClock.Now()
	for realClock.Now().Sub(start) < 250*time.Millisecond {
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
	bus := New(nil, realClock, 50*time.Millisecond, 200*time.Millisecond)
	defer func() {
		if err := bus.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()

	start := realClock.Now()
	for i := 0; i < 100; i++ {
		bus.Publish("onTyping")
	}
	elapsed := realClock.Now().Sub(start)

	if elapsed > 50*time.Millisecond {
		t.Fatalf("Publish blocked the caller for %v", elapsed)
	}
}

func TestEventBusFlush(t *testing.T) {
	bus := New(nil, realClock, 500*time.Millisecond, 1000*time.Millisecond)
	defer func() {
		if err := bus.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()

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
	if err := bus.Flush(t.Context()); err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	if received != "onFile:notes/test.md" {
		t.Fatalf("got %q, want onFile:notes/test.md", received)
	}
}

func TestEventBusUnsubscribe(t *testing.T) {
	bus := New(nil, realClock, 10*time.Millisecond, 50*time.Millisecond)
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

type mockScheduler struct {
	mu      sync.Mutex
	current time.Time
	timers  []*mockTimer
}

type mockTimer struct {
	trigger time.Time
	f       func()
	stopped bool
}

func newMockScheduler(start time.Time) *mockScheduler {
	return &mockScheduler{
		current: start,
		timers:  make([]*mockTimer, 0),
	}
}

func (s *mockScheduler) Now() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}

func (s *mockScheduler) scheduleDelay(d time.Duration, f func()) func() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := &mockTimer{
		trigger: s.current.Add(d),
		f:       f,
		stopped: false,
	}
	s.timers = append(s.timers, t)
	return func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		if t.stopped {
			return false
		}
		t.stopped = true
		return true
	}
}

func (s *mockScheduler) advance(d time.Duration) {
	s.mu.Lock()
	s.current = s.current.Add(d)
	var due []func()
	for _, t := range s.timers {
		if !t.stopped && !t.trigger.After(s.current) {
			t.stopped = true
			due = append(due, t.f)
		}
	}
	s.mu.Unlock()

	for _, f := range due {
		f()
	}
}

func TestEventBusDeterministicDebounce(t *testing.T) {
	sched := newMockScheduler(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	bus := New(nil, sched, 50*time.Millisecond, 200*time.Millisecond,
		WithDelay(sched.scheduleDelay),
	)
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

	for i := 0; i < 5; i++ {
		bus.Publish("onTyping")
		sched.advance(5 * time.Millisecond)
	}

	if got := count.Load(); got != 0 {
		t.Fatalf("expected 0 calls before debounce elapsed, got %d", got)
	}

	sched.advance(60 * time.Millisecond)

	if got := count.Load(); got != 1 {
		t.Fatalf("expected 1 debounced call after advance, got %d", got)
	}
}

func TestEventBusConfigurableKeyExtractor(t *testing.T) {
	bus := New(nil, realClock, 500*time.Millisecond, 1000*time.Millisecond,
		WithKeyExtractor(func(event string) string {
			if strings.HasPrefix(event, "customGroup:") {
				return "customGroup"
			}
			return event
		}),
	)
	defer func() {
		if err := bus.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()

	var received []string
	var mu sync.Mutex
	unsub := bus.Subscribe("customGroup:", func(_ context.Context, event string) error {
		mu.Lock()
		received = append(received, event)
		mu.Unlock()
		return nil
	})
	defer unsub()

	bus.Publish("customGroup:item1")
	bus.Publish("customGroup:item2")
	if err := bus.Flush(t.Context()); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 || received[0] != "customGroup:item2" {
		t.Fatalf("expected last event customGroup:item2, got %v", received)
	}
}

func TestEventBusDebounceRule(t *testing.T) {
	bus := New(nil, realClock, 500*time.Millisecond, 1000*time.Millisecond,
		WithDebounceRule(func(event string) (string, bool) {
			if strings.HasPrefix(event, "ruleA:") {
				return "ruleA-shared", true
			}
			return "", false
		}),
	)
	defer func() {
		if err := bus.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()

	var received []string
	var mu sync.Mutex
	unsub := bus.Subscribe("ruleA:", func(_ context.Context, event string) error {
		mu.Lock()
		received = append(received, event)
		mu.Unlock()
		return nil
	})
	defer unsub()

	bus.Publish("ruleA:1")
	bus.Publish("ruleA:2")
	if err := bus.Flush(t.Context()); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 || received[0] != "ruleA:2" {
		t.Fatalf("expected debounced ruleA:2, got %v", received)
	}
}

func TestEventBusErrorHandling(t *testing.T) {
	expectedErr := context.DeadlineExceeded
	var asyncErr atomic.Pointer[error]

	bus := New(nil, realClock, 10*time.Millisecond, 50*time.Millisecond,
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
