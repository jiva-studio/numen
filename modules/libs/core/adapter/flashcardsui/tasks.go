package flashcardsui

import (
	"context"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/task"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// Tasks reports everything this window is doing behind itself, for as long as
// the client listens.
//
// The whole list goes every time any of it changes, and the first goes at once:
// a window that opened while a vault was being read has to be told about it.
func (a *API) Tasks(
	ctx context.Context,
	_ *connect.Request[v1.FlashcardsServiceTasksRequest],
	out *connect.ServerStream[v1.FlashcardsServiceTasksResponse],
) error {
	if a.Tasking == nil {
		// A window that does nothing behind itself still answers, so the page has
		// one thing to listen to.
		return out.Send(&v1.FlashcardsServiceTasksResponse{})
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
			if err := out.Send(&v1.FlashcardsServiceTasksResponse{Tasks: doing(list)}); err != nil {
				return err
			}
			last = list
			repeat.Reset(again)

		case <-repeat.C:
			if err := out.Send(&v1.FlashcardsServiceTasksResponse{Tasks: doing(last)}); err != nil {
				return err
			}
		}
	}
}

// again is how long after a change the same list is said a second time. A window
// is handed each list by the write that follows it.
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
			Counting: countedIn(at.Counting),
		})
	}
	return out
}

// countedIn is what a task counts, as the schema says it.
func countedIn(in task.Counting) v1.Counting {
	if in == task.Bytes {
		return v1.Counting_COUNTING_BYTES
	}
	return v1.Counting_COUNTING_THINGS
}

// say puts one piece of work in the list, for a window that keeps one.
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
