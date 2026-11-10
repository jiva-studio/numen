// Package broadcast provides broadcaster and inflight synchronization primitives.
package broadcast

import (
	"slices"
	"sync"
)

// Broadcaster delivers messages of one type to its subscribers.
// A subscriber that does not keep up is not waited for, so one slow subscriber
// does not block the vault. A lagging subscriber receives the message produced
// by Fallback; Capacity sets how many messages may be pending before a
// subscriber counts as lagging.
type Broadcaster[T any] struct {
	Fallback   func(latest T) T
	IsLatest   bool
	IsOrdered  bool
	GetTopic   func(T) string
	IsRequired func(T) bool
	Capacity   int

	mu          sync.Mutex
	next        int
	subscribers map[int]*subscriber[T]
}

// subscriber is one subscriber of the broadcaster, and whether it missed a
// message.
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

func (a *Broadcaster[T]) queue(l *subscriber[T], message T) bool {
	wasEmpty := len(l.waiting) == 0

	fitted := false
	switch pos := a.findSupersededIndex(l.waiting, message); {
	case pos >= 0 && a.IsOrdered:
		l.waiting = append(slices.Delete(l.waiting, pos, pos+1), message)
		fitted = true
		if pos == 0 {
			l.frontSeq++
		}
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

func (a *Broadcaster[T]) deliver(l *subscriber[T]) {
	defer close(l.out)

	for {
		head, ok := a.waitForFront(l)
		if !ok || !a.sendHead(l, head) {
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

func (a *Broadcaster[T]) getFront(l *subscriber[T]) (front[T], bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(l.waiting) == 0 {
		return front[T]{}, false
	}
	return front[T]{message: l.waiting[0], seq: l.frontSeq}, true
}

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

func (a *Broadcaster[T]) sendHead(l *subscriber[T], head front[T]) bool {
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
