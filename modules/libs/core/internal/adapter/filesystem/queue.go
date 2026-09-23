// The bounded queue the events stand in between the watcher's channel and the
// debounce, and the goroutine that empties that channel into it.

package filesystem

import (
	"context"
	"sync"

	"github.com/rjeczalik/notify"
)

// queue holds the events between the goroutine that reads them from the
// watcher and the goroutine that debounces them. It is bounded at `bound` events:
// at that many waiting, what is held is dropped and the overflow is
// remembered, and a caller of take is told of it exactly.
//
// The watcher discards an event that finds its channel full and says nothing,
// so the channel it writes into is read by a goroutine that only reads.
type queue struct {
	bound  int
	woke   chan struct{}
	mu     sync.Mutex
	events []notify.EventInfo
	isOver bool
	isDone bool
}

func newQueue(bound int) *queue {
	return &queue{bound: bound, woke: make(chan struct{}, 1)}
}

// put takes one event. At the bound the events waiting are dropped and the
// overflow remembered: a rescan answers for all of them and for the event that
// did not fit.
func (q *queue) put(event notify.EventInfo) {
	q.mu.Lock()
	if len(q.events) >= q.bound {
		q.events, q.isOver = nil, true
	} else {
		q.events = append(q.events, event)
	}
	q.mu.Unlock()
	q.wake()
}

// stop says the watcher's channel is closed and nothing more is coming.
func (q *queue) stop() {
	q.mu.Lock()
	q.isDone = true
	q.mu.Unlock()
	q.wake()
}

// wake carries one edge: whoever is woken empties the queue, so a wake that
// does not fit is a wake already about to happen.
func (q *queue) wake() {
	select {
	case q.woke <- struct{}{}:
	default:
	}
}

// take empties the queue and says whether it overflowed since the last take.
func (q *queue) take() (events []notify.EventInfo, over, done bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	events, over, done = q.events, q.isOver, q.isDone
	q.events, q.isOver = nil, false
	return events, over, done
}

// drain moves events out of the watcher's channel as fast as they arrive.
func drain(ctx context.Context, raw <-chan notify.EventInfo, into *queue) {
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
