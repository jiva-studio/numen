package acp

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
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// StartRun starts the agent on a task.
func (a *Agent) StartRun(ctx context.Context, task port.Task) (port.Run, error) {
	if a.Tools.URL == "" || a.Tools.Token == "" {
		return nil, errors.New("no tools to give an agent")
	}

	running, stop := context.WithCancel(ctx)
	binName, baseArgs := a.resolveCommand()

	isAntigravity := a.Program == "antigravity" || strings.HasSuffix(binName, "agy") || strings.HasSuffix(binName, "antigravity")
	isCodex := a.Program == "codex" || strings.HasSuffix(binName, "codex")
	isRPC := !isAntigravity && !isCodex

	var cmd *exec.Cmd
	var cleanupTempHome func()
	switch {
	case isAntigravity:
		cmd, cleanupTempHome = a.makeAntigravityCommand(running, binName, baseArgs, task)
	case isCodex:
		cmd = a.makeCodexCommand(running, binName, baseArgs, task)
	default:
		cmd = exec.CommandContext(running, binName, baseArgs...)
	}
	a.configureCommand(cmd)

	abandon := func(err error, pipes ...io.Closer) (port.Run, error) {
		for _, pipe := range pipes {
			pipe.Close()
		}
		if cleanupTempHome != nil {
			cleanupTempHome()
		}
		stop()
		return nil, err
	}

	opened, err := openPipes(cmd, isRPC)
	if err != nil {
		return abandon(err)
	}
	pipes := opened.getClosers()

	w := &work{
		cmd:          cmd,
		stop:         stop,
		conversation: task.Conversation,
		steps:        make(chan port.Step, 64),
		cleanup:      cleanupTempHome,
	}

	if !a.registerWork(w) {
		return abandon(errors.New("this agent is closing"), pipes...)
	}
	if err := cmd.Start(); err != nil {
		a.unregisterWork(w)
		return abandon(fmt.Errorf("start %s: %w", binName, err), pipes...)
	}

	w.reader.Add(1)
	go a.read(w, opened.stdin, opened.stdout, task, isRPC)

	return w, nil
}

// configureCommand sets where the process runs and what it is given, and how it
// is ended.
func (a *Agent) configureCommand(cmd *exec.Cmd) {
	cmd.Dir = a.Root
	if len(cmd.Env) == 0 {
		cmd.Env = os.Environ()
	}
	cmd.Env = append(cmd.Env, "NUMEN_TOKEN="+a.Tools.Token)
	detach(cmd)

	cmd.Cancel = func() error { return kill(cmd) }
	cmd.WaitDelay = 2 * time.Second
}

// processPipes are the pipes of the process: its input where it speaks
// JSON-RPC, and its output.
type processPipes struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

// openPipes opens the pipes of the process. Nothing stays open when an error is
// returned.
func openPipes(cmd *exec.Cmd, isRPC bool) (processPipes, error) {
	var opened processPipes
	if isRPC {
		var err error
		if opened.stdin, err = cmd.StdinPipe(); err != nil {
			return processPipes{}, err
		}
	}
	var err error
	if opened.stdout, err = cmd.StdoutPipe(); err != nil {
		if opened.stdin != nil {
			opened.stdin.Close()
		}
		return processPipes{}, err
	}
	return opened, nil
}

// getClosers are the pipes that were opened.
func (p processPipes) getClosers() []io.Closer {
	if p.stdin == nil {
		return []io.Closer{p.stdout}
	}
	return []io.Closer{p.stdin, p.stdout}
}

// getModel is the model that answers a task: the one the task names, or else
// the one the agent is set to.
func (a *Agent) getModel(task port.Task) string {
	if task.Model != "" {
		return task.Model
	}
	return a.Model
}

// makeAntigravityCommand is the antigravity program asked the task, with a home
// folder carrying the vault's tools, and the cleanup of that folder.
func (a *Agent) makeAntigravityCommand(
	running context.Context, binName string, baseArgs []string, task port.Task,
) (*exec.Cmd, func()) {
	args := append([]string{}, baseArgs...)
	args = append(args,
		"-p", task.Question,
		"--output-format", "stream-json",
		"--dangerously-skip-permissions",
	)
	if model := a.getModel(task); model != "" && model != "default" {
		args = append(args, "--model", model)
	}
	if sessID := a.getCarriedSession(task.Conversation); sessID != "" {
		args = append(args, "--conversation", sessID)
	}

	cmd := exec.CommandContext(running, binName, args...)
	tempHome, clean, err := setupAntigravityHome(a.Tools)
	if err != nil {
		return cmd, nil
	}
	cmd.Env = append(os.Environ(), "HOME="+tempHome)
	return cmd, clean
}

// getCarriedSession is the session an earlier task of the conversation was
// answered in, and nothing where there is none.
func (a *Agent) getCarriedSession(conversation string) string {
	a.carried.Lock()
	defer a.carried.Unlock()
	if conversation == "" {
		return ""
	}
	return a.carried.sessions[conversation]
}

// makeCodexCommand is the codex program asked the task, reaching the vault's
// tools by its configuration flags.
func (a *Agent) makeCodexCommand(
	running context.Context, binName string, baseArgs []string, task port.Task,
) *exec.Cmd {
	args := append([]string{}, baseArgs...)
	mcpURLConfig := fmt.Sprintf(`mcp_servers.numen.url=%q`, a.Tools.URL)
	mcpHeaderConfig := fmt.Sprintf(`mcp_servers.numen.http_headers={"Authorization"="Bearer %s"}`, a.Tools.Token)
	args = append(args,
		"exec",
		"--json",
		"--dangerously-bypass-approvals-and-sandbox",
		"-c", mcpURLConfig,
		"-c", mcpHeaderConfig,
	)
	if model := a.getModel(task); model != "" && model != "default" {
		args = append(args, "--model", model)
	}
	args = append(args, task.Question)
	return exec.CommandContext(running, binName, args...)
}

// read talks to the agent and reports what it does as steps until its output
// ends, then waits for the process and lets go of everything the work holds.
func (a *Agent) read(w *work, stdin io.WriteCloser, stdout io.ReadCloser, task port.Task, isRPC bool) {
	defer func() {
		if w.cleanup != nil {
			w.cleanup()
		}
	}()
	defer w.reader.Done()
	defer a.unregisterWork(w)
	defer close(w.steps)
	if stdin != nil {
		defer stdin.Close()
	}
	defer stdout.Close()

	if isRPC && stdin != nil {
		a.sendRequests(stdin, task)
	}

	events := &eventReader{agent: a, work: w, conversation: task.Conversation}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var rawMap map[string]any
		if err := json.Unmarshal(line, &rawMap); err == nil {
			events.handle(rawMap)
		}
	}

	_ = w.cmd.Wait()
}

// sendRequests opens a session on the agent over JSON-RPC and prompts it with
// the task.
func (a *Agent) sendRequests(stdin io.Writer, task port.Task) {
	writeRequest(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-08-01",
			"clientInfo":      map[string]any{"name": "Numen", "version": "1.0.0"},
			"capabilities":    map[string]any{"mcp": true, "sessionPersistence": true},
		},
	})
	writeRequest(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "session/new",
		"params": map[string]any{
			"cwd": a.Root,
			"mcpServers": []map[string]any{
				{
					"name": "numen",
					"transport": map[string]any{
						"type": "http",
						"url":  a.Tools.URL,
						"headers": map[string]string{
							"Authorization": "Bearer " + a.Tools.Token,
						},
					},
				},
			},
		},
	})
	writeRequest(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "session/prompt",
		"params": map[string]any{
			"model": a.Model,
			"prompt": []map[string]any{
				{"role": "user", "text": task.Question},
			},
			"context": map[string]any{
				"focus": task.Focus,
			},
		},
	})
}

// writeRequest writes one request as a line of JSON.
func writeRequest(stdin io.Writer, request map[string]any) {
	if raw, err := json.Marshal(request); err == nil {
		raw = append(raw, '\n')
		_, _ = stdin.Write(raw)
	}
}
