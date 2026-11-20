package host

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// InferenceQueue schedules native model loads and inference on the host's work queue.
type InferenceQueue func(context.Context, Work) (Turn, error)

// NewInferenceQueue connects to the host when a model needs a turn.
func NewInferenceQueue(endpoint func() (hostcall.Endpoint, error)) InferenceQueue {
	return func(ctx context.Context, work Work) (Turn, error) {
		address, err := endpoint()
		if err != nil {
			return nil, err
		}
		return NewQueueClient(address).EnqueueWork(ctx, work)
	}
}

// Acquire waits with ctx and holds the turn until release, including while canceled native work unwinds.
func (q InferenceQueue) Acquire(ctx context.Context, work Work) (Turn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	wait, cancel := context.WithCancel(context.WithoutCancel(ctx))
	stop := context.AfterFunc(ctx, cancel)
	turn, err := q(wait, work)
	stop()
	if err != nil {
		cancel()
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		cancel()
		return nil, errors.Join(err, turn.Release())
	}
	return &inferenceTurn{Turn: turn, cancel: cancel}, nil
}

type inferenceTurn struct {
	Turn
	cancel context.CancelFunc
}

func (t *inferenceTurn) Release() error {
	defer t.cancel()
	return t.Turn.Release()
}

// OpenModel opens a native model after its files are available and closes it if the host loses the turn.
func OpenModel[T interface{ Close() error }](ctx context.Context, queue InferenceQueue, open func() (T, error)) (model T, err error) {
	turn, err := queue.Acquire(ctx, Work{})
	if err != nil {
		return model, err
	}
	didOpen := false
	defer func() {
		if endErr := turn.Release(); endErr != nil {
			if didOpen {
				endErr = errors.Join(endErr, model.Close())
				var zero T
				model = zero
			}
			err = errors.Join(err, endErr)
		}
	}()
	model, err = open()
	didOpen = err == nil
	return model, err
}
