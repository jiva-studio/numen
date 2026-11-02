package testsupport

import (
	"context"
	"net/http"

	"github.com/jiva-studio/numen/libs/core/port/agentport"
)

// NewRootedHandler serves the handler the way a window does: the request
// context is the process's own, and ends when the application ends and at no
// other moment.
func NewRootedHandler(handler http.Handler, entered, returned chan<- struct{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		defer func() { returned <- struct{}{} }()
		handler.ServeHTTP(w, r.WithContext(context.WithoutCancel(r.Context())))
	})
}

// SilentAgent accepts a task and produces no output, as an agent waiting on a
// model does.
type SilentAgent struct{}

func (SilentAgent) StartRun(context.Context, agentport.Task) (agentport.Run, error) {
	return NoopRun{steps: make(chan agentport.Step)}, nil
}

func (SilentAgent) EndConversation(context.Context, string) error { return nil }

// NoopRun is a run of a SilentAgent: a step never arrives, and stopping it
// costs nothing.
type NoopRun struct{ steps chan agentport.Step }

func (w NoopRun) GetSteps() <-chan agentport.Step { return w.steps }
func (w NoopRun) Stop() error                     { return nil }
