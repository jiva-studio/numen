package agentrun

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/completion"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// outputBuffer is how many responses the work holds before its reader waits for the stream.
const outputBuffer = 64

// maxLine is the longest line of output the reader takes.
const maxLine = 4 << 20

// Spec describes a child that writes one JSON event on each line of its output.
type Spec struct {
	Conversation string
	// MaxSteps is how many tool calls the child may finish before it is stopped. Zero is no limit.
	MaxSteps int
	// Root is the folder the child runs in. Empty is the folder of the plugin.
	Root string
	// Token is handed to the child as NUMEN_TOKEN. Empty hands none.
	Token string
	// Build makes the command under ctx, not started, and the function that cleans up after it. A
	// nil cleanup is none. An error ends the run before it starts. A command with no Env runs with the environment of the plugin.
	Build func(ctx context.Context) (*exec.Cmd, func(), error)
	// NewHandler is called once for the run and returns what is called with each event of the
	// output.
	NewHandler func(r *Reader) func(event map[string]any)
}

// Reader sends the responses of one run and counts the tools it has finished.
type Reader struct {
	work         *Work
	registry     *Registry
	conversation string
	limit        int
	done         int
}

// Send writes the response to the work, unless the work was asked to stop.
func (r *Reader) Send(resp *numenv1.AskAgentResponse) {
	select {
	case r.work.Output <- resp:
	case <-r.work.Stopped():
		return
	}
	if resp.GetToolFinished() != nil {
		r.done++
	}
}

// SetSession keeps the session the next task of the conversation is asked in. A run in no
// conversation keeps none.
// CountStep counts one step of the run, for a program whose stream does not say when a tool
// finished.
func (r *Reader) CountStep() { r.done++ }

func (r *Reader) SetSession(session string) {
	if r.conversation != "" && session != "" {
		r.registry.SetSession(r.conversation, session)
	}
}

func (r *Reader) isOverLimit() bool { return r.limit > 0 && r.done >= r.limit }

func (r *Reader) reportLimit() {
	r.reportStop(fmt.Sprintf("reached maximum limit of %d steps", r.limit))
}

func (r *Reader) reportStop(reason string) {
	r.Send(&numenv1.AskAgentResponse{Step: &numenv1.AskAgentResponse_Stopped{Stopped: reason}})
}

// Start starts the child of the spec and returns the work that reads its output. A line that is
// a JSON object goes to the handler; any other line is said as it is.
func Start(ctx context.Context, reg *Registry, spec Spec) (*Work, error) {
	running, stop := context.WithCancel(ctx)
	cmd, cleanup, err := spec.Build(running)
	if err != nil {
		stop()
		return nil, err
	}
	if cleanup == nil {
		cleanup = func() {}
	}
	if spec.Root != "" {
		cmd.Dir = spec.Root
	}
	if len(cmd.Env) == 0 {
		cmd.Env = os.Environ()
	}
	if spec.Token != "" {
		cmd.Env = append(cmd.Env, "NUMEN_TOKEN="+spec.Token)
	}
	var stderr strings.Builder
	if cmd.Stderr == nil {
		cmd.Stderr = &stderr
	}
	Supervise(cmd)

	fail := func(err error) (*Work, error) {
		cleanup()
		stop()
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fail(err)
	}
	w := NewWork(spec.Conversation, outputBuffer, stop)
	if !reg.Add(w) {
		_ = stdout.Close()
		return fail(errors.New("agent runner is terminating"))
	}
	if err := cmd.Start(); err != nil {
		w.End()
		reg.Remove(w)
		_ = stdout.Close()
		return fail(fmt.Errorf("start %s: %w", cmd.Path, err))
	}

	reader := &Reader{work: w, registry: reg, conversation: spec.Conversation, limit: spec.MaxSteps}
	handle := spec.NewHandler(reader)
	go func() {
		defer func() {
			cleanup()
			w.End()
			reg.Remove(w)
			close(w.Output)
		}()
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(nil, maxLine)
		isLimited := false
		for scanner.Scan() {
			line := scanner.Bytes()
			var event map[string]any
			if err := json.Unmarshal(line, &event); err == nil {
				handle(event)
			} else if text := strings.TrimSpace(string(line)); text != "" {
				reader.Send(&numenv1.AskAgentResponse{Step: &numenv1.AskAgentResponse_Said{Said: text + "\n"}})
			}
			if reader.isOverLimit() {
				reader.reportLimit()
				isLimited = true
				stop()
				break
			}
			select {
			case <-w.Stopped():
				_ = stdout.Close()
				_ = cmd.Wait()
				return
			default:
			}
		}
		scanErr := scanner.Err()
		_ = stdout.Close()
		waitErr := cmd.Wait()
		if isLimited || running.Err() != nil {
			return
		}
		switch {
		case scanErr != nil:
			reader.reportStop(scanErr.Error())
		case waitErr != nil:
			reader.reportStop(completion.FailureMessage(waitErr, stderr.String()))
		}
	}()
	return w, nil
}

// Relay sends the responses of the work to the stream until the work is done, and stops the work
// when the stream or the call ends first.
func Relay(ctx context.Context, work *Work, stream interface {
	Send(*numenv1.AskAgentResponse) error
}) error {
	for {
		select {
		case <-ctx.Done():
			_ = work.Stop()
			return ctx.Err()
		case resp, ok := <-work.Output:
			if !ok {
				return nil
			}
			if err := stream.Send(resp); err != nil {
				_ = work.Stop()
				return err
			}
		}
	}
}
