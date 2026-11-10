package process

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/jiva-studio/numen/plugins/agents-claude/backend/ports"
)

// Launcher starts programs as processes of the machine, each supervised in a group of its own.
type Launcher struct{}

type program struct {
	cmd    *exec.Cmd
	out    io.ReadCloser
	stderr *strings.Builder
}

// Launch starts the program in the directory of the launch with its input on the standard input.
func (Launcher) Launch(ctx context.Context, launch ports.Launch) (ports.Process, error) {
	cmd := exec.CommandContext(ctx, launch.Command, launch.Args...)
	cmd.Dir = launch.Dir
	cmd.Env = launch.Env
	cmd.Stdin = strings.NewReader(launch.Input)
	Supervise(cmd)

	out, err := cmd.StdoutPipe()
	// gomutants:disable-next-line BRANCH_IF reason="cmd.StdoutPipe on a fresh command before Start fails only on OS descriptor exhaustion."
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start claude: %w", err)
	}
	return &program{cmd: cmd, out: out, stderr: &stderrBuf}, nil
}

func (p *program) Output() io.ReadCloser {
	return p.out
}

func (p *program) Stderr() string {
	return p.stderr.String()
}

func (p *program) Wait() error {
	return p.cmd.Wait()
}
