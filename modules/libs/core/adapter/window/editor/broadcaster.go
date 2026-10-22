package editor

import "sync"

// broadcaster delivers one kind of message to its subscribers.
//
// A subscriber that is not keeping up is not waited for: one slow subscriber does
// not hold up the vault. What it is given when it falls behind is `fallback`'s
// answer, and how much it may be owed before it is behind is `capacity`.
type broadcaster[T any] struct {
	// fallback makes the message for a subscriber that has not read the last one.
	fallback func(latest T) T
	// isLatest hands a subscriber what has just happened in place of what it has
	// not read yet about the same thing.
	isLatest bool
	// getTopic says what a message is about. Two messages about one thing are that
	// thing said twice, and only the newer is worth reading; messages about
	// different things each wait their turn. Nil makes every message about the
	// same thing.
	getTopic func(T) string
	// isRequired says a message is one a subscriber has to be given. A message waiting
	// under it stays where it is, and what replaces it waits.
	isRequired func(T) bool
	// capacity is how many messages a subscriber may be owed before it is behind.
	capacity int

	mu          sync.Mutex
	next        int
	subscribers map[int]*subscriber[T]
}

// subscriber is one of the broadcaster's subscribers, and whether it is owed a message it never
// received.
//
// What it has not been handed is a slice, read and written only under the
// broadcaster's lock. out is the channel subscribe returned, and is written only by
// deliver: a subscriber's own goroutine, carrying the slice to out one message
// at a time. frontSeq counts how many times the first entry has been
// replaced, so deliver can tell whether the one it is offering is still the
// one at the front or has since been overtaken.
type subscriber[T any] struct {
	out       chan T
	isLagging bool

	waiting  []T
	frontSeq uint64
	notify   chan struct{}
	done     chan struct{}
}

// getCapacity is how many messages one subscriber may have waiting.
func (a *broadcaster[T]) getCapacity() int { return max(a.capacity, 1) }

func (a *broadcaster[T]) subscribe() (<-chan T, func()) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.subscribers == nil {
		a.subscribers = map[int]*subscriber[T]{}
	}
	id := a.next
	a.next++
	l := &subscriber[T]{
		out:    make(chan T),
		notify: make(chan struct{}, 1),
		done:   make(chan struct{}),
	}
	a.subscribers[id] = l
	go a.deliver(l)

	return l.out, func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		if l, open := a.subscribers[id]; open {
			delete(a.subscribers, id)
			close(l.done)
		}
	}
}

func (a *broadcaster[T]) publish(what T) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, l := range a.subscribers {
		message := what
		if l.isLagging && a.fallback != nil {
			message = a.fallback(what)
		}
		l.isLagging = !a.queue(l, message)
	}
}

// queue puts a message where a subscriber will read it, and says whether there
// was anywhere for it to go.
//
// What the subscriber has not read is taken out of the slice and looked at: the
// message takes the place of one waiting about the same thing, and otherwise
// joins the end of the line. A line with no room left keeps what it holds.
func (a *broadcaster[T]) queue(l *subscriber[T], message T) bool {
	wasEmpty := len(l.waiting) == 0

	fitted := false
	switch pos := a.findSupersededIndex(l.waiting, message); {
	case pos >= 0:
		l.waiting[pos] = message
		fitted = true
		if pos == 0 {
			l.frontSeq++
		}
	case len(l.waiting) < a.getCapacity():
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
func (a *broadcaster[T]) findSupersededIndex(waiting []T, message T) int {
	if !a.isLatest {
		return -1
	}
	for item, target := range waiting {
		if a.isRequired != nil && a.isRequired(target) {
			continue
		}
		if a.getTopic == nil || a.getTopic(target) == a.getTopic(message) {
			return item
		}
	}
	return -1
}

// deliver carries what queue puts on the slice to the channel listen handed
// out, one message at a time, in the order queue put them there.
//
// It only takes a message out of the slice once that exact message has been
// handed to out: a replacement that lands on the front while one is offered
// there is picked up in its place, and what was offered before it is dropped
// from the slice only once nothing has overtaken it.
func (a *broadcaster[T]) deliver(l *subscriber[T]) {
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
func (a *broadcaster[T]) getFront(l *subscriber[T]) (front[T], bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(l.waiting) == 0 {
		return front[T]{}, false
	}
	return front[T]{message: l.waiting[0], seq: l.frontSeq}, true
}

// waitForFront blocks until the line holds a message, and reports false once
// the subscriber is done.
func (a *broadcaster[T]) waitForFront(l *subscriber[T]) (front[T], bool) {
	for {
		if head, ok := a.getFront(l); ok {
			return head, true
		}
		select {
		case <-l.notify:
		case <-l.done:
			return front[T]{}, false
		}
	}
}

// offer hands the head to out until it is taken, and takes it off the line when
// nothing has overtaken it. A replacement landing on the head is offered in its
// place. It reports false once the subscriber is done.
func (a *broadcaster[T]) offer(l *subscriber[T], head front[T]) bool {
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
		case <-l.done:
			return false
		}
	}
}
