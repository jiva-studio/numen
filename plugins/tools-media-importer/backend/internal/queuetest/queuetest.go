// Package queuetest is the host's queue for tests that wait for work the plugin runs in the
// background.
package queuetest

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

// Deadline is how long a test waits for a task to end before it fails.
const Deadline = 30 * time.Second

// Log is the host's queue that also counts the tasks that ended, finished or failed.
type Log struct {
	*hosttest.Queue

	mu      sync.Mutex
	ended   map[string]int
	awaited map[string]int
	changed chan struct{}
}

// NewLog returns an empty queue with no task ended.
func NewLog() *Log {
	return &Log{
		Queue: hosttest.NewQueue(), mu: sync.Mutex{},
		ended: map[string]int{}, awaited: map[string]int{}, changed: make(chan struct{}),
	}
}

// ReportTask shows the task, and counts it as ended when it carries a failure.
func (l *Log) ReportTask(ctx context.Context, task host.Task) error {
	err := l.Queue.ReportTask(ctx, task)
	if task.FailureReason != "" {
		l.addEnd(task.ID)
	}
	return err
}

// FinishTask takes the task out of the list and counts it as ended.
func (l *Log) FinishTask(ctx context.Context, id string) error {
	err := l.Queue.FinishTask(ctx, id)
	l.addEnd(id)
	return err
}

func (l *Log) addEnd(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ended[id]++
	close(l.changed)
	l.changed = make(chan struct{})
}

// WaitForEnd waits for the next end of the task with the id that no earlier call waited for, and
// fails the test when it does not come within Deadline.
func (l *Log) WaitForEnd(tb testing.TB, id string) {
	tb.Helper()
	deadline := time.NewTimer(Deadline)
	defer deadline.Stop()
	for {
		l.mu.Lock()
		if l.ended[id] > l.awaited[id] {
			l.awaited[id]++
			l.mu.Unlock()
			return
		}
		changed := l.changed
		l.mu.Unlock()
		select {
		case <-changed:
		case <-deadline.C:
			tb.Fatalf("the task %s did not end within %v; the tasks shown are %+v", id, Deadline, l.ListTasks())
		}
	}
}
