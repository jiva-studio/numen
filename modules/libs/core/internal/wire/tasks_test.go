package wire

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/task"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// listening is the stream a window would hold, over a list a test drives.
func listening(t *testing.T, tasks *task.Tasks) *connect.ServerStreamForClient[v1.WatchTasksResponse] {
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
	t.Cleanup(func() { _ = stream.Close() })
	return stream
}

// told is the list the stream says next.
func told(t *testing.T, stream *connect.ServerStreamForClient[v1.WatchTasksResponse]) []*v1.Task {
	t.Helper()
	if !stream.Receive() {
		t.Fatalf("the stream ended: %v", stream.Err())
	}
	return stream.Msg().GetTasks()
}

// A window is handed each list by the write that follows it, so a list with
// nothing after it is a list nobody is holding: work that ended stands on
// screen as work still running.
func TestOneChangeIsSaidTwice(t *testing.T) {
	tasks := task.New()
	stream := listening(t, tasks)

	if first := told(t, stream); len(first) != 0 {
		t.Fatalf("a window that opened with nothing running was told of %d", len(first))
	}

	tasks.Set(task.Task{ID: "reading", Doing: "Reading a scan"})
	for telling := range 2 {
		list := told(t, stream)
		if len(list) != 1 || list[0].GetId() != "reading" {
			t.Fatalf("telling %d said %v", telling+1, list)
		}
	}

	tasks.Done("reading")
	for telling := range 2 {
		if list := told(t, stream); len(list) != 0 {
			t.Fatalf("telling %d said %v, and the work had ended", telling+1, list)
		}
	}
}

// The editor and the window a person runs their cards in are open on the same
// vault, so a question about a window says which. One naming the other window
// is not answered with this one's.
func TestAQuestionNamingAnotherWindowIsNotAnswered(t *testing.T) {
	route, handler := numenv1connect.NewWindowServiceHandler(
		&Window{Named: Review, Tasking: task.New()},
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
	t.Cleanup(func() { _ = stream.Close() })
	if stream.Receive() {
		t.Fatalf("the review window answered for the editor: %v", stream.Msg().GetTasks())
	}
	if connect.CodeOf(stream.Err()) != connect.CodeNotFound {
		t.Errorf("the answer was %v", stream.Err())
	}

	if _, err := client.ReportFlush(t.Context(), connect.NewRequest(&v1.ReportFlushRequest{
		Window: Editor,
		Token:  "0",
		Owed:   v1.Owed_OWED_WRITTEN,
	})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("a flush for the editor was answered %v", err)
	}
}
