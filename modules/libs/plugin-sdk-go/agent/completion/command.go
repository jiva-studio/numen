package completion

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
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
	out, stderr := &limitedBuffer{stop: stop}, &limitedBuffer{stop: stop}
	cmd := exec.CommandContext(running, name, args...)
	cmd.Env = env
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Stdout = out
	cmd.Stderr = stderr
	cmd.WaitDelay = commandWaitDelay
	if err := cmd.Run(); err != nil {
		if out.exceeded || stderr.exceeded {
			return "", ErrOutputTooLarge
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		return "", errors.New(FailureMessage(err, stderr.buf.String()))
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
func FailureMessage(err error, stderr string) string {
	if trimmed := strings.TrimSpace(stderr); trimmed != "" {
		lines := strings.Split(trimmed, "\n")
		return strings.TrimSpace(lines[len(lines)-1])
	}
	return err.Error()
}
