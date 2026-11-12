package process

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/acp"
)

const (
	// maxLineBytes is the longest message an agent may send on one line.
	maxLineBytes = 4 << 20
	// stderrTailBytes is how much of the agent's error output is retained for its failure message.
	stderrTailBytes = 8 << 10
)

// Launcher starts agent programs as child processes.
type Launcher struct{}

var _ acp.Launcher = Launcher{}

// Child is an agent program started for one session, with a folder of its own as its working
// directory.
type Child struct {
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	lines    *bufio.Scanner
	stderr   *tailBuffer
	stop     context.CancelFunc
	cleanDir func() error
	dir      string
	program  string
}

// Launch starts the command in a new folder, in a process group of its own, and ends the process
// with ctx.
func (Launcher) Launch(ctx context.Context, program string, command []string) (acp.Process, error) {
	if len(command) == 0 {
		return nil, fmt.Errorf("no command starts %s", program)
	}
	dir, cleanDir, err := agentrun.CreateTempDir("numen-acp-*")
	if err != nil {
		return nil, fmt.Errorf("create the session folder: %w", err)
	}
	running, stop := context.WithCancel(ctx)
	child := &Child{
		cmd:      exec.CommandContext(running, command[0], command[1:]...),
		stdin:    nil,
		lines:    nil,
		stderr:   &tailBuffer{mu: sync.Mutex{}, buf: nil},
		stop:     stop,
		cleanDir: cleanDir,
		dir:      dir,
		program:  program,
	}
	child.cmd.Dir = dir
	// gomutants:disable-next-line STATEMENT_REMOVE reason="exec.Cmd uses os.Environ when Env is nil."
	child.cmd.Env = os.Environ()
	child.cmd.Stderr = child.stderr
	Supervise(child.cmd)
	if err := child.start(); err != nil {
		return nil, errors.Join(err, child.Close())
	}
	return child, nil
}

func (c *Child) start() error {
	stdin, err := c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open the input of %s: %w", c.program, err)
	}
	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open the output of %s: %w", c.program, err)
	}
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", c.program, err)
	}
	c.stdin = stdin
	c.lines = bufio.NewScanner(stdout)
	c.lines.Buffer(nil, maxLineBytes)
	return nil
}

// Dir returns the working directory of the program.
func (c *Child) Dir() string { return c.dir }

// Send writes one line to the input of the program.
func (c *Child) Send(line []byte) error {
	_, err := c.stdin.Write(append(line, '\n'))
	return err
}

// Receive returns the next line of the output of the program, valid until the next call, and
// io.EOF when the output has ended.
func (c *Child) Receive() ([]byte, error) {
	if c.lines.Scan() {
		return c.lines.Bytes(), nil
	}
	if err := c.lines.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

// Stderr returns the last bytes the program wrote to its error output.
func (c *Child) Stderr() string { return c.stderr.String() }

// Close ends the input of the program, waits for it to exit, kills it when it does not, and removes
// its folder.
func (c *Child) Close() error {
	var err error
	if c.stdin != nil {
		err = c.stdin.Close()
	}
	if c.cmd.Process != nil {
		if waitErr := WaitForExit(c.cmd, c.stop); !isStopped(waitErr) {
			err = errors.Join(err, waitErr)
		}
	}
	c.stop()
	return errors.Join(err, c.cleanDir())
}

func isStopped(err error) bool {
	var exit *exec.ExitError
	return err == nil || errors.As(err, &exit) || errors.Is(err, context.Canceled) ||
		errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH)
}

// tailBuffer retains the last bytes written to it.
type tailBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if overVal := len(t.buf) - stderrTailBytes; overVal > 0 {
		t.buf = t.buf[overVal:]
	}
	return len(p), nil
}

func (t *tailBuffer) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.buf)
}
