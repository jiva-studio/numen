// Package jsonlrun runs a command-line agent that prints one JSON event per line, and reports the
// events as the responses of a task.
package jsonlrun

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/supervise"
)

const (
	outputBuffer = 64
	maxLine      = 4 << 20
)

var errNoOutput = errors.New("the program ended without output")

// Spec describes one run.
type Spec struct {
	Conversation string
	MaxSteps     int
	// Root is the working folder of the command; empty keeps the one Build set.
	Root string
	// Token is passed to the command as NUMEN_TOKEN when not empty.
	Token string
	// Build creates the command and the function that releases what the command needs. A nil release
	// function needs no release.
	Build func(ctx context.Context) (*exec.Cmd, func() error, error)
	// NewHandler creates the function the run calls with each decoded event.
	NewHandler func(r *Reader) func(event map[string]any)
}

// Reader reports the events of one run.
type Reader struct {
	work         *agentrun.Work
	registry     *agentrun.Registry
	conversation string
	limit        int
	completed    int
	sentCount    int
	isFinished   bool
}

// Send reports a response of the run.
func (r *Reader) Send(resp *agentrun.AskAgentResponse) {
	select {
	case r.work.Output <- resp:
	case <-r.work.Stopped():
		return
	}
	r.sentCount++
	if resp.GetToolFinished() != nil {
		r.completed++
	}
}

// CountStep counts a finished step towards the step limit.
func (r *Reader) CountStep() { r.completed++ }

// SetSession records the session of the program for the conversation of the run.
func (r *Reader) SetSession(session string) {
	if r.conversation != "" && session != "" {
		r.registry.SetSession(r.conversation, session)
	}
}

// Finish ends the run after the event being handled.
func (r *Reader) Finish() { r.isFinished = true }

func (r *Reader) isOverLimit() bool { return r.limit > 0 && r.completed >= r.limit }

func (r *Reader) reportLimit() {
	r.reportStop(fmt.Sprintf("reached maximum limit of %d steps", r.limit))
}

func (r *Reader) reportStop(reason string) {
	r.Send(&agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponseStopped{Stopped: reason}})
}

// Start starts the command of the spec and reads its output in the background. The work ends when
// the command does.
func Start(ctx context.Context, reg *agentrun.Registry, s Spec) (*agentrun.Work, error) {
	runCtx, stop := context.WithCancel(ctx)
	cmd, release, err := s.Build(runCtx)
	if err != nil {
		stop()
		return nil, err
	}
	if release == nil {
		release = func() error { return nil }
	}
	configureCommand(cmd, s)
	var stderr strings.Builder
	if cmd.Stderr == nil {
		cmd.Stderr = &stderr
	}
	supervise.Command(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stop()
		return nil, errors.Join(err, release())
	}
	w := agentrun.NewWork(s.Conversation, outputBuffer, agentrun.StopByCancel(stop))
	if !reg.Add(w) {
		stop()
		return nil, errors.Join(errors.New("agent runner is terminating"), stdout.Close(), release())
	}
	if err := cmd.Start(); err != nil {
		w.End()
		reg.Remove(w)
		stop()
		return nil, errors.Join(fmt.Errorf("start %s: %w", cmd.Path, err), stdout.Close(), release())
	}

	go read(runCtx, stop, cmd, stdout, &stderr, w, reg, s, release)
	return w, nil
}

func configureCommand(cmd *exec.Cmd, s Spec) {
	if s.Root != "" {
		cmd.Dir = s.Root
	}
	if len(cmd.Env) == 0 {
		cmd.Env = os.Environ()
	}
	if s.Token != "" {
		cmd.Env = append(cmd.Env, "NUMEN_TOKEN="+s.Token)
	}
}

func read(
	runCtx context.Context, stop func(), cmd *exec.Cmd, stdout io.ReadCloser,
	stderr *strings.Builder, w *agentrun.Work, reg *agentrun.Registry, s Spec, release func() error,
) {
	r := &Reader{
		work: w, registry: reg, conversation: s.Conversation, limit: s.MaxSteps,
		completed: 0, sentCount: 0, isFinished: false,
	}
	defer func() {
		stop()
		if err := release(); err != nil {
			r.reportStop(fmt.Sprintf("release the resources of the run: %v", err))
		}
		w.End()
		reg.Remove(w)
		close(w.Output)
	}()
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(nil, maxLine)
	isEnded := scanOutput(scanner, s.NewHandler(r), r, w, stop)
	scanErr := scanner.Err()
	isCancelled := runCtx.Err() != nil
	if scanErr != nil {
		stop()
	}
	waitErr := errors.Join(stdout.Close(), cmd.Wait())
	if isEnded || isCancelled {
		return
	}
	reportExit(r, scanErr, waitErr, stderr.String())
}

func scanOutput(
	scanner *bufio.Scanner, handle func(map[string]any), r *Reader, w *agentrun.Work, stop func(),
) bool {
	for scanner.Scan() {
		line := scanner.Bytes()
		var event map[string]any
		if err := json.Unmarshal(line, &event); err == nil {
			handle(event)
		} else if text := strings.TrimSpace(string(line)); text != "" {
			r.Send(&agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponseText{Text: text + "\n"}})
		}
		if r.isFinished {
			stop()
			return true
		}
		if r.isOverLimit() {
			r.reportLimit()
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

func reportExit(r *Reader, scanErr, waitErr error, stderr string) {
	switch {
	case scanErr != nil:
		r.reportStop(scanErr.Error())
	case waitErr != nil:
		r.reportStop(agentrun.FailureMessage(waitErr, stderr))
	case r.sentCount == 0 && strings.TrimSpace(stderr) != "":
		r.reportStop(agentrun.FailureMessage(errNoOutput, stderr))
	}
}
