// Package task runs the Codex program: a prompt turn of acp.task, with the tools of the host served to
// the program, and a single answer with no tool for llm.complete.
package task

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	contract "github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/codexevent"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/domain/settings"
)

const (
	// listLimit is how long the host has to list the tools of a task.
	listLimit = 10 * time.Second
	// maxLine is the longest line of output the program may write.
	maxLine = 4 << 20
)

var (
	errClosed        = errors.New("the plugin is stopping")
	errNotInstalled  = errors.New("codex is not installed")
	errNoOutput      = errors.New("the program ended without output")
	errEmptyResponse = errors.New("the program ended the answer with no text")
)

// Dependencies are what a Runner needs. Without Tools, a task is served no tool.
type Dependencies struct {
	Settings   *settings.Settings
	Finder     Finder
	Commands   Commands
	Launcher   Launcher
	Tools      Tools
	ToolServer ToolServer
}

// Runner starts the program for each task and answer. The tasks of a Runner run in a folder of its
// own, which Close removes.
type Runner struct {
	deps Dependencies

	mu       sync.Mutex
	running  sync.WaitGroup
	cancels  map[uint64]context.CancelFunc
	nextRun  uint64
	isClosed bool
	dir      string
}

// New creates a Runner, and refuses dependencies that are missing.
func New(deps Dependencies) (*Runner, error) {
	if deps.Settings == nil || deps.Finder == nil || deps.Commands == nil || deps.Launcher == nil || deps.ToolServer == nil {
		return nil, errors.New("task: every dependency but the tools is required")
	}
	return &Runner{
		deps: deps, mu: sync.Mutex{}, running: sync.WaitGroup{}, cancels: map[uint64]context.CancelFunc{}, nextRun: 0,
		isClosed: false, dir: "",
	}, nil
}

var _ contract.Runner = (*Runner)(nil)

// FindProgram returns the program: the executable path of the settings when it is runnable, else the
// program found on the machine, else none.
func (r *Runner) FindProgram() string {
	return r.deps.Finder.Find(r.deps.Settings.ExecutablePath())
}

// RunTask runs the program on one prompt turn and sends what it reports as frames. The program may call
// the tools of the host the plugin lists; without them it answers with no tool, in a read-only sandbox.
// Each turn is a run of its own. A program that is not installed is a failure to try the next
// connection after. A run that fails sends what the program said of the failure as a message of its
// own and stops. A canceled task returns the error of ctx.
func (r *Runner) RunTask(
	ctx context.Context, call connection.Call, request *contract.TaskRequest, send func(*contract.TaskFrame) error,
) (done *contract.TaskDone, err error) {
	program := r.FindProgram()
	if program == "" {
		return nil, connection.NewTryNextError(connection.ReasonUnavailable, 0)
	}
	runCtx, end, err := r.begin(ctx)
	if err != nil {
		return nil, connection.NewStopError(connection.ReasonUnavailable)
	}
	defer end()

	listed, err := r.listTools(runCtx)
	if err != nil {
		return nil, readFailure(ctx, connection.NewStopError(connection.ReasonUnavailable))
	}
	spec := Spec{
		Program: program, Prompt: writePrompt(writeInstructions(listed), request),
		Model: call.Options[llm.OptionModel], Reasoning: call.Options[llm.OptionReasoning], ToolsURL: "",
	}
	cmd, served, err := r.makeTaskCommand(runCtx, spec, listed, request)
	if err != nil {
		return nil, readFailure(ctx, connection.NewStopError(connection.ReasonUnavailable))
	}
	defer func() {
		if served == nil {
			return
		}
		if closeErr := served.Close(context.WithoutCancel(ctx)); closeErr != nil && err == nil {
			done, err = nil, connection.NewStopError(connection.ReasonUnavailable)
		}
	}()
	frames := newTurn(indexTools(listed), send)
	reason, err := r.runProgram(runCtx, cmd, frames)
	return endTask(ctx, runCtx, frames, reason, err)
}

// endTask returns how a run that ended with reason and err ends the task: the error of ctx, of the
// frame that could not be sent or of the run, a stop for a run the plugin ended, the end of the turn,
// or the reason the run failed sent as a message of its own and a stop.
func endTask(ctx, runCtx context.Context, frames *turn, reason string, err error) (*contract.TaskDone, error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if frames.sendErr != nil {
		return nil, frames.sendErr
	}
	switch {
	case err != nil:
		return nil, readFailure(ctx, err)
	case runCtx.Err() != nil:
		return nil, connection.NewStopError(connection.ReasonUnavailable)
	case reason == "":
		return &contract.TaskDone{StopReason: contract.StopReason_STOP_REASON_END_TURN}, nil
	}
	if err := frames.SendMessage(reason); err != nil {
		return nil, err
	}
	return nil, connection.NewStopError(connection.ReasonUnavailable)
}

func (r *Runner) makeTaskCommand(
	ctx context.Context, spec Spec, listed []*tools.ToolInfo, request *contract.TaskRequest,
) (Command, ServedTools, error) {
	if len(listed) == 0 {
		cmd, err := r.deps.Commands.MakeAnswerCommand(spec)
		return cmd, nil, err
	}
	dir, err := r.getDir()
	if err != nil {
		return Command{}, nil, err
	}
	toolContext, vaultID := maps.Clone(request.GetToolContext()), request.GetVaultId()
	served, err := r.deps.ToolServer.Serve(ctx, listed, func(ctx context.Context, name string, arguments []byte) (*tools.CallResponse, error) {
		return r.deps.Tools.InvokeTool(ctx, tools.Call{Name: name, Arguments: arguments, Context: toolContext, VaultID: vaultID})
	})
	if err != nil {
		return Command{}, nil, err
	}
	spec.ToolsURL = served.GetURL()
	cmd := r.deps.Commands.MakeToolsCommand(spec)
	cmd.Dir, cmd.Token = dir, served.GetToken()
	return cmd, served, nil
}

// Answer runs the program once on the prompt, with the instructions before it, no tool, in a read-only
// sandbox, and returns the text of its answer. A run that fails returns what the program said of it.
func (r *Runner) Answer(ctx context.Context, call connection.Call, instructions, prompt string) (string, error) {
	program := r.FindProgram()
	if program == "" {
		return "", errNotInstalled
	}
	runCtx, end, err := r.begin(ctx)
	if err != nil {
		return "", err
	}
	defer end()
	cmd, err := r.deps.Commands.MakeAnswerCommand(Spec{
		Program: program, Prompt: joinParagraphs(instructions, prompt),
		Model: call.Options[llm.OptionModel], Reasoning: call.Options[llm.OptionReasoning], ToolsURL: "",
	})
	if err != nil {
		return "", err
	}
	answer := &collector{text: strings.Builder{}}
	reason, err := r.runProgram(runCtx, cmd, answer)
	switch {
	case err != nil:
		return "", err
	case runCtx.Err() != nil:
		return "", runCtx.Err()
	case reason != "":
		return "", errors.New(reason)
	case strings.TrimSpace(answer.text.String()) == "":
		return "", errEmptyResponse
	}
	return answer.text.String(), nil
}

// runProgram starts the command, hands what it writes to sink and returns the reason the run failed:
// what the program wrote of an exit that failed, or of a run that wrote nothing else, else none. A
// line that is no event is text. A run whose context ends returns no reason. The command is released
// when the run ends.
func (r *Runner) runProgram(ctx context.Context, cmd Command, sink codexevent.Sink) (reason string, err error) {
	if cmd.Release != nil {
		defer func() { err = errors.Join(err, cmd.Release()) }()
	}
	proc, err := r.deps.Launcher.Start(ctx, cmd)
	if err != nil {
		return "", connection.NewTryNextError(connection.ReasonUnavailable, 0)
	}
	stdout := proc.Output()
	scanned := make(chan scanResult, 1)
	go func() { scanned <- scanOutput(stdout, sink) }()
	var result scanResult
	var closeErr error
	select {
	case result = <-scanned:
	case <-ctx.Done():
		closeErr = stdout.Close()
		result = <-scanned
	}
	waitErr := errors.Join(closeErr, proc.Wait())
	switch {
	case result.sendErr != nil:
		return "", result.sendErr
	case ctx.Err() != nil:
		return "", nil
	case result.scanErr != nil:
		return result.scanErr.Error(), nil
	case waitErr != nil:
		return getFailureMessage(waitErr, proc.Stderr()), nil
	case result.reports == 0 && strings.TrimSpace(proc.Stderr()) != "":
		return getFailureMessage(errNoOutput, proc.Stderr()), nil
	}
	return "", nil
}

// scanResult is what the output of a run held: how many reports it made, the error of the sink that
// refused one, and the error of reading it.
type scanResult struct {
	reports int
	sendErr error
	scanErr error
}

func scanOutput(stdout io.Reader, sink codexevent.Sink) scanResult {
	counted := &countingSink{sink: sink, count: 0}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(nil, maxLine)
	for scanner.Scan() {
		line := scanner.Bytes()
		var err error
		if event, isEvent := codexevent.DecodeLine(line); isEvent {
			err = codexevent.Handle(counted, event)
		} else if text := strings.TrimSpace(string(line)); text != "" {
			err = counted.SendText(text + "\n")
		}
		if err != nil {
			return scanResult{reports: counted.count, sendErr: err, scanErr: nil}
		}
	}
	return scanResult{reports: counted.count, sendErr: nil, scanErr: scanner.Err()}
}

// countingSink counts the reports it passes on.
type countingSink struct {
	sink  codexevent.Sink
	count int
}

func (s *countingSink) SendText(text string) error {
	s.count++
	return s.sink.SendText(text)
}

func (s *countingSink) SendToolCall(id, tool, arguments string) error {
	s.count++
	return s.sink.SendToolCall(id, tool, arguments)
}

func (s *countingSink) SendToolResult(id string, isFailed bool) error {
	s.count++
	return s.sink.SendToolResult(id, isFailed)
}

func (r *Runner) listTools(ctx context.Context) ([]*tools.ToolInfo, error) {
	if r.deps.Tools == nil {
		return nil, nil
	}
	listing, cancel := context.WithTimeout(ctx, listLimit)
	defer cancel()
	listed, err := r.deps.Tools.ListTools(listing)
	if err != nil {
		return nil, err
	}
	return listed.Tools, nil
}

// begin counts a run that Close waits for, and returns its context, which Close cancels.
func (r *Runner) begin(ctx context.Context) (context.Context, func(), error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.isClosed {
		return nil, nil, errClosed
	}
	r.running.Add(1)
	runCtx, cancel := context.WithCancel(ctx)
	r.nextRun++
	run := r.nextRun
	r.cancels[run] = cancel
	return runCtx, func() {
		r.mu.Lock()
		delete(r.cancels, run)
		r.mu.Unlock()
		cancel()
		r.running.Done()
	}, nil
}

func (r *Runner) getDir() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.dir == "" {
		dir, err := os.MkdirTemp("", "numen-codex-tasks-*")
		if err != nil {
			return "", fmt.Errorf("create the folder the program runs in: %w", err)
		}
		r.dir = dir
	}
	return r.dir, nil
}

// Close ends every run, waits for it, refuses further runs and removes the folder of the tasks.
func (r *Runner) Close() error {
	r.mu.Lock()
	if r.isClosed {
		r.mu.Unlock()
		return nil
	}
	r.isClosed = true
	for _, cancel := range r.cancels {
		cancel()
	}
	r.mu.Unlock()
	r.running.Wait()
	if r.dir == "" {
		return nil
	}
	if err := os.RemoveAll(r.dir); err != nil {
		return fmt.Errorf("remove the folder of the tasks: %w", err)
	}
	return nil
}

func readFailure(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if _, isTryNext := connection.IsTryNext(err); isTryNext || connection.IsStop(err) {
		return err
	}
	return connection.NewStopError(connection.ReasonUnavailable)
}

func indexTools(listed []*tools.ToolInfo) map[string]*tools.ToolInfo {
	byName := make(map[string]*tools.ToolInfo, len(listed))
	for _, one := range listed {
		byName[one.GetName()] = one
	}
	return byName
}

// writeInstructions joins the distinct instruction sentences of the tools, in the order of the listing.
func writeInstructions(listed []*tools.ToolInfo) string {
	var sentences []string
	for _, one := range listed {
		for _, sentence := range one.GetInstructions() {
			if !slices.Contains(sentences, sentence) {
				sentences = append(sentences, sentence)
			}
		}
	}
	return strings.Join(sentences, " ")
}

// writePrompt returns the instructions, the notes of the request and its prompt, a paragraph apart.
func writePrompt(instructions string, request *contract.TaskRequest) string {
	texts := make([]string, 0, len(request.GetContextNotes())+2)
	texts = append(texts, instructions)
	for _, note := range request.GetContextNotes() {
		heading := "The note " + note.GetPath()
		if title := note.GetTitle(); title != "" {
			heading += " (" + title + ")"
		}
		texts = append(texts, heading+":\n\n"+note.GetText())
	}
	return joinParagraphs(append(texts, request.GetPrompt())...)
}

// joinParagraphs joins the texts that are not blank with a blank line between them.
func joinParagraphs(texts ...string) string {
	kept := slices.DeleteFunc(slices.Clone(texts), func(text string) bool { return strings.TrimSpace(text) == "" })
	return strings.Join(kept, "\n\n")
}

// collector keeps the text of an answer.
type collector struct {
	text strings.Builder
}

func (c *collector) SendText(text string) error {
	c.text.WriteString(text)
	return nil
}

func (*collector) SendToolCall(string, string, string) error { return nil }

func (*collector) SendToolResult(string, bool) error { return nil }
