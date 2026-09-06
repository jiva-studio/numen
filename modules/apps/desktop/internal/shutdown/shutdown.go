// Package shutdown is the order the things a window holds are let go of in.
//
// The application runs the steps as it stops, and the run they were made in
// runs them where it returns. They happen once, and an ask arriving while they
// run waits for them.
package shutdown

import "sync"

// Steps are what is let go of, first to last.
type Steps struct {
	steps []func()
	once  sync.Once
}

// InOrder is the steps, in the order they are to run.
func InOrder(steps ...func()) *Steps {
	return &Steps{steps: steps}
}

// Go runs each step in turn. Every call after the first does nothing, and one
// arriving while the steps are running waits for them.
func (s *Steps) Go() {
	s.once.Do(func() {
		for _, step := range s.steps {
			step()
		}
	})
}
