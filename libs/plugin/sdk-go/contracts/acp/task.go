package acp

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
)

// OperationTask is the name of the operation of the contract.
const OperationTask = "acp.task"

// TaskStream is the updates of one prompt turn. A reader calls Next until it returns false, then reads
// Err, and reads Done where Err is nil.
type TaskStream struct {
	frames connection.Frames
	frame  *TaskFrame
	done   *TaskDone
	err    error
}

// RunTask asks the rows of the plugin, in order, to carry out one prompt turn. The core goes to the next
// row only when a row fails before its first update. The caller closes the stream.
func RunTask(ctx context.Context, caller connection.Caller, request *TaskRequest, priority connection.Priority) (*TaskStream, error) {
	payload, err := proto.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("acp: encode %s: %w", OperationTask, err)
	}
	frames, err := caller.Call(ctx, OperationTask, payload, priority)
	if err != nil {
		return nil, err
	}
	return &TaskStream{frames: frames}, nil
}

// Next moves to the next update. It returns false at the end of the turn and on an error.
func (s *TaskStream) Next() bool {
	if s.err != nil || !s.frames.Next() {
		return false
	}
	frame := &TaskFrame{}
	if err := proto.Unmarshal(s.frames.Data(), frame); err != nil {
		s.err = fmt.Errorf("acp: decode an update of %s: %w", OperationTask, err)
		return false
	}
	s.frame = frame
	return true
}

// Frame is the update Next moved to.
func (s *TaskStream) Frame() *TaskFrame { return s.frame }

// Done holds the stop reason of the turn, once Next returned false and Err is nil.
func (s *TaskStream) Done() *TaskDone {
	if s.done == nil && s.err == nil && s.frames.Err() == nil {
		done := &TaskDone{}
		if err := proto.Unmarshal(s.frames.Done(), done); err != nil {
			s.err = fmt.Errorf("acp: decode the end of %s: %w", OperationTask, err)
			return nil
		}
		s.done = done
	}
	return s.done
}

// Err returns why the turn ended: nil after the done, or the errors of connection.Frames.Err.
func (s *TaskStream) Err() error { return errors.Join(s.err, s.frames.Err()) }

// Close ends the stream and cancels the turn if it has not ended.
func (s *TaskStream) Close() error { return s.frames.Close() }

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
