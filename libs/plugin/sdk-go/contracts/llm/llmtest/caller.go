package llmtest

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
)

// NewCaller returns a Caller that carries out each operation with the operation of provider, for call, in
// the process of the test, as the core carries it out for one row. It runs an operation to its end before
// it returns the frames. The frames end with the errors of connection.Frames.Err, which name the
// connection by call.ConnectionID: a stop, or an error that is no failure kind, is a *StoppedError; a
// try-next, or a failure after the first data frame, is an *AllFailedError with one failure; an operation
// the provider does not serve is ErrNoConnection; a canceled call is the error of the context. Unlike the
// core, the caller tries a row that is unavailable only once.
func NewCaller(provider connection.Provider, call connection.Call) connection.Caller {
	return &caller{provider: provider, call: call}
}

type caller struct {
	provider connection.Provider
	call     connection.Call
}

func (c *caller) Call(ctx context.Context, operation string, payload []byte, _ connection.Priority) (connection.Frames, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	frames := &frames{next: -1}
	served, isServed := c.provider[operation]
	if !isServed {
		frames.err = connection.ErrNoConnection
		return frames, nil
	}
	done, err := served(ctx, c.call, payload, func(data []byte) error {
		frames.data = append(frames.data, data)
		return nil
	})
	frames.done, frames.err = done, c.readError(ctx, err, len(frames.data) > 0)
	return frames, nil
}

func (c *caller) readError(ctx context.Context, err error, hasData bool) error {
	if err == nil {
		return nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	reason, _ := connection.GetReason(err)
	if _, isTryNext := connection.IsTryNext(err); isTryNext || hasData {
		return &connection.AllFailedError{Failures: []connection.Failure{{ConnectionName: c.call.ConnectionID, Reason: reason.String()}}}
	}
	return &connection.StoppedError{ConnectionName: c.call.ConnectionID, Reason: reason.String()}
}

// frames are the frames an operation sent, and its done payload or its error.
type frames struct {
	data [][]byte
	next int
	done []byte
	err  error
}

func (f *frames) Next() bool {
	if f.next+1 >= len(f.data) {
		f.next = len(f.data)
		return false
	}
	f.next++
	return true
}

func (f *frames) Data() []byte {
	if f.next < 0 || f.next >= len(f.data) {
		return nil
	}
	return f.data[f.next]
}

func (f *frames) Done() []byte { return f.done }

func (f *frames) Err() error { return f.err }

func (f *frames) Close() error { return nil }
