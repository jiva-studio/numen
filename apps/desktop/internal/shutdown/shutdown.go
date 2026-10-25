// Package shutdown is the order the things a window holds are release in.
//
// The application runs the steps as it stops, and the run they were made in
// runs them where it returns. They happen once, and an ask arriving while they
// run waits for them.
package shutdown

import (
	"fmt"
	"io"
	"sync"
)

// Sequence are what is release, first to last.
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

// Report is a step that runs fn and writes its fault to out as "name: fault",
// or "name: label: fault" when label is set.
func Report(name string, out io.Writer, label string, fn func() error) func() {
	return func() {
		err := fn()
		switch {
		case err == nil:
		case label == "":
			fmt.Fprintf(out, "%s: %v\n", name, err)
		default:
			fmt.Fprintf(out, "%s: %s: %v\n", name, label, err)
		}
	}
}
