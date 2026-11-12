package llmtest

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
)

// NewCaller returns a Caller that carries out each operation with the operation of provider, for call, in
// the process of the test. It runs an operation to its end before it returns the frames, and the frames
// end with the error of the operation as it is.
func NewCaller(provider connection.Provider, call connection.Call) connection.Caller {
	return &caller{provider: provider, call: call}
}

type caller struct {
	provider connection.Provider
	call     connection.Call
}

func (c *caller) Call(ctx context.Context, operation string, payload []byte, _ connection.Priority) (connection.Frames, error) {
	served, isServed := c.provider[operation]
	if !isServed {
		return nil, fmt.Errorf("llmtest: the provider does not serve %s", operation)
	}
	frames := &frames{next: -1}
	frames.done, frames.err = served(ctx, c.call, payload, func(data []byte) error {
		frames.data = append(frames.data, data)
		return nil
	})
	return frames, nil
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
