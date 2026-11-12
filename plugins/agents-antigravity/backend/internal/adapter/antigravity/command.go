package antigravity

import (
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/model"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/usecase/task"
)

const (
	flagPrompt       = "-p"
	flagOutputFormat = "--output-format"
	streamJSON       = "stream-json"
)

// makeTaskCommand returns the command of a run that may call the tools served at the address of run,
// without asking, in a home of its own whose MCP configuration names them. A run with no address is an
// answer.
func makeTaskCommand(path string, run task.Run) (task.Command, error) {
	if run.ToolsURL == "" {
		return makeAnswerCommand(path, run)
	}
	args := appendRunFlags([]string{flagPrompt, run.Prompt, flagOutputFormat, streamJSON, "--dangerously-skip-permissions"}, run)
	return makeCommand(path, args, &endpoint{URL: run.ToolsURL, Token: run.ToolsToken})
}

// makeAnswerCommand returns the command of a run that only answers, in an empty folder and a home with
// no MCP server.
func makeAnswerCommand(path string, run task.Run) (task.Command, error) {
	args := appendRunFlags([]string{flagPrompt, run.Prompt, flagOutputFormat, streamJSON}, run)
	return makeCommand(path, args, nil)
}

func makeCommand(path string, args []string, tools *endpoint) (task.Command, error) {
	tempHome, cleanHome, err := setupAntigravityHome(tools)
	if err != nil {
		return task.Command{}, fmt.Errorf("set up the Antigravity home: %w", err)
	}
	dir, cleanDir, err := createTempDir("numen-antigravity-run-*")
	if err != nil {
		return task.Command{}, errors.Join(fmt.Errorf("create the folder to run in: %w", err), cleanHome())
	}
	return task.Command{
		Path: path, Args: args, Dir: dir, Env: append(hostcall.GetChildEnvironment(), "HOME="+tempHome),
		Release: func() error { return errors.Join(cleanHome(), cleanDir()) },
	}, nil
}

func appendRunFlags(args []string, run task.Run) []string {
	if run.Model != "" && run.Model != model.DefaultName {
		args = append(args, "--model", run.Model)
	}
	if run.Session != "" {
		args = append(args, "--conversation", run.Session)
	}
	return args
}
