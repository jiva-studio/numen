// Package broadcast provides broadcaster and inflight synchronization primitives.
package broadcast

import "sync"

// Broadcaster delivers one kind of message to its subscribers.
//
// A subscriber that is not keeping up is not waited for: one slow subscriber does
// not hold up the vault. What it is given when it falls behind is Fallback's
// answer, and how much it may be owed before it is behind is Capacity.
type Broadcaster[T any] struct {
	// Fallback makes the message for a subscriber that has not read the last one.
	Fallback func(latest T) T
	// IsLatest hands a subscriber what has just happened in place of what it has
	// not read yet about the same thing.
	IsLatest bool
	// GetTopic says what a message is about. Two messages about one thing are that
	// thing said twice, and only the newer is worth reading; messages about
	// different things each wait their turn. Nil makes every message about the
	// same thing.
	GetTopic func(T) string
	// IsRequired says a message is one a subscriber has to be given. A message waiting
	// under it stays where it is, and what replaces it waits.
	IsRequired func(T) bool
	// Capacity is how many messages a subscriber may be owed before it is behind.
	Capacity int

	mu          sync.Mutex
	next        int
	subscribers map[int]*subscriber[T]
}

// subscriber is one of the broadcaster's subscribers, and whether it is owed a message it never
// received.
type subscriber[T any] struct {
	out       chan T
	isLagging bool

	waiting  []T
	frontSeq uint64
	notify   chan struct{}
	doneChan chan struct{}
}

// GetCapacity is how many messages one subscriber may have waiting.
func (a *Broadcaster[T]) GetCapacity() int { return max(a.Capacity, 1) }

func (a *Broadcaster[T]) Subscribe() (<-chan T, func()) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.subscribers == nil {
		a.subscribers = map[int]*subscriber[T]{}
	}
	id := a.next
	a.next++
	l := &subscriber[T]{
		out:      make(chan T),
		notify:   make(chan struct{}, 1),
		doneChan: make(chan struct{}),
	}
	a.subscribers[id] = l
	go a.deliver(l)

	return l.out, func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		if l, open := a.subscribers[id]; open {
			delete(a.subscribers, id)
			close(l.doneChan)
		}
	}
}

func (a *Broadcaster[T]) Publish(what T) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, l := range a.subscribers {
		message := what
		if l.isLagging && a.Fallback != nil {
			message = a.Fallback(what)
		}
		l.isLagging = !a.queue(l, message)
	}
}

// queue puts a message where a subscriber will read it, and says whether there
// was anywhere for it to go.
func (a *Broadcaster[T]) queue(l *subscriber[T], message T) bool {
	wasEmpty := len(l.waiting) == 0

	fitted := false
	switch pos := a.findSupersededIndex(l.waiting, message); {
	case pos >= 0:
		l.waiting[pos] = message
		fitted = true
		if pos == 0 {
			l.frontSeq++
		}
	case len(l.waiting) < a.GetCapacity():
		l.waiting = append(l.waiting, message)
		fitted = true
		if wasEmpty {
			l.frontSeq++
		}
	}

	if fitted {
		select {
		case l.notify <- struct{}{}:
		default:
		}
	}
	return fitted
}

// findSupersededIndex is where in the line a message stands that this one says again, or
// -1 when this message replaces nothing.
func (a *Broadcaster[T]) findSupersededIndex(waiting []T, message T) int {
	if !a.IsLatest {
		return -1
	}
	for item, target := range waiting {
		if a.IsRequired != nil && a.IsRequired(target) {
			continue
		}
		if a.GetTopic == nil || a.GetTopic(target) == a.GetTopic(message) {
			return item
		}
	}
	return -1
}

// deliver carries what queue puts on the slice to the channel listen handed
// out, one message at a time, in the order queue put them there.
func (a *Broadcaster[T]) deliver(l *subscriber[T]) {
	defer close(l.out)

	for {
		head, ok := a.waitForFront(l)
		if !ok || !a.offer(l, head) {
			return
		}
	}
}

// front is the message at the head of a subscriber's line, and the count of
// times that head had been replaced when it was read.
type front[T any] struct {
	message T
	seq     uint64
}

// getFront reads the head of the line, and reports whether the line holds one.
func (a *Broadcaster[T]) getFront(l *subscriber[T]) (front[T], bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(l.waiting) == 0 {
		return front[T]{}, false
	}
	return front[T]{message: l.waiting[0], seq: l.frontSeq}, true
}

// waitForFront blocks until the line holds a message, and reports false once
// the subscriber is done.
func (a *Broadcaster[T]) waitForFront(l *subscriber[T]) (front[T], bool) {
	for {
		if head, ok := a.getFront(l); ok {
			return head, true
		}
		select {
		case <-l.notify:
		case <-l.doneChan:
			return front[T]{}, false
		}
	}
}

// offer hands the head to out until it is taken, and takes it off the line when
// nothing has overtaken it.
func (a *Broadcaster[T]) offer(l *subscriber[T], head front[T]) bool {
	for {
		select {
		case l.out <- head.message:
			a.mu.Lock()
			if l.frontSeq == head.seq && len(l.waiting) > 0 {
				l.waiting = l.waiting[1:]
			}
			a.mu.Unlock()
			return true
		case <-l.notify:
			next, ok := a.getFront(l)
			if !ok {
				return true
			}
			head = next
		case <-l.doneChan:
			return false
		}
	}
}
