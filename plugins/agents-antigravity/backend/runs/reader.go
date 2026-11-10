package runs

import (
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
)

type reader struct {
	work         *agentrun.Work
	registry     *agentrun.Registry
	conversation string
	limit        int
	completed    int
	sentCount    int
	isFinished   bool
}

func (r *reader) Send(resp *agentrun.Response) {
	select {
	case r.work.Output <- resp:
	case <-r.work.Stopped():
		return
	}
	r.sentCount++
	if resp.IsToolFinished() {
		r.completed++
	}
}

func (r *reader) CountStep() { r.completed++ }

func (r *reader) Finish() { r.isFinished = true }

func (r *reader) SetSession(session string) {
	if r.conversation != "" && session != "" {
		r.registry.SetSession(r.conversation, session)
	}
}

func (r *reader) isOverLimit() bool { return r.limit > 0 && r.completed >= r.limit }

func (r *reader) reportLimit() {
	r.reportStop(fmt.Sprintf("reached maximum limit of %d steps", r.limit))
}

func (r *reader) reportStop(reason string) {
	r.Send(&agentrun.Response{Step: &agentrun.StoppedStep{Stopped: reason}})
}
