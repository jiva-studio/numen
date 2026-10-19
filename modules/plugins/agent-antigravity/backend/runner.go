package main

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

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// AgentRunner executes the Antigravity CLI and emits AskAgentResponse events.
type AgentRunner struct {
	Registry *agentrun.Registry
}

// RunParams specifies execution options for an Antigravity invocation.
type RunParams struct {
	Command      []string
	Model        string
	Prompt       string
	Conversation string
	Tools        Endpoint
	MaxSteps     int
	Root         string
}

// StartRun starts the Antigravity process and returns the Work handle.
func (r *AgentRunner) StartRun(ctx context.Context, params RunParams) (*agentrun.Work, error) {
	running, stop := context.WithCancel(ctx)
	binName, baseArgs := resolveCommand(params)
	cmd, cleanupTempHome := makeAntigravityCommand(running, binName, baseArgs, params, r.Registry)

	configureCommand(cmd, params)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		if cleanupTempHome != nil {
			cleanupTempHome()
		}
		stop()
		return nil, err
	}

	w := agentrun.NewWork(params.Conversation, 64, stop)
	if !r.Registry.Add(w) {
		_ = stdout.Close()
		if cleanupTempHome != nil {
			cleanupTempHome()
		}
		stop()
		return nil, errors.New("agent runner is terminating")
	}

	if err := cmd.Start(); err != nil {
		w.End()
		r.Registry.Remove(w)
		_ = stdout.Close()
		if cleanupTempHome != nil {
			cleanupTempHome()
		}
		stop()
		return nil, fmt.Errorf("start %s: %w", binName, err)
	}

	go readAgentOutput(w, r.Registry, cmd, stdout, cleanupTempHome, params.MaxSteps, params.Conversation)

	return w, nil
}

func resolveCommand(params RunParams) (string, []string) {
	if len(params.Command) > 0 {
		return params.Command[0], params.Command[1:]
	}
	return FindAntigravity(), nil
}

func configureCommand(cmd *exec.Cmd, params RunParams) {
	if params.Root != "" {
		cmd.Dir = params.Root
	}
	if len(cmd.Env) == 0 {
		cmd.Env = os.Environ()
	}
	if params.Tools.Token != "" {
		cmd.Env = append(cmd.Env, "NUMEN_TOKEN="+params.Tools.Token)
	}
	agentrun.Supervise(cmd)
}

func makeAntigravityCommand(
	running context.Context, binName string, baseArgs []string, params RunParams, reg *agentrun.Registry,
) (*exec.Cmd, func()) {
	args := append([]string{}, baseArgs...)
	args = append(args,
		"-p", params.Prompt,
		"--output-format", "stream-json",
		"--dangerously-skip-permissions",
	)
	if params.Model != "" && params.Model != "default" {
		args = append(args, "--model", params.Model)
	}
	if sessID := reg.GetSession(params.Conversation); sessID != "" {
		args = append(args, "--conversation", sessID)
	}

	cmd := exec.CommandContext(running, binName, args...)
	if params.Tools.URL != "" {
		tempHome, clean, err := setupAntigravityHome(params.Tools)
		if err == nil {
			cmd.Env = append(os.Environ(), "HOME="+tempHome)
			return cmd, clean
		}
	}
	return cmd, nil
}

func readAgentOutput(
	w *agentrun.Work,
	reg *agentrun.Registry,
	cmd *exec.Cmd,
	stdout io.ReadCloser,
	cleanup func(),
	maxSteps int,
	conversation string,
) {
	defer func() {
		if cleanup != nil {
			cleanup()
		}
		w.End()
		reg.Remove(w)
		close(w.Output)
	}()

	reader := &eventReader{
		registry:     reg,
		work:         w,
		limit:        maxSteps,
		conversation: conversation,
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(nil, 4<<20)

	for scanner.Scan() {
		line := scanner.Bytes()
		var rawMap map[string]any
		if err := json.Unmarshal(line, &rawMap); err == nil {
			reader.handle(rawMap)
		} else if text := strings.TrimSpace(string(line)); text != "" {
			reader.send(&numenv1.AskAgentResponse{
				Step: &numenv1.AskAgentResponse_Said{
					Said: text + "\n",
				},
			})
		}

		if reader.isOverLimit() {
			reader.reportLimit()
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

	_ = stdout.Close()
	_ = cmd.Wait()
}
