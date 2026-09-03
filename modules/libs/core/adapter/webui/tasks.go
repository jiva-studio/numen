package webui

import (
	"context"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/task"
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

	watch := a.Tasking.Watch(watching)
	repeat := time.NewTimer(again)
	repeat.Stop()
	defer repeat.Stop()

	var last []task.Task
	for {
		select {
		case list, standing := <-watch:
			if !standing {
				return nil
			}
			if err := out.Send(&v1.TasksResponse{Tasks: doing(list)}); err != nil {
				return err
			}
			last = list
			repeat.Reset(again)

		case <-repeat.C:
			if err := out.Send(&v1.TasksResponse{Tasks: doing(last)}); err != nil {
				return err
			}
		}
	}
}

// again is how long after a change the same list is said a second time. A
// window is handed each list by the write that follows it.
const again = time.Second

// doing is the work as the schema says it.
func doing(list []task.Task) []*v1.Task {
	out := make([]*v1.Task, 0, len(list))
	for _, at := range list {
		out = append(out, &v1.Task{
			Id:       at.ID,
			Doing:    at.Doing,
			About:    at.About,
			Done:     at.Done,
			Total:    at.Total,
			Failed:   at.Failed,
			Asked:    at.Asked,
			Counting: counted(at.Counting),
		})
	}
	return out
}

// counted is what a task counts, as the schema says it.
func counted(in task.Unit) v1.Counting {
	switch in {
	case task.Bytes:
		return v1.Counting_COUNTING_BYTES
	case task.Seconds:
		return v1.Counting_COUNTING_SECONDS
	case task.Things:
		return v1.Counting_COUNTING_THINGS
	}
	return v1.Counting_COUNTING_THINGS
}

// say puts one piece of work in the list, for a build that keeps one.
func (a *API) say(at task.Task) {
	if a.Tasking != nil {
		a.Tasking.Set(at)
	}
}

// finished takes one piece of work out of the list.
func (a *API) finished(id string) {
	if a.Tasking != nil {
		a.Tasking.Done(id)
	}
}
