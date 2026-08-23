package webui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/task"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// listening is the stream a window would hold, over a list a test drives.
func listening(t *testing.T, tasks *task.Tasks) *connect.ServerStreamForClient[v1.TasksResponse] {
	t.Helper()

	route, handler := numenv1connect.NewVaultServiceHandler(&API{Tasking: tasks})
	mux := http.NewServeMux()
	mux.Handle(route, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	ctx, stop := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(stop)

	client := numenv1connect.NewVaultServiceClient(server.Client(), server.URL)
	stream, err := client.Tasks(ctx, connect.NewRequest(&v1.TasksRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stream.Close() })
	return stream
}

// told is the list the stream says next.
func told(t *testing.T, stream *connect.ServerStreamForClient[v1.TasksResponse]) []*v1.Task {
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
