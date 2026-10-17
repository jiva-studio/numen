// Package claudecode puts a recognition right through the command line a person already has
// installed, one batch to a process.
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

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/runner"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Proofreader puts a recognition right through the command line a person already
// has installed. It is a child process and nothing more: a batch on its input,
// the lines it would put right on its output.
type Proofreader struct {
	// Command starts it. Empty means `claude` from the path.
	Command []string
	// Model is which model corrects a recognition, by the name the command line
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
	once      sync.Once
	semaphore chan struct{}
}

// acquire waits for a turn at the command line, and hands back what gives it up.
func (p *Proofreader) acquire(ctx context.Context) (func(), error) {
	p.once.Do(func() {
		p.semaphore = make(chan struct{}, max(p.InFlight, 1))
	})
	select {
	case p.semaphore <- struct{}{}:
		return func() { <-p.semaphore }, nil
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

// GetName is what put a correction right, recorded beside every line it made.
func (p *Proofreader) GetName() string {
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
func (p *Proofreader) Proofread(ctx context.Context, batches []domain.ProofreadBatch) (map[int]string, error) {
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
		turn, err := p.acquire(ctx)
		if err != nil {
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer turn()
			reply, err := p.runBatch(ctx, empty, batch)
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

// runBatch sends one batch and returns what the command line said about it.
func (p *Proofreader) runBatch(ctx context.Context, dir string, batch domain.ProofreadBatch) (string, error) {
	name, rest := p.getCommand()
	cmd := exec.CommandContext(ctx, name, append(rest, p.buildArguments()...)...)
	cmd.Dir = dir
	cmd.Env = buildEnvironment(os.Environ())
	runner.Supervise(cmd)

	// The batch goes on the input: --disallowed-tools takes as many names as
	// follow it, and a batch is longer than a command line holds.
	cmd.Stdin = strings.NewReader(batch.Prompt)
	var said, stderr bytes.Buffer
	cmd.Stdout = &said
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("batch %d: %w: %s", batch.Number, err, getLastLine(stderr.String()))
	}
	return strings.TrimSpace(said.String()), nil
}

// getCommand returns the command line to run and what stands before its own arguments.
func (p *Proofreader) getCommand() (string, []string) {
	if len(p.Command) == 0 {
		return agent.FindClaude(), nil
	}
	return p.Command[0], p.Command[1:]
}

// buildArguments returns what the command line is started with: one question, no
// servers, no tools, and the caller's instruction appended to what the command
// line says of itself.
func (p *Proofreader) buildArguments() []string {
	args := []string{"-p", "--strict-mcp-config"}
	if p.Model != "" {
		args = append(args, "--model", p.Model)
	}
	args = append(args, "--disallowed-tools")
	args = append(args, disallowed...)
	return append(args, "--append-system-prompt", p.Instruction)
}

// getLastLine is the part of the error output worth showing: what a command line says last is what
// went wrong.
func getLastLine(said string) string {
	lines := strings.Split(strings.TrimSpace(said), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}
