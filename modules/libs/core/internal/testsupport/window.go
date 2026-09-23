package testsupport

import (
	"context"
	"net/http"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// NewRootedHandler serves the handler the way a window does: the request
// context is the process's own, and ends when the application ends and at no
// other moment.
func NewRootedHandler(handler http.Handler, entered, returned chan<- struct{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		defer func() { returned <- struct{}{} }()
		handler.ServeHTTP(w, r.WithContext(context.Background()))
	})
}

// SilentAgent takes a task and says nothing about it, which is what an agent
// waiting on a model looks like.
type SilentAgent struct{}

func (SilentAgent) Take(context.Context, port.Task) (port.Run, error) {
	return SilentWork{steps: make(chan port.Step)}, nil
}

func (SilentAgent) Finish(context.Context, string) error { return nil }

// SilentWork is a run of a SilentAgent: a step never arrives, and stopping it
// costs nothing.
type SilentWork struct{ steps chan port.Step }

func (w SilentWork) Steps() <-chan port.Step { return w.steps }
func (w SilentWork) Stop() error             { return nil }
