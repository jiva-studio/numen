package plugin

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// EventHandler handles a dispatched event string.
type EventHandler func(ctx context.Context, event string) error

// KeyExtractor maps an event to a debounce group key.
type KeyExtractor func(event string) string

// DebounceRule defines custom key extraction for an event.
type DebounceRule func(event string) (key string, matches bool)

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
	handler EventHandler
}

// EventBusOption configures an EventBus instance.
type EventBusOption func(*EventBus)

// EventBus debounces and throttles events sent to plugins to protect the UI main thread.
type EventBus struct {
	mu               sync.Mutex
	host             *Host
	debounceDelay    time.Duration
	throttleInterval time.Duration
	pending          map[string]*eventState
	subscribers      map[string][]subscriber
	nextSubID        uint64
	isClosed         bool
	clock            port.Clock
	delay            port.Delay
	keyExtractor     KeyExtractor
	debounceRules    []DebounceRule
	errorHandler     ErrorHandler
}

// WithClock sets the clock port used for time measurements.
func WithClock(clock port.Clock) EventBusOption {
	return func(b *EventBus) {
		b.clock = clock
	}
}

// WithDelay sets the delay port used for scheduling timers.
func WithDelay(delay port.Delay) EventBusOption {
	return func(b *EventBus) {
		b.delay = delay
	}
}

// WithKeyExtractor sets the key extractor used for debounce grouping.
func WithKeyExtractor(extractor KeyExtractor) EventBusOption {
	return func(b *EventBus) {
		b.keyExtractor = extractor
	}
}

// WithDebounceRule adds a debounce rule for key extraction.
func WithDebounceRule(rule DebounceRule) EventBusOption {
	return func(b *EventBus) {
		b.debounceRules = append(b.debounceRules, rule)
	}
}

// WithErrorHandler sets the error handler for asynchronous event dispatch failures.
func WithErrorHandler(handler ErrorHandler) EventBusOption {
	return func(b *EventBus) {
		b.errorHandler = handler
	}
}

// NewEventBus creates a debouncing and throttling event bus.
func NewEventBus(host *Host, debounceDelay, throttleInterval time.Duration, opts ...EventBusOption) *EventBus {
	if debounceDelay <= 0 {
		debounceDelay = 50 * time.Millisecond
	}
	if throttleInterval <= 0 {
		throttleInterval = 250 * time.Millisecond
	}
	bus := &EventBus{
		mu:               sync.Mutex{},
		host:             host,
		debounceDelay:    debounceDelay,
		throttleInterval: throttleInterval,
		pending:          make(map[string]*eventState),
		subscribers:      make(map[string][]subscriber),
		nextSubID:        0,
		isClosed:         false,
		clock:            nil,
		delay:            nil,
		keyExtractor:     nil,
		debounceRules:    nil,
		errorHandler:     nil,
	}
	for _, opt := range opts {
		opt(bus)
	}
	return bus
}

// SetClock sets the clock port used for time measurements.
func (b *EventBus) SetClock(clock port.Clock) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clock = clock
}

// SetDelay sets the delay port used for scheduling timers.
func (b *EventBus) SetDelay(delay port.Delay) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.delay = delay
}

// SetKeyExtractor sets the key extractor used for debounce grouping.
func (b *EventBus) SetKeyExtractor(extractor KeyExtractor) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.keyExtractor = extractor
}

// AddDebounceRule registers an additional debounce rule.
func (b *EventBus) AddDebounceRule(rule DebounceRule) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.debounceRules = append(b.debounceRules, rule)
}

// SetErrorHandler sets the callback for asynchronous errors.
func (b *EventBus) SetErrorHandler(handler ErrorHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.errorHandler = handler
}

func (b *EventBus) getNow() time.Time {
	if b.clock != nil {
		return b.clock()
	}
	return time.Now()
}

func (b *EventBus) scheduleDelay(d time.Duration, f func()) func() bool {
	if b.delay != nil {
		return b.delay(d, f)
	}
	t := time.AfterFunc(d, f)
	return t.Stop
}

// DefaultKeyExtractor groups typing events and leaves other events distinct.
func DefaultKeyExtractor(event string) string {
	if strings.HasPrefix(event, "onTyping") {
		return "onTyping"
	}
	return event
}

func (b *EventBus) resolveKey(event string) string {
	for _, rule := range b.debounceRules {
		if key, matches := rule(event); matches {
			return key
		}
	}
	if b.keyExtractor != nil {
		return b.keyExtractor(event)
	}
	return DefaultKeyExtractor(event)
}

// Publish enqueues an event for debounced and throttled delivery without blocking the caller.
func (b *EventBus) Publish(event string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.isClosed || event == "" {
		return
	}

	key := b.resolveKey(event)
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
	nextDelay := b.debounceDelay
	if nextDelay > remainingThrottle {
		nextDelay = remainingThrottle
	}

	if state.cancel != nil {
		state.cancel()
	}
	state.cancel = b.scheduleDelay(nextDelay, func() {
		b.dispatchAsyncKey(key)
	})
}

// Subscribe registers an event handler for events matching the specified prefix.
// The returned function unregisters the handler.
func (b *EventBus) Subscribe(prefix string, handler EventHandler) func() {
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
func (b *EventBus) Flush(ctx context.Context) error {
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
func (b *EventBus) Close(ctx context.Context) error {
	b.mu.Lock()
	if b.isClosed {
		b.mu.Unlock()
		return nil
	}
	b.isClosed = true
	b.mu.Unlock()

	return b.Flush(ctx)
}

func (b *EventBus) dispatchAsyncKey(key string) {
	if err := b.flushKey(context.Background(), key); err != nil {
		b.reportError(err)
	}
}

func (b *EventBus) reportError(err error) {
	b.mu.Lock()
	handler := b.errorHandler
	b.mu.Unlock()

	if handler != nil {
		handler(err)
	}
}

func (b *EventBus) flushKey(ctx context.Context, key string) error {
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
		if _, err := host.TriggerEvent(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (b *EventBus) findMatchingHandlers(event string) []EventHandler {
	var result []EventHandler
	for prefix, list := range b.subscribers {
		if prefix == "" || prefix == "*" || strings.HasPrefix(event, prefix) {
			for _, sub := range list {
				result = append(result, sub.handler)
			}
		}
	}
	return result
}
