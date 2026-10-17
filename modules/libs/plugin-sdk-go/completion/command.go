package completion

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

const commandWaitDelay = 2 * time.Second

// ErrNoCommand means no command was named to run.
var ErrNoCommand = errors.New("completion: no command to run")

// RunCommand runs the command with stdin as its input and returns what it wrote to stdout.
// A command that fails is reported by the last line it wrote to stderr.
func RunCommand(ctx context.Context, name string, args, env []string, stdin string) (string, error) {
	if name == "" {
		return "", ErrNoCommand
	}
	var out, said bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Stdout = &out
	cmd.Stderr = &said
	cmd.WaitDelay = commandWaitDelay
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		return "", errors.New(FailureMessage(err, said.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

// FailureMessage is what to tell the person when a command stopped with an error: the last line of
// its error output, or the error itself when it wrote none.
func FailureMessage(err error, said string) string {
	if trimmed := strings.TrimSpace(said); trimmed != "" {
		lines := strings.Split(trimmed, "\n")
		return strings.TrimSpace(lines[len(lines)-1])
	}
	return err.Error()
}

// ClaudeArguments are the arguments with which the Claude Code CLI gives one answer as plain text:
// no tools, no MCP servers, no settings and one turn.
func ClaudeArguments(system, model string) []string {
	args := []string{
		"-p",
		"--output-format", "text",
		"--permission-mode", "dontAsk",
		"--max-turns", "1",
		"--tools", "",
		"--strict-mcp-config",
		"--setting-sources", "",
	}
	if system != "" {
		args = append(args, "--append-system-prompt", system)
	}
	if model != "" {
		args = append(args, "--model", model)
	}
	return args
}
