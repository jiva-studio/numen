package eventbus

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// Handler handles a dispatched event string.
type Handler func(ctx context.Context, event string) error

// ErrorHandler receives asynchronous dispatch errors.
type ErrorHandler func(err error)

type eventState struct {
	cancel    func() bool
	firstSeen time.Time
	lastSeen  time.Time
	event     string
}

type subscriber struct {
	id      uint64
	handler Handler
}

// Clock gives the time the debounce and throttle windows are measured from.
type Clock interface {
	Now() time.Time
}

// Trigger delivers an event to the plugins that take it.
type Trigger interface {
	TriggerEvent(ctx context.Context, event string) ([]string, error)
}

// Bus debounces and throttles events sent to plugins to protect the UI main thread.
type Bus struct {
	mu               sync.Mutex
	host             Trigger
	debounceDelay    time.Duration
	throttleInterval time.Duration
	pending          map[string]*eventState
	subscribers      map[string][]subscriber
	nextSubID        uint64
	isClosed         bool
	inFlight         sync.WaitGroup
	clock            Clock
	errorHandler     ErrorHandler
}

// New creates a debouncing and throttling event bus. report receives the errors of the deliveries that run after
// Publish returned; a nil report drops them.
func New(host Trigger, clock Clock, debounceDelay, throttleInterval time.Duration, report ErrorHandler) *Bus {
	if debounceDelay <= 0 {
		debounceDelay = 50 * time.Millisecond
	}
	if throttleInterval <= 0 {
		throttleInterval = 250 * time.Millisecond
	}
	return &Bus{
		mu:               sync.Mutex{},
		host:             host,
		debounceDelay:    debounceDelay,
		throttleInterval: throttleInterval,
		pending:          make(map[string]*eventState),
		subscribers:      make(map[string][]subscriber),
		nextSubID:        0,
		isClosed:         false,
		inFlight:         sync.WaitGroup{},
		clock:            clock,
		errorHandler:     report,
	}
}

func (b *Bus) getNow() time.Time {
	return b.clock.Now()
}

func (b *Bus) scheduleDelay(d time.Duration, f func()) func() bool {
	t := time.AfterFunc(d, f)
	return t.Stop
}

func getDebounceKey(event string) string {
	if strings.HasPrefix(event, "onTyping") {
		return "onTyping"
	}
	return event
}

// Publish enqueues an event for debounced and throttled delivery without blocking the caller.
func (b *Bus) Publish(event string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.isClosed || event == "" {
		return
	}

	key := getDebounceKey(event)
	now := b.getNow()

	state, exists := b.pending[key]
	if !exists {
		state = &eventState{
			cancel:    nil,
			firstSeen: now,
			lastSeen:  now,
			event:     event,
		}
		state.cancel = b.scheduleDelay(b.debounceDelay, func() {
			b.dispatchAsyncKey(key)
		})
		b.pending[key] = state
		return
	}

	state.event = event
	state.lastSeen = now

	elapsed := now.Sub(state.firstSeen)
	if elapsed >= b.throttleInterval {
		if state.cancel != nil {
			state.cancel()
			state.cancel = nil
		}
		go b.dispatchAsyncKey(key)
		return
	}

	remainingThrottle := b.throttleInterval - elapsed
	nextDelay := min(b.debounceDelay, remainingThrottle)

	if state.cancel != nil {
		state.cancel()
	}
	state.cancel = b.scheduleDelay(nextDelay, func() {
		b.dispatchAsyncKey(key)
	})
}

// Subscribe registers an event handler for events matching the specified prefix.
// The returned function unregisters the handler.
func (b *Bus) Subscribe(prefix string, handler Handler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextSubID++
	subID := b.nextSubID
	b.subscribers[prefix] = append(b.subscribers[prefix], subscriber{
		id:      subID,
		handler: handler,
	})

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		handlers := b.subscribers[prefix]
		for i, sub := range handlers {
			if sub.id == subID {
				b.subscribers[prefix] = append(handlers[:i], handlers[i+1:]...)
				if len(b.subscribers[prefix]) == 0 {
					delete(b.subscribers, prefix)
				}
				break
			}
		}
	}
}

// Flush dispatches all currently queued pending events immediately.
func (b *Bus) Flush(ctx context.Context) error {
	b.mu.Lock()
	keys := make([]string, 0, len(b.pending))
	for k := range b.pending {
		keys = append(keys, k)
	}
	b.mu.Unlock()

	var errs []error
	for _, k := range keys {
		if err := b.flushKey(ctx, k); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Close stops all pending timers and flushes queued events.
func (b *Bus) Close(ctx context.Context) error {
	b.mu.Lock()
	if b.isClosed {
		b.mu.Unlock()
		return nil
	}
	b.isClosed = true
	b.mu.Unlock()

	flushErr := b.Flush(ctx)

	drained := make(chan struct{})
	go func() {
		b.inFlight.Wait()
		close(drained)
	}()
	select {
	case <-drained:
		return flushErr
	case <-ctx.Done():
		return errors.Join(flushErr, fmt.Errorf("wait for event dispatch: %w", ctx.Err()))
	}
}

func (b *Bus) beginDispatch() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.isClosed {
		return false
	}
	b.inFlight.Add(1)
	return true
}

func (b *Bus) dispatchAsyncKey(key string) {
	if !b.beginDispatch() {
		return
	}
	defer b.inFlight.Done()
	if err := b.flushKey(context.Background(), key); err != nil {
		b.reportError(err)
	}
}

func (b *Bus) reportError(err error) {
	if b.errorHandler != nil {
		b.errorHandler(err)
	}
}

func (b *Bus) flushKey(ctx context.Context, key string) error {
	b.mu.Lock()
	state, exists := b.pending[key]
	if !exists {
		b.mu.Unlock()
		return nil
	}
	if state.cancel != nil {
		state.cancel()
		state.cancel = nil
	}
	event := state.event
	delete(b.pending, key)

	handlers := b.findMatchingHandlers(event)
	host := b.host
	b.mu.Unlock()

	var errs []error
	for _, h := range handlers {
		if err := h(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}

	if host != nil {
		if _, err := host.TriggerEvent(plugindomain.WithBackground(ctx, plugindomain.WhoEvents), event); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (b *Bus) findMatchingHandlers(event string) []Handler {
	var result []Handler
	for prefix, list := range b.subscribers {
		if prefix == "" || prefix == "*" || strings.HasPrefix(event, prefix) {
			for _, sub := range list {
				result = append(result, sub.handler)
			}
		}
	}
	return result
}
