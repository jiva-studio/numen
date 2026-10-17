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

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/claudecode/proofread"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/runner"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Proofreader puts a recognition right through the command line a person already
// has installed. It is a child process and nothing more: a prompt on its input,
// the lines it would put right on its output.
type Proofreader struct {
	// Command starts it. Empty means `claude` from the path.
	Command []string
	// Model is which model corrects a recognition, by the name the command line
	// knows it as. Empty leaves the choice to the installation.
	Model string
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

// Proofread asks the command line about the lines of a request and answers with
// the corrections its replies make.
//
// One prompt that fails ends the run: what was already answered is dropped and
// the caller asks again.
func (p *Proofreader) Proofread(ctx context.Context, request domain.ProofreadRequest) (domain.ProofreadAnswer, error) {
	if len(request.Groups) == 0 {
		return domain.ProofreadAnswer{IsDone: true}, nil
	}

	// The command line reads whatever standing instructions and settings live
	// where it was started, and those would land in the proofreading prompt.
	// An empty folder holds none.
	empty, err := os.MkdirTemp("", "numen-proofreading-")
	if err != nil {
		return domain.ProofreadAnswer{}, err
	}
	defer os.RemoveAll(empty)

	ask := func(ctx context.Context, instruction, prompt string) (string, error) {
		turn, err := p.acquire(ctx)
		if err != nil {
			return "", err
		}
		defer turn()
		return p.runPrompt(ctx, empty, instruction, prompt)
	}
	switch request.Kind {
	case domain.ProofreadScan:
		return proofread.Scan(ctx, ask, request.Groups, request.MaxEditDistance)
	case domain.ProofreadSpeech:
		return proofread.Speech(ctx, ask, request)
	}
	return domain.ProofreadAnswer{}, errUnknownKind
}

var errUnknownKind = errors.New("unknown kind of text to proofread")

// runPrompt sends one prompt and returns what the command line said about it.
func (p *Proofreader) runPrompt(ctx context.Context, dir, instruction, prompt string) (string, error) {
	name, rest := p.getCommand()
	cmd := exec.CommandContext(ctx, name, append(rest, p.buildArguments(instruction)...)...)
	cmd.Dir = dir
	cmd.Env = buildEnvironment(os.Environ())
	runner.Supervise(cmd)

	// The prompt goes on the input: --disallowed-tools takes as many names as
	// follow it, and a prompt is longer than a command line holds.
	cmd.Stdin = strings.NewReader(prompt)
	var said, stderr bytes.Buffer
	cmd.Stdout = &said
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, getLastLine(stderr.String()))
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
// servers, no tools, and the instruction appended to what the command
// line says of itself.
func (p *Proofreader) buildArguments(instruction string) []string {
	args := []string{"-p", "--strict-mcp-config"}
	if p.Model != "" {
		args = append(args, "--model", p.Model)
	}
	args = append(args, "--disallowed-tools")
	args = append(args, disallowed...)
	return append(args, "--append-system-prompt", instruction)
}
