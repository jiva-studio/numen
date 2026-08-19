package webui

import (
	"context"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/task"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// Tasks reports everything the application is doing behind the window, for as
// long as the client listens.
//
// The whole list goes every time any of it changes, and the first goes at once:
// a window that opened while work was running has to be told about it, and a
// stream that says nothing until something changes is indistinguishable from one
// that never opened.
func (a *API) Tasks(
	ctx context.Context,
	_ *connect.Request[v1.TasksRequest],
	out *connect.ServerStream[v1.TasksResponse],
) error {
	if a.Tasking == nil {
		// An installation that does nothing behind the window still answers, so
		// that the window has one thing to listen to rather than two ways of
		// finding out whether it should.
		return out.Send(&v1.TasksResponse{})
	}

	watching, stop := context.WithCancel(ctx)
	defer stop()

	for list := range a.Tasking.Watch(watching) {
		if err := out.Send(&v1.TasksResponse{Tasks: doing(list)}); err != nil {
			return err
		}
	}
	return nil
}

// doing is the work as the schema says it.
func doing(list []task.Task) []*v1.Task {
	out := make([]*v1.Task, 0, len(list))
	for _, at := range list {
		out = append(out, &v1.Task{
			Id:     at.ID,
			Doing:  at.Doing,
			About:  at.About,
			Done:   at.Done,
			Total:  at.Total,
			Failed: at.Failed,
		})
	}
	return out
}
