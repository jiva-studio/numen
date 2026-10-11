package editor

import "sync"

// audience is everyone listening for one kind of message.
//
// A listener that is not keeping up is not waited for: one slow listener does
// not hold up the vault. What it is given when it falls behind is `fallback`'s
// answer, and how much it may be owed before it is behind is `room`.
type audience[T any] struct {
	// fallback makes the message for a listener that has not read the last one.
	fallback func(latest T) T
	// isLatest hands a listener what has just happened in place of what it has
	// not read yet about the same thing.
	isLatest bool
	// about says what a message is about. Two messages about one thing are that
	// thing said twice, and only the newer is worth reading; messages about
	// different things each wait their turn. Nil makes every message about the
	// same thing.
	about func(T) string
	// keep says a message is one a listener has to be given. A message waiting
	// under it stays where it is, and what replaces it waits.
	keep func(T) bool
	// room is how many messages a listener may be owed before it is behind.
	room int

	mu        sync.Mutex
	next      int
	listeners map[int]*listener[T]
}

// listener is one of the audience, and whether it is owed a message it never
// received.
//
// What it has not been handed is a slice, read and written only under the
// audience's lock. out is the channel listen gave out, and is written only by
// deliver: a listener's own goroutine, carrying the slice to out one message
// at a time. frontSeq counts how many times the first entry has been
// replaced, so deliver can tell whether the one it is offering is still the
// one at the front or has since been overtaken.
type listener[T any] struct {
	out      chan T
	isBehind bool

	waiting  []T
	frontSeq uint64
	notify   chan struct{}
	done     chan struct{}
}

// getRoom is how many messages one listener may have waiting.
func (a *audience[T]) getRoom() int { return max(a.room, 1) }

func (a *audience[T]) listen() (<-chan T, func()) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.listeners == nil {
		a.listeners = map[int]*listener[T]{}
	}
	id := a.next
	a.next++
	l := &listener[T]{
		out:    make(chan T),
		notify: make(chan struct{}, 1),
		done:   make(chan struct{}),
	}
	a.listeners[id] = l
	go a.deliver(l)

	return l.out, func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		if l, open := a.listeners[id]; open {
			delete(a.listeners, id)
			close(l.done)
		}
	}
}

func (a *audience[T]) tell(what T) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, l := range a.listeners {
		message := what
		if l.isBehind && a.fallback != nil {
			message = a.fallback(what)
		}
		l.isBehind = !a.queue(l, message)
	}
}

// queue puts a message where a listener will read it, and says whether there
// was anywhere for it to go.
//
// What the listener has not read is taken out of the slice and looked at: the
// message takes the place of one waiting about the same thing, and otherwise
// joins the end of the line. A line with no room left keeps what it holds.
func (a *audience[T]) queue(l *listener[T], message T) bool {
	wasEmpty := len(l.waiting) == 0

	fitted := false
	switch at := a.findReplaced(l.waiting, message); {
	case at >= 0:
		l.waiting[at] = message
		fitted = true
		if at == 0 {
			l.frontSeq++
		}
	case len(l.waiting) < a.getRoom():
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

// findReplaced is where in the line a message stands that this one says again, or
// -1 when this message replaces nothing.
func (a *audience[T]) findReplaced(waiting []T, message T) int {
	if !a.isLatest {
		return -1
	}
	for at, held := range waiting {
		if a.keep != nil && a.keep(held) {
			continue
		}
		if a.about == nil || a.about(held) == a.about(message) {
			return at
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
func (a *audience[T]) deliver(l *listener[T]) {
	defer close(l.out)

outer:
	for {
		a.mu.Lock()
		for len(l.waiting) == 0 {
			a.mu.Unlock()
			select {
			case <-l.notify:
			case <-l.done:
				return
			}
			a.mu.Lock()
		}
		head, seq := l.waiting[0], l.frontSeq
		a.mu.Unlock()

		for {
			select {
			case l.out <- head:
				a.mu.Lock()
				if l.frontSeq == seq && len(l.waiting) > 0 {
					l.waiting = l.waiting[1:]
				}
				a.mu.Unlock()
				continue outer
			case <-l.notify:
				a.mu.Lock()
				if len(l.waiting) == 0 {
					a.mu.Unlock()
					continue outer
				}
				head, seq = l.waiting[0], l.frontSeq
				a.mu.Unlock()
			case <-l.done:
				return
			}
		}
	}
}
