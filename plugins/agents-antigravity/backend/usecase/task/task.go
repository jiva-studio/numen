// Package task runs the Antigravity program: a prompt turn of acp.task, with the tools of the host
// served to the program, and a single answer with no tool for llm.complete.
package task

import (
	"bufio"
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	contract "github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/events"
)

const (
	// listLimit is how long the host has to list the tools of a task.
	listLimit = 10 * time.Second
	// maxLine is the longest line of output the program may write.
	maxLine = 4 << 20
	// stoppedDone is the reason the program gives for a run that finished.
	stoppedDone = "done"
)

var (
	errClosed        = errors.New("the plugin is stopping")
	errNotInstalled  = errors.New("antigravity is not installed")
	errNoOutput      = errors.New("the program ended without output")
	errEmptyResponse = errors.New("the program ended the answer with no text")
)

// Dependencies are what a Runner needs. Without Tools, a task is served no tool.
type Dependencies struct {
	Settings   *settings.Settings
	Program    Program
	Models     Models
	Launcher   Launcher
	Tools      Tools
	ToolServer ToolServer
}

// Runner starts the program for each task and answer, and keeps the session of each conversation.
type Runner struct {
	deps     Dependencies
	sessions sessions

	mu       sync.Mutex
	running  sync.WaitGroup
	closing  context.Context
	closeAll context.CancelFunc
	isClosed bool
}

// New creates a Runner, and refuses dependencies that are missing.
func New(deps Dependencies) (*Runner, error) {
	if deps.Settings == nil || deps.Program == nil || deps.Models == nil || deps.Launcher == nil || deps.ToolServer == nil {
		return nil, errors.New("task: every dependency but the tools is required")
	}
	closing, closeAll := context.WithCancel(context.Background())
	return &Runner{
		deps: deps, sessions: sessions{mu: sync.Mutex{}, byKey: nil, order: nil},
		mu: sync.Mutex{}, running: sync.WaitGroup{}, closing: closing, closeAll: closeAll, isClosed: false,
	}, nil
}

var _ contract.Runner = (*Runner)(nil)

// FindProgram returns the program to run, or none when it is not installed: the executable path of the
// settings when it is runnable, else the program found on the machine.
func (r *Runner) FindProgram() string {
	path := r.deps.Settings.ExecutablePath()
	if !r.deps.Program.IsInstalled(path) {
		return ""
	}
	return r.deps.Program.Find(path)
}

// RunTask runs the program on one prompt turn and sends what it reports as frames. The program may call
// the tools of the host the plugin lists without asking; the turns of one conversation of one connection
// continue one conversation of the program. A program that is not installed is a failure to try the next
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
	served, err := r.serveTools(runCtx, listed, request)
	if err != nil {
		return nil, readFailure(ctx, connection.NewStopError(connection.ReasonUnavailable))
	}
	run := Run{
		Prompt: writePrompt(listed, request), Model: r.chooseModel(runCtx, program, call),
		Session: "", ToolsURL: "", ToolsToken: "",
	}
	if served != nil {
		// The endpoint ends with the run; a call cut by its shutdown has already failed the run.
		defer func() { _ = served.Close(context.WithoutCancel(ctx)) }()
		run.ToolsURL, run.ToolsToken = served.GetURL(), served.GetToken()
	}
	key := ""
	if request.GetConversation() != "" {
		key = request.GetConversation() + "\x00" + call.ConnectionID
	}
	run.Session = r.sessions.get(key)
	command, err := r.deps.Program.MakeTaskCommand(program, run)
	if err != nil {
		return nil, readFailure(ctx, connection.NewStopError(connection.ReasonUnavailable))
	}
	frames := newTurn(indexTools(listed), send)
	reason, err := r.runProgram(runCtx, command, frames, func(session string) { r.sessions.set(key, session) })
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
		if err := frames.finishCalls(); err != nil {
			return nil, err
		}
		return &contract.TaskDone{StopReason: contract.StopReason_STOP_REASON_END_TURN}, nil
	}
	if err := frames.SendMessage(reason); err != nil {
		return nil, err
	}
	return nil, connection.NewStopError(connection.ReasonUnavailable)
}

// Answer runs the program once on the prompt, with no tool, in an empty folder, and returns the text of
// its answer. A run that fails returns what the program said of it.
func (r *Runner) Answer(ctx context.Context, call connection.Call, prompt string) (string, error) {
	program := r.FindProgram()
	if program == "" {
		return "", errNotInstalled
	}
	runCtx, end, err := r.begin(ctx)
	if err != nil {
		return "", err
	}
	defer end()
	command, err := r.deps.Program.MakeAnswerCommand(program, Run{
		Prompt: prompt, Model: r.chooseModel(runCtx, program, call), Session: "", ToolsURL: "", ToolsToken: "",
	})
	if err != nil {
		return "", err
	}
	answer := &collector{text: strings.Builder{}}
	reason, err := r.runProgram(runCtx, command, answer, nil)
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

func (r *Runner) chooseModel(ctx context.Context, program string, call connection.Call) string {
	return r.deps.Models.ResolveModel(ctx, program, call.Options[llm.OptionModel], call.Options[llm.OptionReasoning])
}

// output is where what a run reports goes.
type output interface {
	SendText(text string) error
	SendToolCall(tool, arguments string) error
}

// runSink hands the reports of a run to its output, and keeps what the run says of itself.
type runSink struct {
	output
	record     func(string)
	reason     string
	isFinished bool
}

func (s *runSink) SetSession(session string) {
	if s.record != nil && session != "" {
		s.record(session)
	}
}

func (s *runSink) Stop(reason string) {
	if s.reason == "" && reason != stoppedDone {
		s.reason = reason
	}
}

func (s *runSink) Finish() { s.isFinished = true }

// runProgram starts the command, hands what the program writes to out until it reports its result, and
// returns the reason the run failed: what the program reported, else what it wrote of an exit that
// failed, else none. A run whose context ends returns no reason.
func (r *Runner) runProgram(ctx context.Context, command Command, out output, record func(string)) (reason string, err error) {
	if command.Release != nil {
		defer func() { err = errors.Join(err, command.Release()) }()
	}
	runCtx, stop := context.WithCancel(ctx)
	defer stop()
	proc, err := r.deps.Launcher.Start(runCtx, command)
	if err != nil {
		return "", connection.NewTryNextError(connection.ReasonUnavailable, 0)
	}
	sink := &runSink{output: out, record: record, reason: "", isFinished: false}
	handler := events.NewHandler(sink)
	stdout := proc.Output()
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(nil, maxLine)
	isWritten := false
	for !sink.isFinished && handler.Err() == nil && scanner.Scan() {
		line := scanner.Bytes()
		if event, isEvent := events.DecodeLine(line); isEvent {
			isWritten = true
			handler.Handle(event)
		} else if text := strings.TrimSpace(string(line)); text != "" {
			isWritten = true
			handler.SendText(text + "\n")
		}
	}
	scanErr := scanner.Err()
	if sink.isFinished || handler.Err() != nil || scanErr != nil {
		stop()
	}
	waitErr := errors.Join(stdout.Close(), proc.Wait())
	switch {
	case handler.Err() != nil:
		return "", handler.Err()
	case ctx.Err() != nil:
		return "", nil
	case sink.reason != "":
		return sink.reason, nil
	case sink.isFinished:
		return "", nil
	case scanErr != nil:
		return scanErr.Error(), nil
	case waitErr != nil:
		return getFailureMessage(waitErr, proc.Stderr()), nil
	case !isWritten && strings.TrimSpace(proc.Stderr()) != "":
		return getFailureMessage(errNoOutput, proc.Stderr()), nil
	}
	return "", nil
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

func (r *Runner) serveTools(ctx context.Context, listed []*tools.ToolInfo, request *contract.TaskRequest) (ServedTools, error) {
	if len(listed) == 0 {
		return nil, nil
	}
	toolContext, vaultID := maps.Clone(request.GetToolContext()), request.GetVaultId()
	return r.deps.ToolServer.Serve(ctx, listed, func(ctx context.Context, name string, arguments []byte) (*tools.CallResponse, error) {
		return r.deps.Tools.InvokeTool(ctx, tools.Call{Name: name, Arguments: arguments, Context: toolContext, VaultID: vaultID})
	})
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
	stop := context.AfterFunc(r.closing, cancel)
	return runCtx, func() {
		stop()
		cancel()
		r.running.Done()
	}, nil
}

// Close ends every run, waits for it and refuses further runs.
func (r *Runner) Close() error {
	r.mu.Lock()
	if r.isClosed {
		r.mu.Unlock()
		return nil
	}
	r.isClosed = true
	r.closeAll()
	r.mu.Unlock()
	r.running.Wait()
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

// writePrompt joins the distinct instruction sentences of the tools, in the order of the listing, the
// notes of the task under their paths and titles, and the prompt.
func writePrompt(listed []*tools.ToolInfo, request *contract.TaskRequest) string {
	var sentences []string
	for _, one := range listed {
		for _, sentence := range one.GetInstructions() {
			if !slices.Contains(sentences, sentence) {
				sentences = append(sentences, sentence)
			}
		}
	}
	var texts []string
	if len(sentences) > 0 {
		texts = append(texts, strings.Join(sentences, " "))
	}
	for _, note := range request.GetContextNotes() {
		heading := "The note " + note.GetPath()
		if title := note.GetTitle(); title != "" {
			heading += " (" + title + ")"
		}
		texts = append(texts, heading+":\n\n"+note.GetText())
	}
	return strings.Join(append(texts, request.GetPrompt()), "\n\n")
}

// collector keeps the text of an answer.
type collector struct {
	text strings.Builder
}

func (c *collector) SendText(text string) error {
	c.text.WriteString(text)
	return nil
}

func (*collector) SendToolCall(string, string) error { return nil }
