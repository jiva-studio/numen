package run

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/codexevent"
)

type reader struct {
	work         *agentrun.Work
	registry     *agentrun.Registry
	conversation string
	limit        int
	completed    int
	sentCount    int
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

func (r *reader) isOverLimit() bool { return r.limit > 0 && r.completed >= r.limit }

func (r *reader) reportLimit() {
	r.reportStop(fmt.Sprintf("reached maximum limit of %d steps", r.limit))
}

func (r *reader) reportStop(reason string) {
	r.Send(&agentrun.Response{Step: &agentrun.StoppedStep{Stopped: reason}})
}

func newOutputScanner(stdout io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(nil, maxLine)
	return scanner
}

func scanOutput(
	scanner *bufio.Scanner, handle func(map[string]any), r *reader, w *agentrun.Work, stop func(),
) bool {
	for scanner.Scan() {
		line := scanner.Bytes()
		if event, ok := codexevent.DecodeLine(line); ok {
			handle(event)
		} else if text := strings.TrimSpace(string(line)); text != "" {
			r.Send(&agentrun.Response{Step: &agentrun.TextStep{Text: text + "\n"}})
		}
		if r.isOverLimit() {
			r.reportLimit()
			// gomutants:disable-next-line STATEMENT_REMOVE reason="Calling stop() when the step limit is reached cancels the child before runReader's defer calls stop()."
			stop()
			return true
		}
		select {
		case <-w.Stopped():
			return false
		default:
		}
	}
	return false
}

func reportExit(r *reader, scanErr, waitErr error, stderr string) {
	switch {
	case scanErr != nil:
		r.reportStop(scanErr.Error())
	case waitErr != nil:
		r.reportStop(agentrun.FailureMessage(waitErr, stderr))
	case r.sentCount == 0 && strings.TrimSpace(stderr) != "":
		r.reportStop(agentrun.FailureMessage(errNoOutput, stderr))
	}
}
