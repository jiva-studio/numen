package webui

import "sync"

// audience is everyone listening for one kind of message.
//
// A listener that is not keeping up is not waited for: one slow listener does
// not hold up the vault. What it is given when it falls behind is `behind`'s
// answer, which is the one thing the two audiences here differ in.
type audience[T any] struct {
	// behind makes the message for a listener that has not read the last one.
	behind func(latest T) T
	// latest hands a listener what has just happened in place of what it has
	// not read yet.
	latest bool
	// room is how many messages a listener may be owed before it is behind.
	room int

	mu   sync.Mutex
	next int
	to   map[int]*line[T]
}

// line is one listener, and whether it is owed a message it never received.
type line[T any] struct {
	ch     chan T
	behind bool
}

func (a *audience[T]) listen() (<-chan T, func()) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.to == nil {
		a.to = map[int]*line[T]{}
	}
	id := a.next
	a.next++
	l := &line[T]{ch: make(chan T, max(a.room, 1))}
	a.to[id] = l

	return l.ch, func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		if l, open := a.to[id]; open {
			delete(a.to, id)
			close(l.ch)
		}
	}
}

func (a *audience[T]) tell(what T) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, l := range a.to {
		message := what
		if l.behind && a.behind != nil {
			message = a.behind(what)
		}
		select {
		case l.ch <- message:
			l.behind = false
			continue
		default:
		}

		if !a.latest {
			l.behind = true
			continue
		}
		select {
		case <-l.ch:
		default:
		}
		select {
		case l.ch <- message:
			l.behind = false
		default:
			l.behind = true
		}
	}
}
