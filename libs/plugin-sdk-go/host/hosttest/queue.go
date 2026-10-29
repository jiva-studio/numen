package hosttest

import (
	"context"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// Queue is a queue that runs one piece of work at a time, as the host's does, and keeps the tasks
// reported to it in memory. Work a person started runs before work nobody asked for, and work of
// one kind runs in the order it arrived.
type Queue struct {
	mu         sync.Mutex
	isIndexed  bool
	requested  []chan struct{}
	background []chan struct{}
	watchers   []chan struct{}
	tasks      map[string]host.Task
	order      []string
}

var _ host.Queue = (*Queue)(nil)

// NewQueue is an empty queue with no tasks.
func NewQueue() *Queue { return &Queue{tasks: map[string]host.Task{}} }

func (q *Queue) EnqueueWork(ctx context.Context, work host.Work) (host.Turn, error) {
	q.mu.Lock()
	if !q.isIndexed {
		q.isIndexed = true
		q.mu.Unlock()
		return &queueTurn{queue: q}, nil
	}
	turn := make(chan struct{})
	if work.IsUserStarted {
		q.requested = append(q.requested, turn)
	} else {
		q.background = append(q.background, turn)
	}
	for _, watcher := range q.watchers {
		close(watcher)
	}
	q.watchers = nil
	q.mu.Unlock()

	if work.OnWaiting != nil {
		work.OnWaiting()
	}
	select {
	case <-turn:
		return &queueTurn{queue: q}, nil
	case <-ctx.Done():
		q.leave(turn)
		return nil, ctx.Err()
	}
}

func (q *Queue) ReportTask(_ context.Context, task host.Task) error {
	if task.ID == "" {
		return nil
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, isListed := q.tasks[task.ID]; !isListed {
		q.order = append(q.order, task.ID)
	}
	q.tasks[task.ID] = task
	return nil
}

func (q *Queue) FinishTask(_ context.Context, id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.tasks, id)
	q.order = slices.DeleteFunc(q.order, func(one string) bool { return one == id })
	return nil
}

// ListTasks returns the tasks shown now, in the order they were first reported.
func (q *Queue) ListTasks() []host.Task {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]host.Task, 0, len(q.order))
	for _, id := range q.order {
		out = append(out, q.tasks[id])
	}
	return out
}

// release hands the queue to the work that waited longest, a person's first.
func (q *Queue) release() {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, line := range []*[]chan struct{}{&q.requested, &q.background} {
		if len(*line) > 0 {
			next := (*line)[0]
			*line = (*line)[1:]
			close(next)
			return
		}
	}
	q.isIndexed = false
}

// leave takes a wait that ended out of the queue, and hands on a turn that arrived as it ended.
func (q *Queue) leave(turn chan struct{}) {
	q.mu.Lock()
	for _, line := range []*[]chan struct{}{&q.requested, &q.background} {
		if i := slices.Index(*line, turn); i >= 0 {
			*line = slices.Delete(*line, i, i+1)
			q.mu.Unlock()
			return
		}
	}
	q.mu.Unlock()
	<-turn
	q.release()
}

func (q *Queue) watchWaiting() <-chan struct{} {
	q.mu.Lock()
	defer q.mu.Unlock()
	watcher := make(chan struct{})
	if len(q.requested)+len(q.background) > 0 {
		close(watcher)
		return watcher
	}
	q.watchers = append(q.watchers, watcher)
	return watcher
}

type queueTurn struct {
	queue *Queue
	once  sync.Once
}

func (t *queueTurn) Release() { t.once.Do(t.queue.release) }

func (t *queueTurn) WatchWaiting() <-chan struct{} { return t.queue.watchWaiting() }
