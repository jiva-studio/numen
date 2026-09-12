package claudecode

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// Proofreader puts a reading right through the command line a person already
// has installed. It is a child process and nothing more: a batch on its input,
// the lines it would put right on its output.
type Proofreader struct {
	// Command starts it. Empty means `claude` from the path.
	Command []string
	// Model is which model corrects a reading, by the name the command line
	// knows it as. Empty leaves the choice to the installation.
	Model string
	// Instruction is what the proofreader is told it is doing. A scan and
	// speech are corrected for different mistakes, and the caller says which.
	Instruction string

	// InFlight is how many batches are asked about at once. It is how much of
	// a person's own model is taken while they are using it. Zero or less
	// takes one.
	InFlight int

	// turns is how many runs may stand at once, taken before one starts and
	// given back after it ends.
	once  sync.Once
	turns chan struct{}
}

// take waits for a turn at the command line, and hands back what gives it up.
func (p *Proofreader) take(ctx context.Context) (func(), error) {
	p.once.Do(func() {
		p.turns = make(chan struct{}, max(p.InFlight, 1))
	})
	select {
	case p.turns <- struct{}{}:
		return func() { <-p.turns }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// disallowed are the tools the command line is started without. It reads one
// batch and writes one answer, and touches neither this machine nor the web.
var disallowed = []string{
	"Write", "Edit", "Bash", "Read", "WebFetch", "WebSearch",
	"Glob", "Grep", "NotebookEdit", "Task", "TodoWrite",
}

// Name is what put a correction right, recorded beside every line it made.
func (p *Proofreader) Name() string {
	if p.Model == "" {
		return "claude"
	}
	return "claude " + p.Model
}

// Proofread asks about every batch and answers with what came back about each,
// by the batch it is about. A batch nothing came back about is left out.
//
// One batch that fails ends the run: the batches already answered are dropped
// and the caller asks again.
func (p *Proofreader) Proofread(ctx context.Context, batches []proofread.Batch) (map[int]string, error) {
	if len(batches) == 0 {
		return nil, nil
	}
	if p.Instruction == "" {
		return nil, errors.New("no instruction for the proofreading command line")
	}

	// The command line reads whatever standing instructions and settings live
	// where it was started, and those would land in the proofreading prompt.
	// An empty folder holds none.
	empty, err := os.MkdirTemp("", "numen-proofreading-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(empty)

	ctx, stop := context.WithCancel(ctx)
	defer stop()

	var (
		mu     sync.Mutex
		out    = make(map[int]string, len(batches))
		failed error
		wg     sync.WaitGroup
	)
	for _, batch := range batches {
		turn, err := p.take(ctx)
		if err != nil {
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer turn()
			reply, err := p.ask(ctx, empty, batch)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err != nil:
				if failed == nil {
					failed = err
					stop()
				}
			case reply != "":
				out[batch.Number] = reply
			}
		}()
	}
	wg.Wait()

	if failed != nil {
		return nil, failed
	}
	return out, ctx.Err()
}

// ask sends one batch and returns what the command line said about it.
func (p *Proofreader) ask(ctx context.Context, dir string, batch proofread.Batch) (string, error) {
	name, rest := p.starts()
	cmd := exec.CommandContext(ctx, name, append(rest, p.arguments()...)...)
	cmd.Dir = dir
	cmd.Env = environment(os.Environ())
	detach(cmd)

	// The whole group goes, and the pipes are let go of shortly after: a
	// grandchild holding the child's output keeps a wait from returning.
	cmd.Cancel = func() error { return kill(cmd) }
	cmd.WaitDelay = 2 * time.Second

	// The batch goes on the input: --disallowed-tools takes as many names as
	// follow it, and a batch is longer than a command line holds.
	cmd.Stdin = strings.NewReader(proofread.Ask(batch))
	var said, stderr bytes.Buffer
	cmd.Stdout = &said
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("batch %d: %w: %s", batch.Number, err, lastLine(stderr.String()))
	}
	return strings.TrimSpace(said.String()), nil
}

// starts is the command line to run and what stands before its own arguments.
func (p *Proofreader) starts() (string, []string) {
	if len(p.Command) == 0 {
		return findCommand(), nil
	}
	return p.Command[0], p.Command[1:]
}

// arguments are what the command line is started with: one question, no
// servers, no tools, and the caller's instruction appended to what the command
// line says of itself.
func (p *Proofreader) arguments() []string {
	args := []string{"-p", "--strict-mcp-config"}
	if p.Model != "" {
		args = append(args, "--model", p.Model)
	}
	args = append(args, "--disallowed-tools")
	args = append(args, disallowed...)
	return append(args, "--append-system-prompt", p.Instruction)
}
