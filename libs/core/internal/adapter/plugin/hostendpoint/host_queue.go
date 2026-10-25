package hostendpoint

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

var errNoQueue = errors.New("the host keeps no queue for plugins")

// queueEndpoint answers HostQueueService for the plugins that call the host.
type queueEndpoint struct {
	endpoint *Endpoint
}

var _ numenv1connect.HostQueueServiceHandler = (*queueEndpoint)(nil)

// begin authenticates the call and returns the plugin that made it and the
// queue it works.
func (e *queueEndpoint) begin(header http.Header) (string, pluginport.PluginWork, error) {
	pluginID, err := e.endpoint.authenticate(header.Get("Authorization"))
	if err != nil {
		return "", nil, err
	}
	work := e.endpoint.callers.GetPluginWork()
	if work == nil {
		return "", nil, connect.NewError(connect.CodeUnavailable, errNoQueue)
	}
	return pluginID, work, nil
}

// EnqueueWork streams WAITING while other work holds the queue, RUNNING once
// the plugin's turn begins and CONTENDED once other work waits behind it. The
// stream ends when the turn does.
func (e *queueEndpoint) EnqueueWork(
	ctx context.Context, req *connect.Request[v1.EnqueueWorkRequest], stream *connect.ServerStream[v1.EnqueueWorkResponse],
) error {
	pluginID, work, err := e.begin(req.Header())
	if err != nil {
		return err
	}
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
	waiting := turn.WatchWaiting()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-turn.WatchEnd():
			return nil
		case <-waiting:
			waiting = nil
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
	work.ReportTask(pluginID, pluginport.PluginTask{
		ID:            task.GetId(),
		Title:         task.GetTitle(),
		About:         task.GetAbout(),
		Done:          task.GetDone(),
		Total:         task.GetTotal(),
		Unit:          readWireUnit(task.GetUnit()),
		Error:         task.GetError(),
		IsUserStarted: task.GetIsUserStarted(),
	})
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
	return connect.NewResponse(&v1.FinishTaskResponse{}), nil
}

func readWireUnit(unit v1.Unit) pluginport.TaskUnit {
	switch unit {
	case v1.Unit_UNIT_BYTES:
		return pluginport.TaskUnitBytes
	case v1.Unit_UNIT_SECONDS:
		return pluginport.TaskUnitSeconds
	case v1.Unit_UNIT_THINGS, v1.Unit_UNIT_UNSPECIFIED:
		return pluginport.TaskUnitThings
	}
	return pluginport.TaskUnitThings
}
