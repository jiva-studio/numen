package webui

import "sync"

// audience is everyone listening for one kind of message.
//
// A listener that is not keeping up is not waited for: one slow listener does
// not hold up the vault. What it is given when it falls behind is `fallback`'s
// answer, and how much it may be owed before it is behind is `room`.
type audience[T any] struct {
	// fallback makes the message for a listener that has not read the last one.
	fallback func(latest T) T
	// latest hands a listener what has just happened in place of what it has
	// not read yet about the same thing.
	latest bool
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
type listener[T any] struct {
	ch     chan T
	behind bool
}

// held is how many messages one listener may have waiting.
func (a *audience[T]) held() int { return max(a.room, 1) }

func (a *audience[T]) listen() (<-chan T, func()) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.listeners == nil {
		a.listeners = map[int]*listener[T]{}
	}
	id := a.next
	a.next++
	l := &listener[T]{ch: make(chan T, a.held())}
	a.listeners[id] = l

	return l.ch, func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		if l, open := a.listeners[id]; open {
			delete(a.listeners, id)
			close(l.ch)
		}
	}
}

func (a *audience[T]) tell(what T) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, l := range a.listeners {
		message := what
		if l.behind && a.fallback != nil {
			message = a.fallback(what)
		}
		l.behind = !a.queue(l, message)
	}
}

// queue puts a message where a listener will read it, and says whether there
// was anywhere for it to go.
//
// What the listener has not read is taken back out and looked at: the message
// takes the place of one waiting about the same thing, and otherwise joins the
// end of the line. A line with no room left keeps what it holds.
func (a *audience[T]) queue(l *listener[T], message T) bool {
	waiting := unread(l.ch)

	fitted := false
	switch at := a.replacing(waiting, message); {
	case at >= 0:
		waiting[at] = message
		fitted = true
	case len(waiting) < a.held():
		waiting = append(waiting, message)
		fitted = true
	}

	for _, held := range waiting {
		l.ch <- held
	}
	return fitted
}

// replacing is where in the line a message stands that this one says again, or
// -1 when this message replaces nothing.
func (a *audience[T]) replacing(waiting []T, message T) int {
	if !a.latest {
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

// unread is everything a listener was given and has not read, in the order it
// was given.
func unread[T any](ch chan T) []T {
	var waiting []T
	for {
		select {
		case held := <-ch:
			waiting = append(waiting, held)
		default:
			return waiting
		}
	}
}
