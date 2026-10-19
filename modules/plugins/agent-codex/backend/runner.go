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

// Endpoint is where the agent reaches the tools of this vault over MCP.
type Endpoint struct {
	URL   string
	Token string
}

// AgentRunner executes the Codex CLI and emits AskAgentResponse events.
type AgentRunner struct {
	Registry *agentrun.Registry
}

// RunParams specifies execution options for a Codex invocation.
type RunParams struct {
	Command      []string
	Model        string
	Prompt       string
	Conversation string
	Tools        Endpoint
	MaxSteps     int
	Root         string
}

// StartRun starts the Codex process and returns the Work handle.
func (r *AgentRunner) StartRun(ctx context.Context, params RunParams) (*agentrun.Work, error) {
	running, stop := context.WithCancel(ctx)
	binName, baseArgs := resolveCommand(params)
	cmd := makeCodexCommand(running, binName, baseArgs, params)

	configureCommand(cmd, params)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stop()
		return nil, err
	}

	w := agentrun.NewWork(params.Conversation, 64, stop)
	if !r.Registry.Add(w) {
		_ = stdout.Close()
		stop()
		return nil, errors.New("agent runner is terminating")
	}

	if err := cmd.Start(); err != nil {
		w.End()
		r.Registry.Remove(w)
		_ = stdout.Close()
		stop()
		return nil, fmt.Errorf("start %s: %w", binName, err)
	}

	go readAgentOutput(w, r.Registry, cmd, stdout, params.MaxSteps)

	return w, nil
}

func resolveCommand(params RunParams) (string, []string) {
	if len(params.Command) > 0 {
		return params.Command[0], params.Command[1:]
	}
	return FindCodex(), nil
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

func makeCodexCommand(
	running context.Context, binName string, baseArgs []string, params RunParams,
) *exec.Cmd {
	args := append([]string{}, baseArgs...)
	args = append(args,
		"exec",
		"--json",
		"--dangerously-bypass-approvals-and-sandbox",
	)
	if params.Tools.URL != "" {
		mcpURL := fmt.Sprintf(`mcp_servers.numen.url=%q`, params.Tools.URL)
		mcpHeader := fmt.Sprintf(`mcp_servers.numen.http_headers={"Authorization"="Bearer %s"}`, params.Tools.Token)
		args = append(args, "-c", mcpURL, "-c", mcpHeader)
	}
	if params.Model != "" && params.Model != "default" {
		args = append(args, "--model", params.Model)
	}
	args = append(args, params.Prompt)
	return exec.CommandContext(running, binName, args...)
}

func readAgentOutput(
	w *agentrun.Work,
	reg *agentrun.Registry,
	cmd *exec.Cmd,
	stdout io.ReadCloser,
	maxSteps int,
) {
	defer func() {
		w.End()
		reg.Remove(w)
		close(w.Output)
	}()

	reader := &eventReader{
		work:  w,
		limit: maxSteps,
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
