package modelcall

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
)

const (
	// InteractiveWait is how long a call made for a person's action waits for a slot of its connection.
	InteractiveWait = 60 * time.Second
	// BackgroundWait is how long a call made for background work waits for a slot of its connection.
	BackgroundWait = 10 * time.Minute
)

// errQueueTimeout is a call that waited for a slot of its connection as long as its priority allows.
var errQueueTimeout = errors.New("the connection stayed busy")

// queues holds the calls of each connection that run and wait, shared by every plugin that uses it.
type queues struct {
	mu    sync.Mutex
	wait  Wait
	lines map[string]*line
}

func newQueues(wait Wait) *queues {
	return &queues{wait: wait, lines: map[string]*line{}}
}

// line is the calls of one connection: how many run, and the two lists of those that wait.
type line struct {
	capacity    int
	running     int
	interactive []*waiter
	background  []*waiter
}

type waiter struct {
	granted chan struct{}
	isReady bool
}

func (q *queues) acquire(ctx context.Context, connectionID string, capacity int, priority modelcall.Priority) (func(), error) {
	if capacity < 1 {
		return func() {}, nil
	}
	one := &waiter{granted: make(chan struct{})}
	q.mu.Lock()
	queued := q.lines[connectionID]
	if queued == nil {
		queued = &line{}
		q.lines[connectionID] = queued
	}
	queued.capacity = capacity
	if priority == modelcall.PriorityInteractive {
		queued.interactive = append(queued.interactive, one)
	} else {
		queued.background = append(queued.background, one)
	}
	q.grant(connectionID, queued)
	q.mu.Unlock()
	release := func() { q.release(connectionID) }
	select {
	case <-one.granted:
		return release, nil
	default:
	}
	timedOut, stop := q.startTimer(ctx, priority)
	defer stop()
	select {
	case <-one.granted:
		return release, nil
	case <-timedOut:
		return q.leave(connectionID, one, errQueueTimeout, true)
	case <-ctx.Done():
		return q.leave(connectionID, one, ctx.Err(), false)
	}
}

func (q *queues) startTimer(ctx context.Context, priority modelcall.Priority) (<-chan struct{}, func()) {
	timerCtx, cancel := context.WithCancel(ctx)
	timedOut := make(chan struct{})
	timerDone := make(chan struct{})
	go func() {
		defer close(timerDone)
		if q.wait(timerCtx, getQueueWait(priority)) == nil {
			close(timedOut)
		}
	}()
	return timedOut, func() {
		cancel()
		<-timerDone
	}
}

func getQueueWait(priority modelcall.Priority) time.Duration {
	if priority == modelcall.PriorityInteractive {
		return InteractiveWait
	}
	return BackgroundWait
}

func (q *queues) leave(connectionID string, one *waiter, reason error, keepGranted bool) (func(), error) {
	q.mu.Lock()
	if one.isReady {
		q.mu.Unlock()
		if keepGranted {
			return func() { q.release(connectionID) }, nil
		}
		q.release(connectionID)
		return nil, reason
	}
	defer q.mu.Unlock()
	if queued := q.lines[connectionID]; queued != nil {
		queued.interactive = removeWaiter(queued.interactive, one)
		queued.background = removeWaiter(queued.background, one)
		q.drop(connectionID, queued)
	}
	return nil, reason
}

func removeWaiter(list []*waiter, one *waiter) []*waiter {
	for i, other := range list {
		if other == one {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func (q *queues) release(connectionID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	queued := q.lines[connectionID]
	if queued == nil {
		return
	}
	queued.running--
	q.grant(connectionID, queued)
}

func (q *queues) grant(connectionID string, queued *line) {
	// The caller holds mu. The head of the interactive list goes first.
	for queued.running < queued.capacity {
		var next *waiter
		switch {
		case len(queued.interactive) > 0:
			next, queued.interactive = queued.interactive[0], queued.interactive[1:]
		case len(queued.background) > 0:
			next, queued.background = queued.background[0], queued.background[1:]
		default:
			q.drop(connectionID, queued)
			return
		}
		queued.running++
		next.isReady = true
		close(next.granted)
	}
}

func (q *queues) drop(connectionID string, queued *line) {
	// The caller holds mu.
	if queued.running == 0 && len(queued.interactive) == 0 && len(queued.background) == 0 {
		delete(q.lines, connectionID)
	}
}

// Wait holds the caller for d, or returns the error of the context as soon as the context ends.
type Wait func(ctx context.Context, d time.Duration) error
