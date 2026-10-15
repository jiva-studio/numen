package plugin

import (
	"context"
	"strings"
	"sync"
	"time"
)

// EventHandler handles a dispatched event string.
type EventHandler func(ctx context.Context, event string) error

type eventState struct {
	timer      *time.Timer
	firstSeen  time.Time
	lastSeen   time.Time
	event      string
}

// EventBus debounces and throttles events sent to plugins to protect the UI main thread.
type EventBus struct {
	mu               sync.Mutex
	host             *Host
	debounceDelay    time.Duration
	throttleInterval time.Duration
	pending          map[string]*eventState
	subscribers      map[string][]EventHandler
	isClosed         bool
}

// NewEventBus creates a debouncing and throttling event bus.
func NewEventBus(host *Host, debounceDelay, throttleInterval time.Duration) *EventBus {
	if debounceDelay <= 0 {
		debounceDelay = 50 * time.Millisecond
	}
	if throttleInterval <= 0 {
		throttleInterval = 250 * time.Millisecond
	}
	return &EventBus{
		mu:               sync.Mutex{},
		host:             host,
		debounceDelay:    debounceDelay,
		throttleInterval: throttleInterval,
		pending:          make(map[string]*eventState),
		subscribers:      make(map[string][]EventHandler),
		isClosed:         false,
	}
}

// Publish enqueues an event for debounced and throttled delivery without blocking the caller.
func (b *EventBus) Publish(event string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.isClosed || event == "" {
		return
	}

	key := eventKey(event)
	now := time.Now()

	state, exists := b.pending[key]
	if !exists {
		state = &eventState{
			timer:     nil,
			firstSeen: now,
			lastSeen:  now,
			event:     event,
		}
		state.timer = time.AfterFunc(b.debounceDelay, func() {
			b.flushKey(context.Background(), key)
		})
		b.pending[key] = state
		return
	}

	state.event = event
	state.lastSeen = now

	elapsed := now.Sub(state.firstSeen)
	if elapsed >= b.throttleInterval {
		if state.timer != nil {
			state.timer.Stop()
		}
		go b.flushKey(context.Background(), key)
		return
	}

	remainingThrottle := b.throttleInterval - elapsed
	nextDelay := b.debounceDelay
	if nextDelay > remainingThrottle {
		nextDelay = remainingThrottle
	}

	if state.timer != nil {
		state.timer.Stop()
	}
	state.timer = time.AfterFunc(nextDelay, func() {
		b.flushKey(context.Background(), key)
	})
}

// Subscribe registers an event handler for events matching the specified prefix.
// The returned function unregisters the handler.
func (b *EventBus) Subscribe(prefix string, handler EventHandler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[prefix] = append(b.subscribers[prefix], handler)

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		handlers := b.subscribers[prefix]
		for i, h := range handlers {
			if &h == &handler {
				b.subscribers[prefix] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
	}
}

// Flush dispatches all currently queued pending events immediately.
func (b *EventBus) Flush(ctx context.Context) {
	b.mu.Lock()
	keys := make([]string, 0, len(b.pending))
	for k := range b.pending {
		keys = append(keys, k)
	}
	b.mu.Unlock()

	for _, k := range keys {
		b.flushKey(ctx, k)
	}
}

// Close stops all pending timers and flushes queued events.
func (b *EventBus) Close(ctx context.Context) {
	b.mu.Lock()
	if b.isClosed {
		b.mu.Unlock()
		return
	}
	b.isClosed = true
	b.mu.Unlock()

	b.Flush(ctx)
}

func (b *EventBus) flushKey(ctx context.Context, key string) {
	b.mu.Lock()
	state, exists := b.pending[key]
	if !exists {
		b.mu.Unlock()
		return
	}
	if state.timer != nil {
		state.timer.Stop()
	}
	event := state.event
	delete(b.pending, key)

	handlers := b.findMatchingHandlers(event)
	host := b.host
	b.mu.Unlock()

	for _, h := range handlers {
		_ = h(ctx, event)
	}

	if host != nil {
		_, _ = host.TriggerEvent(ctx, event)
	}
}

func (b *EventBus) findMatchingHandlers(event string) []EventHandler {
	var result []EventHandler
	for prefix, list := range b.subscribers {
		if prefix == "" || prefix == "*" || strings.HasPrefix(event, prefix) {
			result = append(result, list...)
		}
	}
	return result
}

func eventKey(event string) string {
	if strings.HasPrefix(event, "onTyping") {
		return "onTyping"
	}
	return event
}
