// The bounded queue the events stand in between the watcher's channel and the
// debounce, and the goroutine that empties that channel into it.

package vaultwatch

import (
	"context"
	"sync"

	"github.com/rjeczalik/notify"
)

// queue holds the events between the goroutine that reads them from the
// watcher and the goroutine that debounces them. It is bounded at `bound` events:
// when that many are waiting, the queued events are dropped and the overflow is
// recorded, and a caller of take is informed of it exactly.
//
// The watcher silently discards an event that finds its channel full, so the
// channel it writes into is drained by a goroutine that only reads.
type queue struct {
	maxBound int
	woke     chan struct{}
	mu       sync.Mutex
	events   []notify.EventInfo
	isOver   bool
	isDone   bool
}

func newQueue(limit int) *queue {
	return &queue{maxBound: limit, woke: make(chan struct{}, 1)}
}

// put takes one event. At the bound the events waiting are dropped and the
// overflow remembered: a rescan answers for all of them and for the event that
// did not fit.
func (q *queue) put(event notify.EventInfo) {
	q.mu.Lock()
	if len(q.events) >= q.maxBound {
		q.events, q.isOver = nil, true
	} else {
		q.events = append(q.events, event)
	}
	q.mu.Unlock()
	q.wake()
}

// stop records that the watcher's channel is closed and no more events are coming.
func (q *queue) stop() {
	q.mu.Lock()
	q.isDone = true
	q.mu.Unlock()
	q.wake()
}

// wake signals one edge: the woken goroutine empties the queue, so a wake that
// does not fit is a wake already pending.
func (q *queue) wake() {
	select {
	case q.woke <- struct{}{}:
	default:
	}
}

// take empties the queue and reports whether it overflowed since the last take.
func (q *queue) take() (events []notify.EventInfo, finished, completed bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	events, finished, completed = q.events, q.isOver, q.isDone
	q.events, q.isOver = nil, false
	return events, finished, completed
}

// forwardEvents moves events out of the watcher's channel as fast as they arrive.
func forwardEvents(ctx context.Context, raw <-chan notify.EventInfo, into *queue) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, open := <-raw:
			if !open {
				into.stop()
				return
			}
			into.put(event)
		}
	}
}
