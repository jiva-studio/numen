package wire

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// openTaskStream is the stream a window would hold, over a list a test drives.
func openTaskStream(t *testing.T, tasks *run.Tasks) *connect.ServerStreamForClient[v1.WatchTasksResponse] {
	t.Helper()

	route, handler := numenv1connect.NewWindowServiceHandler(&Window{Named: Editor, Tasking: tasks})
	mux := http.NewServeMux()
	mux.Handle(route, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	ctx, stop := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(stop)

	client := numenv1connect.NewWindowServiceClient(server.Client(), server.URL)
	stream, err := client.WatchTasks(ctx, connect.NewRequest(&v1.WatchTasksRequest{Window: Editor}))
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, stream)
	return stream
}

// receiveTasks returns the next task list from the stream.
func receiveTasks(t *testing.T, stream *connect.ServerStreamForClient[v1.WatchTasksResponse]) []*v1.Task {
	t.Helper()
	if !stream.Receive() {
		t.Fatalf("the stream ended: %v", stream.Err())
	}
	return stream.Msg().GetTasks()
}

// A client receives each list when the following write arrives, so a list with
// no write after it is never delivered. The stream repeats an unchanged
// snapshot, which clears finished work from the running display.
func TestTasksStream_RepeatsUnchangedSnapshot(t *testing.T) {
	tasks := run.NewTasks()
	stream := openTaskStream(t, tasks)

	if first := receiveTasks(t, stream); len(first) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(first))
	}

	tasks.Set(run.Task{ID: "recognition", Title: "Recognising a scan"})
	for telling := range 2 {
		list := receiveTasks(t, stream)
		if len(list) != 1 || list[0].GetId() != "recognition" {
			t.Fatalf("telling %d said %v", telling+1, list)
		}
	}

	tasks.Remove("recognition")
	for telling := range 2 {
		if list := receiveTasks(t, stream); len(list) != 0 {
			t.Fatalf("telling %d said %v, and the work had ended", telling+1, list)
		}
	}
}

// Two windows can be open on the same vault, so a request identifies its
// window. A request naming another window is not answered with this window's
// list.
func TestTasksStream_RejectsOtherWindowName(t *testing.T) {
	route, handler := numenv1connect.NewWindowServiceHandler(
		&Window{Named: "another", Tasking: run.NewTasks()},
	)
	mux := http.NewServeMux()
	mux.Handle(route, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := numenv1connect.NewWindowServiceClient(server.Client(), server.URL)
	stream, err := client.WatchTasks(t.Context(), connect.NewRequest(&v1.WatchTasksRequest{Window: Editor}))
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, stream)
	if stream.Receive() {
		t.Fatalf("another window answered for the editor: %v", stream.Msg().GetTasks())
	}
	if connect.CodeOf(stream.Err()) != connect.CodeNotFound {
		t.Errorf("the answer was %v", stream.Err())
	}

	if _, err := client.ReportFlush(t.Context(), connect.NewRequest(&v1.ReportFlushRequest{
		Window: Editor,
		Token:  "0",
		Result: v1.FlushResult_FLUSH_RESULT_WRITTEN,
	})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("a flush for the editor was answered %v", err)
	}
}

func TestWatchTasksSendsThePluginOfEachTask(t *testing.T) {
	tasks := run.NewTasks()
	stream := openTaskStream(t, tasks)
	receiveTasks(t, stream)

	tasks.Set(run.Task{ID: "one", Title: "Reading", PluginID: "md.example.engine", PluginName: "Example Engine"})
	list := receiveTasks(t, stream)
	if len(list) != 1 || list[0].GetPluginId() != "md.example.engine" || list[0].GetPluginName() != "Example Engine" {
		t.Fatalf("got %v, want the task of md.example.engine named Example Engine", list)
	}
}
