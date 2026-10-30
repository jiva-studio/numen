package hostendpoint

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
)

// newQueueClient serves the queue over the work given and returns a client
// calling it with the token.
func newQueueClient(t *testing.T, work *run.PluginWork, token string) numenv1connect.HostQueueServiceClient {
	t.Helper()
	return newQueueClientOver(t, newCallers(nil), work, token)
}

// newQueueClientOver is that client over the host the callers are.
func newQueueClientOver(t *testing.T, fakeCallers *callers, work *run.PluginWork, token string) numenv1connect.HostQueueServiceClient {
	t.Helper()
	fakeCallers.work = work
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostQueueServiceHandler(&queueEndpoint{endpoint: &Endpoint{callers: fakeCallers}}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewHostQueueServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.AuthInterceptor(token)))
}

// openWork enqueues work and returns the stream once it says the state given.
func openWork(
	t *testing.T, ctx context.Context, client numenv1connect.HostQueueServiceClient, isUserStarted bool, want v1.WorkState,
) *connect.ServerStreamForClient[v1.EnqueueWorkResponse] {
	t.Helper()
	stream, err := client.EnqueueWork(ctx, connect.NewRequest(&v1.EnqueueWorkRequest{IsUserStarted: isUserStarted}))
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, stream)
	readState(t, stream, want)
	return stream
}

func readState(t *testing.T, stream *connect.ServerStreamForClient[v1.EnqueueWorkResponse], want v1.WorkState) {
	t.Helper()
	if !stream.Receive() {
		t.Fatalf("the stream ended before %s: %v", want, stream.Err())
	}
	if got := stream.Msg().GetState(); got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestEnqueueWorkRunsUserWorkFirst(t *testing.T) {
	lock := &run.Lock{}
	work := run.NewPluginWork(lock, run.NewTasks(), nil)
	release, err := lock.Acquire(t.Context(), true, nil)
	if err != nil {
		t.Fatal(err)
	}

	background := openWork(t, t.Context(), newQueueClient(t, work, "token-a"), false, v1.WorkState_WORK_STATE_WAITING)
	requestedCtx, closeRequested := context.WithCancel(t.Context())
	requested := openWork(t, requestedCtx, newQueueClient(t, work, "token-b"), true, v1.WorkState_WORK_STATE_WAITING)

	release()
	readState(t, requested, v1.WorkState_WORK_STATE_RUNNING)
	readState(t, requested, v1.WorkState_WORK_STATE_CONTENDED)

	closeRequested()
	readState(t, background, v1.WorkState_WORK_STATE_RUNNING)
}

func TestEnqueueWorkEndsStreamOfClearedPlugin(t *testing.T) {
	work := run.NewPluginWork(&run.Lock{}, run.NewTasks(), nil)
	stream := openWork(t, t.Context(), newQueueClient(t, work, "token-a"), false, v1.WorkState_WORK_STATE_RUNNING)

	work.ClearPlugin("md.numen.a")
	ended := make(chan bool, 1)
	go func() { ended <- stream.Receive() }()
	select {
	case isMore := <-ended:
		if isMore {
			t.Fatalf("the stream went on with %s", stream.Msg().GetState())
		}
	case <-time.After(time.Second):
		t.Fatal("the stream of a stopped plugin did not end")
	}
	openWork(t, t.Context(), newQueueClient(t, work, "token-c"), false, v1.WorkState_WORK_STATE_RUNNING)
}

func TestReportTaskUsesCallerPluginID(t *testing.T) {
	tasks := run.NewTasks()
	client := newQueueClient(t, run.NewPluginWork(&run.Lock{}, tasks, nil), "token-a")

	_, err := client.ReportTask(t.Context(), connect.NewRequest(&v1.ReportTaskRequest{Task: &v1.Task{
		Id: "scan.pdf", Title: "Recognising", About: "scan.pdf", Done: 2, Total: 9, Unit: v1.Unit_UNIT_BYTES,
		IsUserStarted: true, PluginId: "md.numen.other",
	}}))
	if err != nil {
		t.Fatal(err)
	}
	want := run.Task{
		ID: "plugin:md.numen.a:scan.pdf", Title: "Recognising", About: "scan.pdf", Count: 2, Total: 9,
		Unit: run.Bytes, IsUserStarted: true, PluginID: "md.numen.a",
	}
	if got := tasks.List(); len(got) != 1 || got[0] != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}

	if _, err := client.FinishTask(t.Context(), connect.NewRequest(&v1.FinishTaskRequest{Id: "scan.pdf"})); err != nil {
		t.Fatal(err)
	}
	if got := tasks.List(); len(got) != 0 {
		t.Fatalf("got %+v after finishing, want none", got)
	}
}

func TestQueueRejectsCallerWithoutToken(t *testing.T) {
	client := newQueueClient(t, run.NewPluginWork(&run.Lock{}, run.NewTasks(), nil), "token-unknown")
	_, err := client.ReportTask(t.Context(), connect.NewRequest(&v1.ReportTaskRequest{Task: &v1.Task{Id: "a"}}))
	if code := readCode(t, err); code != connect.CodeUnauthenticated {
		t.Fatalf("got %s, want unauthenticated", code)
	}
}

func TestEnqueueWorkAcquiresPluginForStream(t *testing.T) {
	work := run.NewPluginWork(&run.Lock{}, run.NewTasks(), nil)
	fakeCallers := newCallers(nil)
	fakeCallers.releases = make(chan string, 1)
	ctx, closeStream := context.WithCancel(t.Context())
	openWork(t, ctx, newQueueClientOver(t, fakeCallers, work, "token-a"), true, v1.WorkState_WORK_STATE_RUNNING)

	if got := fakeCallers.getAcquired("md.numen.a"); got != 1 {
		t.Fatalf("the plugin is held %d times during its turn, want 1", got)
	}
	closeStream()
	select {
	case <-fakeCallers.releases:
	case <-time.After(5 * time.Second):
		t.Fatal("the plugin was not released after its turn ended")
	}
	if got := fakeCallers.getAcquired("md.numen.a"); got != 0 {
		t.Fatalf("the plugin is still held %d times after its turn ended", got)
	}
}
