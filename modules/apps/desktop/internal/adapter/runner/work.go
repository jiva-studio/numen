package runner

import (
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Work is one task being worked: the steps it reports, the reader producing them, and what stops
// it. It is the port.Run of every agent.
type Work struct {
	// Output is where the reader writes the steps. The reader closes it when it is done.
	Output chan port.Step

	conversation string
	stop         func()
	reader       sync.WaitGroup
	once         sync.Once
}

// NewWork makes the work of a task asked in a conversation, empty for none. Stop ends the child
// the reader reads.
func NewWork(conversation string, buffer int, stop func()) *Work {
	return &Work{Output: make(chan port.Step, buffer), conversation: conversation, stop: stop}
}

// Begin counts a reader that must finish before the work is over.
func (w *Work) Begin() { w.reader.Add(1) }

// End says the reader has finished.
func (w *Work) End() { w.reader.Done() }

// GetSteps returns the steps in the order the agent takes them.
func (w *Work) GetSteps() <-chan port.Step { return w.Output }

// Stop ends the child and everything it started, then waits for the reader to finish.
func (w *Work) Stop() error {
	stopAll([]*Work{w})
	return nil
}

// stopAll cancels each work and waits for its reader to finish.
func stopAll(works []*Work) {
	for _, w := range works {
		w.once.Do(w.stop)
	}
	for _, w := range works {
		w.reader.Wait()
	}
}
