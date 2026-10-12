// Package shutdown is the order the things a window holds are let go of in.
//
// The application runs the steps as it stops, and the run they were made in
// runs them where it returns. They happen once, and an ask arriving while they
// run waits for them.
package shutdown

import "sync"

// Sequence are what is let go of, first to last.
type Sequence struct {
	steps []func()
	once  sync.Once
}

// NewSequence is the steps, in the order they are to run.
func NewSequence(steps ...func()) *Sequence {
	return &Sequence{steps: steps}
}

// Run runs each step in turn. Every call after the first does nothing, and one
// arriving while the steps are running waits for them.
func (s *Sequence) Run() {
	s.once.Do(func() {
		for _, step := range s.steps {
			step()
		}
	})
}
