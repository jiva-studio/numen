package completion

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

const (
	commandWaitDelay = 2 * time.Second
	maxOutputBytes   = 4 << 20
)

// ErrNoCommand means no command was named to run.
var ErrNoCommand = errors.New("completion: no command to run")

// ErrOutputTooLarge means the command wrote more than the output may hold.
var ErrOutputTooLarge = errors.New("completion: the command wrote too much output")

// RunCommand runs the command with stdin as its input and returns what it wrote to stdout.
// A command that fails is reported by the last line it wrote to stderr, and one that writes more
// than 4 MiB to either stream is stopped.
func RunCommand(ctx context.Context, name string, args, env []string, stdin string) (string, error) {
	if name == "" {
		return "", ErrNoCommand
	}
	running, stop := context.WithCancel(ctx)
	defer stop()
	out, said := &limitedBuffer{stop: stop}, &limitedBuffer{stop: stop}
	cmd := exec.CommandContext(running, name, args...)
	cmd.Env = env
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Stdout = out
	cmd.Stderr = said
	cmd.WaitDelay = commandWaitDelay
	if err := cmd.Run(); err != nil {
		if out.exceeded || said.exceeded {
			return "", ErrOutputTooLarge
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		return "", errors.New(FailureMessage(err, said.buf.String()))
	}
	return strings.TrimSpace(out.buf.String()), nil
}

// limitedBuffer keeps what is written to it and calls stop once the written bytes pass the limit.
type limitedBuffer struct {
	buf      bytes.Buffer
	exceeded bool
	stop     func()
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	if l.buf.Len()+len(p) > maxOutputBytes {
		l.exceeded = true
		l.stop()
		return 0, ErrOutputTooLarge
	}
	return l.buf.Write(p)
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

// ListClaudeModels returns the model aliases the Claude Code CLI takes after --model. None is
// marked default: without --model the CLI answers with the model its installation names.
func ListClaudeModels() []*numenv1.CompletionModel {
	return []*numenv1.CompletionModel{
		{Name: "sonnet", Title: "Sonnet"},
		{Name: "opus", Title: "Opus"},
		{Name: "haiku", Title: "Haiku"},
	}
}
