package acp

import (
	"context"

	"google.golang.org/protobuf/proto"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
)

// OperationTask is the name of the operation of the contract.
const OperationTask = "acp.task"

// Runner carries out acp.task. It sends the updates of one turn in order and returns the stop reason. It
// returns the error of the first send that fails, connection.NewTryNextError or connection.NewStopError
// for a turn that failed, and ctx.Err() for a turn that was canceled.
type Runner interface {
	RunTask(ctx context.Context, call connection.Call, request *TaskRequest, send func(*TaskFrame) error) (*TaskDone, error)
}

// Provide adds acp.task, served by runner, to provider and returns it. A nil provider is a new one.
func Provide(provider connection.Provider, runner Runner) connection.Provider {
	if provider == nil {
		provider = connection.Provider{}
	}
	provider[OperationTask] = func(ctx context.Context, call connection.Call, payload []byte, send func([]byte) error) ([]byte, error) {
		request := &TaskRequest{}
		if err := proto.Unmarshal(payload, request); err != nil {
			return nil, connection.NewStopError(connection.ReasonInvalidRequest)
		}
		done, err := runner.RunTask(ctx, call, request, func(frame *TaskFrame) error {
			data, err := proto.Marshal(frame)
			if err != nil {
				return err
			}
			return send(data)
		})
		if err != nil {
			return nil, err
		}
		return proto.Marshal(done)
	}
	return provider
}
