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
	TempDirs   TempDirs
}

// Runner starts the program for each task and answer, and keeps the thread of each conversation. The
// tasks of a Runner run in a folder of its own, which Close removes; Close also returns the failures to
// stop serving the tools of a task.
type Runner struct {
	deps    Dependencies
	threads sessions

	mu        sync.Mutex
	running   sync.WaitGroup
	cancels   map[uint64]context.CancelFunc
	nextRun   uint64
	isClosed  bool
	dir       string
	cleanErrs error
}

// New creates a Runner, and refuses dependencies that are missing.
func New(deps Dependencies) (*Runner, error) {
	if deps.Settings == nil || deps.Finder == nil || deps.Commands == nil || deps.Launcher == nil || deps.ToolServer == nil ||
		deps.TempDirs == nil {
		return nil, errors.New("task: every dependency but the tools is required")
	}
	return &Runner{
		deps: deps, threads: sessions{mu: sync.Mutex{}, byKey: nil, order: nil},
		mu: sync.Mutex{}, running: sync.WaitGroup{}, cancels: map[uint64]context.CancelFunc{}, nextRun: 0,
		isClosed: false, dir: "", cleanErrs: nil,
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
// The turns of one conversation of one connection continue one thread of the program; a thread the
// program no longer has, which it answers with a failure before it reports any thread or anything else,
// is started anew. A program that is not installed is a failure to try the next
// connection after. A run that fails sends what the program said of the failure as a message of its own
// and stops. A canceled task returns the error of ctx.
func (r *Runner) RunTask(
	ctx context.Context, call connection.Call, request *contract.TaskRequest, send func(*contract.TaskFrame) error,
) (*contract.TaskDone, error) {
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
	served, err := r.serveTools(runCtx, listed, request, call.ConnectionID)
	if err != nil {
		return nil, readFailure(ctx, connection.NewStopError(connection.ReasonUnavailable))
	}
	defer r.releaseTools(ctx, served)

	key := ""
	if request.GetConversation() != "" {
		key = request.GetConversation() + "\x00" + call.ConnectionID
	}
	spec := Spec{
		Program: program, Prompt: writePrompt(writeInstructions(listed), request),
		Model: call.Options[llm.OptionModel], Reasoning: call.Options[llm.OptionReasoning], ToolsURL: "",
		Thread: r.threads.get(key), IsKept: key != "",
	}
	frames := newTurn(indexTools(listed), send)
	reason, err := r.runTurn(runCtx, spec, served, frames)
	if spec.Thread != "" && isThreadGone(runCtx, frames, reason, err) {
		spec.Thread = ""
		reason, err = r.runTurn(runCtx, spec, served, frames)
	}
	r.threads.set(key, frames.thread)
	return endTask(ctx, runCtx, frames, reason, err)
}

func isThreadGone(runCtx context.Context, frames *turn, reason string, err error) bool {
	return err == nil && reason != "" && runCtx.Err() == nil && frames.thread == "" && !frames.isSent
}

func (r *Runner) runTurn(ctx context.Context, spec Spec, served ServedTools, frames *turn) (string, error) {
	cmd, err := r.makeTaskCommand(spec, served)
	if err != nil {
		return "", connection.NewStopError(connection.ReasonUnavailable)
	}
	return r.runProgram(ctx, cmd, frames)
}

func (r *Runner) releaseTools(ctx context.Context, served ServedTools) {
	if served == nil {
		return
	}
	if err := served.Close(context.WithoutCancel(ctx)); err != nil {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.cleanErrs = errors.Join(r.cleanErrs, err)
	}
}

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

func (r *Runner) serveTools(ctx context.Context, listed []*tools.ToolInfo, request *contract.TaskRequest, connectionID string) (ServedTools, error) {
	if len(listed) == 0 {
		return nil, nil
	}
	toolContext, vaultID := maps.Clone(request.GetToolContext()), request.GetVaultId()
	return r.deps.ToolServer.Serve(ctx, listed, func(ctx context.Context, name string, arguments []byte) (*tools.CallResponse, error) {
		return r.deps.Tools.InvokeTool(ctx, tools.Call{Name: name, Arguments: arguments, Context: toolContext, VaultID: vaultID, ConnectionID: connectionID})
	})
}

func (r *Runner) makeTaskCommand(spec Spec, served ServedTools) (Command, error) {
	if served == nil {
		return r.deps.Commands.MakeAnswerCommand(spec)
	}
	dir, err := r.getDir()
	if err != nil {
		return Command{}, err
	}
	spec.ToolsURL = served.GetURL()
	cmd := r.deps.Commands.MakeToolsCommand(spec)
	cmd.Dir, cmd.Token = dir, served.GetToken()
	return cmd, nil
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
		Thread: "", IsKept: false,
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

func (s *countingSink) RecordThread(id string) {
	s.sink.RecordThread(id)
}

func (s *countingSink) SendThought(text string) error {
	s.count++
	return s.sink.SendThought(text)
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
		dir, err := r.deps.TempDirs.Create()
		if err != nil {
			return "", fmt.Errorf("create the folder the program runs in: %w", err)
		}
		r.dir = dir
	}
	return r.dir, nil
}

// Close ends every run, waits for it, refuses further runs and removes the folder of the tasks. It returns
// the failures to stop serving the tools of the tasks.
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
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.dir == "" {
		return r.cleanErrs
	}
	if err := r.deps.TempDirs.Remove(r.dir); err != nil {
		return errors.Join(r.cleanErrs, fmt.Errorf("remove the folder of the tasks: %w", err))
	}
	return r.cleanErrs
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

func (*collector) RecordThread(string) {}

func (*collector) SendThought(string) error { return nil }

func (*collector) SendToolCall(string, string, string) error { return nil }

func (*collector) SendToolResult(string, bool) error { return nil }
