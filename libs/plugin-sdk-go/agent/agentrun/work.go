package agentrun

import (
	"context"
	"errors"
	"sync"
)

// Work is one task being worked: the responses it reports, the reader producing them, and what
// stops it.
type Work struct {
	// Output is where the reader writes the responses. The reader closes it when it is done.
	Output chan *AskAgentResponse

	conversation string
	stop         func()
	reader       sync.WaitGroup
	once         sync.Once
	stopped      chan struct{}
}

// NewWork creates the work of a task requested in a conversation, empty for none. Stop ends the child
// the reader reads.
func NewWork(conversation string, buffer int, stop func()) *Work {
	return &Work{
		Output:       make(chan *AskAgentResponse, buffer),
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

// Stop ends the child and everything it started, then waits for the reader to finish.
func (w *Work) Stop() error {
	stopAll([]*Work{w})
	return nil
}

// stopAll cancels each work and waits for its reader to finish.
func stopAll(works []*Work) {
	for _, w := range works {
		w.once.Do(func() {
			close(w.stopped)
			w.stop()
		})
	}
	for _, w := range works {
		w.reader.Wait()
	}
}

// ForwardResponses sends the responses of the work to the stream until the work is done, and stops the work
// when the stream or the call ends first.
func ForwardResponses(ctx context.Context, work *Work, stream interface {
	Send(*AskAgentResponse) error
}) error {
	for {
		select {
		case <-ctx.Done():
			return joinStopError(ctx.Err(), work)
		case resp, ok := <-work.Output:
			if !ok {
				return nil
			}
			if err := stream.Send(resp); err != nil {
				return joinStopError(err, work)
			}
		}
	}
}

// joinStopError stops the work and returns cause, joined with the stop error when stopping fails.
func joinStopError(cause error, work *Work) error {
	if stopErr := work.Stop(); stopErr != nil {
		return errors.Join(cause, stopErr)
	}
	return cause
}
