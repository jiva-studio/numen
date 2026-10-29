package host_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// queueing is a host whose queue makes every caller wait once, grants it the turn, and notifies it
// that other work waits behind it. It records what it sent.
type queueing struct {
	numenv1connect.UnimplementedHostQueueServiceHandler
	mu         sync.Mutex
	authHeader string
	reported   []*v1.Task
	finished   []string
	ended      chan struct{}
	contend    chan struct{}
}

func (q *queueing) EnqueueWork(
	ctx context.Context, req *connect.Request[v1.EnqueueWorkRequest], stream *connect.ServerStream[v1.EnqueueWorkResponse],
) error {
	q.mu.Lock()
	q.authHeader = req.Header().Get("Authorization")
	q.mu.Unlock()
	defer close(q.ended)
	for _, state := range []v1.WorkState{v1.WorkState_WORK_STATE_WAITING, v1.WorkState_WORK_STATE_RUNNING} {
		if err := stream.Send(&v1.EnqueueWorkResponse{State: state}); err != nil {
			return err
		}
	}
	select {
	case <-q.contend:
		if err := stream.Send(&v1.EnqueueWorkResponse{State: v1.WorkState_WORK_STATE_CONTENDED}); err != nil {
			return err
		}
	case <-ctx.Done():
		return nil
	}
	<-ctx.Done()
	return nil
}

func (q *queueing) ReportTask(_ context.Context, req *connect.Request[v1.ReportTaskRequest]) (*connect.Response[v1.ReportTaskResponse], error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.reported = append(q.reported, req.Msg.GetTask())
	return connect.NewResponse(&v1.ReportTaskResponse{}), nil
}

func (q *queueing) FinishTask(_ context.Context, req *connect.Request[v1.FinishTaskRequest]) (*connect.Response[v1.FinishTaskResponse], error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.finished = append(q.finished, req.Msg.GetId())
	return connect.NewResponse(&v1.FinishTaskResponse{}), nil
}

func newQueueClient(t *testing.T) (*host.QueueClient, *queueing) {
	t.Helper()
	q := &queueing{ended: make(chan struct{}), contend: make(chan struct{})}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostQueueServiceHandler(q))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return host.NewQueueClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"}), q
}

func TestQueueClientKeepsTheTurnUntilRelease(t *testing.T) {
	client, q := newQueueClient(t)

	var waited int
	turn, err := client.EnqueueWork(t.Context(), host.Work{IsUserStarted: true, OnWaiting: func() { waited++ }})
	if err != nil {
		t.Fatal(err)
	}
	if waited != 1 {
		t.Errorf("OnWaiting ran %d times, want 1", waited)
	}
	if q.authHeader != "Bearer tok" {
		t.Errorf("the host was called with %q", q.authHeader)
	}
	select {
	case <-turn.WatchWaiting():
		t.Fatal("signalled work behind the turn before the host reported any")
	default:
	}

	close(q.contend)
	select {
	case <-turn.WatchWaiting():
	case <-time.After(time.Second):
		t.Fatal("no signal for the work queued behind the turn")
	}
	select {
	case <-q.ended:
		t.Fatal("the turn ended before it was released")
	default:
	}

	turn.Release()
	turn.Release()
	select {
	case <-q.ended:
	case <-time.After(time.Second):
		t.Fatal("the host still holds the turn after its release")
	}
}

func TestQueueClientReportsAndFinishesTasks(t *testing.T) {
	client, q := newQueueClient(t)

	task := host.Task{ID: "a.mp3", Title: "Transcribing", About: "a.mp3", Done: 4, Total: 60, Unit: host.UnitSeconds, IsUserStarted: true}
	if err := client.ReportTask(t.Context(), task); err != nil {
		t.Fatal(err)
	}
	if err := client.FinishTask(t.Context(), "a.mp3"); err != nil {
		t.Fatal(err)
	}
	if len(q.reported) != 1 {
		t.Fatalf("the host received %d tasks, want 1", len(q.reported))
	}
	got := q.reported[0]
	if got.GetId() != "a.mp3" || got.GetTitle() != "Transcribing" || got.GetAbout() != "a.mp3" || got.GetDone() != 4 ||
		got.GetTotal() != 60 || got.GetUnit() != v1.Unit_UNIT_SECONDS || !got.GetIsUserStarted() {
		t.Errorf("the host received %v", got)
	}
	if len(q.finished) != 1 || q.finished[0] != "a.mp3" {
		t.Errorf("the host finished %v", q.finished)
	}
}
