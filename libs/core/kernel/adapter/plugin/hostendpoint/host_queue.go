package hostendpoint

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

var errNoQueue = errors.New("the host keeps no queue for plugins")

// queueEndpoint answers HostQueueService for the plugins that call the host.
type queueEndpoint struct {
	endpoint *Endpoint
}

var _ numenv1connect.HostQueueServiceHandler = (*queueEndpoint)(nil)

func (e *queueEndpoint) begin(header http.Header) (string, WorkQueue, error) {
	pluginID, err := e.endpoint.authenticate(header.Get("Authorization"))
	if err != nil {
		return "", nil, err
	}
	work := e.endpoint.callers.Lent.GetPluginWork()
	if work == nil {
		return "", nil, connect.NewError(connect.CodeUnavailable, errNoQueue)
	}
	return pluginID, work, nil
}

// EnqueueWork streams WAITING while other work holds the queue, RUNNING once
// the plugin's turn begins and CONTENDED once other work waits behind it. The
// stream ends when the turn does. The plugin is not stopped as idle while the
// stream lasts.
func (e *queueEndpoint) EnqueueWork(
	ctx context.Context, req *connect.Request[v1.EnqueueWorkRequest], stream *connect.ServerStream[v1.EnqueueWorkResponse],
) error {
	pluginID, work, err := e.begin(req.Header())
	if err != nil {
		return err
	}
	holder := plugin.Holder{PluginID: pluginID, Why: "wait for a turn in the work queue"}
	e.endpoint.callers.Leases.Acquire(pluginID, holder)
	defer e.endpoint.callers.Leases.Release(pluginID, holder)
	var waitErr error
	turn, err := work.EnqueueWork(ctx, pluginID, req.Msg.GetIsUserStarted(), func() {
		waitErr = stream.Send(&v1.EnqueueWorkResponse{State: v1.WorkState_WORK_STATE_WAITING})
	})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return connect.NewError(connect.CodeCanceled, ctxErr)
		}
		return connect.NewError(connect.CodeAborted, err)
	}
	defer turn.Release()
	if waitErr != nil {
		return waitErr
	}
	if err := stream.Send(&v1.EnqueueWorkResponse{State: v1.WorkState_WORK_STATE_RUNNING}); err != nil {
		return err
	}
	contended := turn.WatchWaiting()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-turn.WatchEnd():
			return nil
		case <-contended:
			contended = nil
			if err := stream.Send(&v1.EnqueueWorkResponse{State: v1.WorkState_WORK_STATE_CONTENDED}); err != nil {
				return err
			}
		}
	}
}

func (e *queueEndpoint) ReportTask(
	_ context.Context, req *connect.Request[v1.ReportTaskRequest],
) (*connect.Response[v1.ReportTaskResponse], error) {
	pluginID, work, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	task := req.Msg.GetTask()
	work.ReportTask(pluginID, plugin.Task{
		ID:            task.GetId(),
		Title:         task.GetTitle(),
		About:         task.GetAbout(),
		Done:          task.GetDone(),
		Total:         task.GetTotal(),
		Unit:          readWireUnit(task.GetUnit()),
		Error:         task.GetError(),
		IsUserStarted: task.GetIsUserStarted(),
	})
	if task.GetId() != "" {
		if isTaskFinished(task) {
			e.endpoint.tasks.release(e.endpoint.callers.Leases, pluginID, task.GetId())
		} else {
			e.endpoint.tasks.hold(e.endpoint.callers.Leases, pluginID, task.GetId(), task.GetTitle())
		}
	}
	return connect.NewResponse(&v1.ReportTaskResponse{}), nil
}

func (e *queueEndpoint) FinishTask(
	_ context.Context, req *connect.Request[v1.FinishTaskRequest],
) (*connect.Response[v1.FinishTaskResponse], error) {
	pluginID, work, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	work.FinishTask(pluginID, req.Msg.GetId())
	e.endpoint.tasks.release(e.endpoint.callers.Leases, pluginID, req.Msg.GetId())
	return connect.NewResponse(&v1.FinishTaskResponse{}), nil
}

func isTaskFinished(task *v1.Task) bool {
	return task.GetError() != "" || (task.GetTotal() > 0 && task.GetDone() >= task.GetTotal())
}

func readWireUnit(unit v1.Unit) plugin.TaskUnit {
	switch unit {
	case v1.Unit_UNIT_BYTES:
		return plugin.TaskUnitBytes
	case v1.Unit_UNIT_SECONDS:
		return plugin.TaskUnitSeconds
	case v1.Unit_UNIT_THINGS, v1.Unit_UNIT_UNSPECIFIED:
		return plugin.TaskUnitThings
	}
	return plugin.TaskUnitThings
}
