package agentrun

import (
	"context"
	"errors"
	"sync"
)

// Work is one task being worked: the responses it reports, the reader producing them, and what
// stops it.
//
//numen:api
type Work struct {
	// Output is where the reader writes the responses. The reader closes it when it is done.
	Output chan *Response

	conversation string
	stop         func() error
	reader       sync.WaitGroup
	once         sync.Once
	stopped      chan struct{}
}

// NewWork creates the work of a task requested in a conversation, empty for none. Stop ends the child
// the reader reads by calling stop once.
func NewWork(conversation string, buffer int, stop func() error) *Work {
	return &Work{
		Output:       make(chan *Response, buffer),
		conversation: conversation,
		stop:         stop,
		stopped:      make(chan struct{}),
	}
}

// End signals that the reader has finished. The registry counts the reader when it takes the work in, so a
// caller that does not start the reader calls End.
func (w *Work) End() { w.reader.Done() }

// Stopped is closed once the work is requested to stop.
func (w *Work) Stopped() <-chan struct{} { return w.stopped }

// Stop ends the child and everything it started, then waits for the reader to finish. It returns
// the error of the stop function.
func (w *Work) Stop() error {
	return stopAll([]*Work{w})
}

// StopByCancel returns a stop function for NewWork that calls cancel.
func StopByCancel(cancel context.CancelFunc) func() error {
	return func() error {
		cancel()
		return nil
	}
}

func stopAll(works []*Work) error {
	var failures []error
	for _, w := range works {
		w.once.Do(func() {
			close(w.stopped)
			if err := w.stop(); err != nil {
				failures = append(failures, err)
			}
		})
	}
	for _, w := range works {
		w.reader.Wait()
	}
	return errors.Join(failures...)
}

// ForwardResponses sends the responses of the work to send until the work is done, and stops the work
// when send fails or the call ends first.
func ForwardResponses(ctx context.Context, work *Work, send func(*Response) error) error {
	for {
		select {
		case <-ctx.Done():
			return joinStopError(ctx.Err(), work)
		case resp, ok := <-work.Output:
			if !ok {
				return nil
			}
			if err := send(resp); err != nil {
				return joinStopError(err, work)
			}
		}
	}
}

func joinStopError(cause error, work *Work) error {
	if stopErr := work.Stop(); stopErr != nil {
		return errors.Join(cause, stopErr)
	}
	return cause
}
