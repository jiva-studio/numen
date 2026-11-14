package host

import (
	"context"
	"errors"
	"sync"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// ErrTurnRefused is what EnqueueWork returns when the host ended the wait without a turn.
var ErrTurnRefused = errors.New("host: the queue gave no turn")

// Queue is the core's shared queue of heavy work and its list of tasks. It is
// the same queue that the core's own heavy work uses, and a reported task is
// listed with the core's tasks. The host removes the plugin's tasks and ends
// its turns when the plugin stops. Plugins use it through QueueClient; tests
// use hosttest.Queue.
type Queue interface {
	// EnqueueWork waits for the work's turn in the queue and returns it. Work
	// that the user started runs before other work. A context that ends while
	// waiting acquires no turn, and one that ends during the turn ends the turn.
	EnqueueWork(ctx context.Context, work Work) (Turn, error)
	// ReportTask shows the task, or replaces the one shown under its id.
	ReportTask(ctx context.Context, task Task) error
	// FinishTask takes the task with the id out of the list.
	FinishTask(ctx context.Context, id string) error
}

// Work is one piece of heavy work as it waits in the queue. OnWaiting, when set, is called where
// other work holds the queue.
type Work struct {
	IsUserStarted bool
	OnWaiting     func()
}

// Turn is the work's hold on the queue.
type Turn interface {
	// Release ends the turn and lets the next work in the queue run, and returns the failure the turn
	// ended on before it was released. A turn released twice is released once.
	Release() error
	// WatchWaiting returns a channel closed once other work waits for the queue.
	WatchWaiting() <-chan struct{}
}

// Task is one piece of the plugin's work as displayed to the user. ID is
// chosen by the plugin. CompletedCount and TotalCount are counted in Unit, and
// both zero means there is nothing to count. A task with a FailureReason stays
// listed until it is finished.
type Task struct {
	ID             string
	Title          string
	Detail         string
	CompletedCount int64
	TotalCount     int64
	Unit           Unit
	FailureReason  string
	IsUserStarted  bool
}

// Unit is what a task counts.
type Unit int

const (
	UnitThings Unit = iota
	UnitBytes
	UnitSeconds
)

// QueueClient works the queue through HostQueueService.
type QueueClient struct {
	rpc numenv1connect.HostQueueServiceClient
}

var _ Queue = (*QueueClient)(nil)

// NewQueueClientFromEnv connects to the host the process was started for. It
// returns hostcall.ErrNoHost when no host is configured.
func NewQueueClientFromEnv() (*QueueClient, error) {
	addr, err := hostcall.FromEnv()
	if err != nil {
		return nil, err
	}
	return NewQueueClient(addr), nil
}

func NewQueueClient(endpoint hostcall.Endpoint) *QueueClient {
	return NewQueueClientOver(hostcall.HTTPClient(), endpoint)
}

func NewQueueClientOver(httpClient connect.HTTPClient, endpoint hostcall.Endpoint) *QueueClient {
	return &QueueClient{rpc: numenv1connect.NewHostQueueServiceClient(httpClient, endpoint.BaseURL, endpoint.Options()...)}
}

// EnqueueWork holds the stream of the call open for as long as the turn lasts.
func (c *QueueClient) EnqueueWork(ctx context.Context, work Work) (Turn, error) {
	ctx, cancel := context.WithCancel(ctx)
	stream, err := c.rpc.EnqueueWork(ctx, connect.NewRequest(&v1.EnqueueWorkRequest{IsUserStarted: work.IsUserStarted}))
	if err != nil {
		cancel()
		return nil, err
	}
	for stream.Receive() {
		switch stream.Msg().GetState() {
		case v1.WorkState_WORK_STATE_WAITING:
			if work.OnWaiting != nil {
				work.OnWaiting()
			}
		case v1.WorkState_WORK_STATE_RUNNING, v1.WorkState_WORK_STATE_CONTENDED:
			turn := &clientTurn{cancel: cancel, waiting: make(chan struct{}), ended: make(chan struct{})}
			if stream.Msg().GetState() == v1.WorkState_WORK_STATE_CONTENDED {
				turn.closeWaitingChannel()
			}
			go turn.follow(ctx, stream)
			return turn, nil
		case v1.WorkState_WORK_STATE_UNSPECIFIED:
		}
	}
	err = stream.Err()
	cancel()
	if closeErr := stream.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = ErrTurnRefused
	}
	return nil, err
}

func (c *QueueClient) ReportTask(ctx context.Context, task Task) error {
	_, err := c.rpc.ReportTask(ctx, connect.NewRequest(&v1.ReportTaskRequest{Task: &v1.Task{
		Id:            task.ID,
		Title:         task.Title,
		About:         task.Detail,
		Done:          task.CompletedCount,
		Total:         task.TotalCount,
		Unit:          writeUnit(task.Unit),
		Error:         task.FailureReason,
		IsUserStarted: task.IsUserStarted,
	}}))
	return err
}

func (c *QueueClient) FinishTask(ctx context.Context, id string) error {
	_, err := c.rpc.FinishTask(ctx, connect.NewRequest(&v1.FinishTaskRequest{Id: id}))
	return err
}

func writeUnit(unit Unit) v1.Unit {
	switch unit {
	case UnitBytes:
		return v1.Unit_UNIT_BYTES
	case UnitSeconds:
		return v1.Unit_UNIT_SECONDS
	case UnitThings:
		return v1.Unit_UNIT_THINGS
	}
	return v1.Unit_UNIT_THINGS
}

// clientTurn is a turn held by an open stream. The goroutine reading the stream ends when the
// stream does, which Release or the end of the context brings about.
type clientTurn struct {
	cancel      context.CancelFunc
	waiting     chan struct{}
	waitingOnce sync.Once
	ended       chan struct{}
	releaseOnce sync.Once
	err         error
}

func (t *clientTurn) Release() error {
	t.releaseOnce.Do(func() {
		t.cancel()
		<-t.ended
	})
	return t.err
}

func (t *clientTurn) WatchWaiting() <-chan struct{} { return t.waiting }

func (t *clientTurn) closeWaitingChannel() { t.waitingOnce.Do(func() { close(t.waiting) }) }

func (t *clientTurn) follow(ctx context.Context, stream *connect.ServerStreamForClient[v1.EnqueueWorkResponse]) {
	defer close(t.ended)
	for stream.Receive() {
		if stream.Msg().GetState() == v1.WorkState_WORK_STATE_CONTENDED {
			t.closeWaitingChannel()
		}
	}
	err := errors.Join(stream.Err(), closeStream(stream))
	if ctx.Err() == nil {
		t.err = err
	}
}
