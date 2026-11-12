// Package task runs the Claude Code program: a prompt turn of acp.task, with the tools of the host
// served to the program, and a single answer with no tool for llm.complete.
package task

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/arguments"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/domain/environment"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/model"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/ports"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/stream"
)

// listLimit is how long the host has to list the tools of a task.
const listLimit = 10 * time.Second

var (
	errClosed        = errors.New("the plugin is stopping")
	errNotInstalled  = errors.New("claude code is not installed")
	errEmptyResponse = errors.New("the program ended the answer with no text")
)

// Dependencies are what a Runner needs. Without Tools, a task is served no tool.
type Dependencies struct {
	Settings    *settings.Settings
	Finder      ports.ProgramFinder
	Reasoning   Reasoning
	Launcher    ports.Launcher
	ToolConfig  ports.ToolConfig
	Environment ports.Environment
	Tools       Tools
	ToolServer  ToolServer
}

// Runner starts the program for each task and answer, and keeps the session of each conversation.
// The runs of a Runner start in a folder of its own, which Close removes.
type Runner struct {
	deps     Dependencies
	sessions sessions

	mu       sync.Mutex
	running  sync.WaitGroup
	closing  context.Context
	closeAll context.CancelFunc
	isClosed bool
	dir      string
}

// New creates a Runner, and refuses dependencies that are missing.
func New(deps Dependencies) (*Runner, error) {
	if deps.Settings == nil || deps.Finder == nil || deps.Reasoning == nil || deps.Launcher == nil ||
		deps.ToolConfig == nil || deps.Environment == nil || deps.ToolServer == nil {
		return nil, errors.New("task: every dependency but the tools is required")
	}
	closing, closeAll := context.WithCancel(context.Background())
	return &Runner{
		deps: deps, sessions: sessions{mu: sync.Mutex{}, byKey: nil, order: nil},
		mu: sync.Mutex{}, running: sync.WaitGroup{}, closing: closing, closeAll: closeAll, isClosed: false, dir: "",
	}, nil
}

var _ contract.Runner = (*Runner)(nil)

// FindProgram returns the command of the program: the executable path of the settings when it is
// runnable, else the program found on the machine, else none.
func (r *Runner) FindProgram() string {
	return r.deps.Finder.Find(r.deps.Settings.Get().ExecutablePath)
}

// GetEnvironment returns the variables the program is started with.
func (r *Runner) GetEnvironment() []string {
	return environment.DropSession(r.deps.Environment.GetVariables())
}

// RunTask runs the program on one prompt turn and sends what it reports as frames. The program may
// call the tools of the host the plugin lists, and the web search of its own; the turns of one
// conversation of one connection continue one session of the program. A program that is not installed
// is a failure to try the next connection after. A run that fails sends what the program said of the
// failure as a message of its own and stops; a run that reached its limit of turns ends with that
// reason. A canceled task returns the error of ctx.
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
	served, configFile, err := r.serveTools(runCtx, listed, request)
	if err != nil {
		return nil, readFailure(ctx, connection.NewStopError(connection.ReasonUnavailable))
	}
	defer func() {
		if served != nil {
			// The endpoint ends with the run; a call cut by its shutdown has already failed the run.
			_ = served.Close(context.WithoutCancel(ctx))
		}
		_ = r.deps.ToolConfig.Remove(configFile)
	}()

	key := ""
	if request.GetConversation() != "" {
		key = request.GetConversation() + "\x00" + call.ConnectionID
	}
	values := r.deps.Settings.Get()
	modelName, level := r.chooseModel(call)
	args := arguments.Build(arguments.Options{
		Instructions: writeInstructions(listed), Model: modelName, Reasoning: level, Configuration: configFile,
		Session: r.sessions.get(key), Turns: 0, Tools: listNames(listed), MayReadHooks: values.MayReadHooks,
	})
	frames := newTurn(indexTools(listed), send)
	reason, err := r.runProgram(runCtx, program, args, writePrompt(request), frames, stream.Options{
		Record: func(session string) { r.sessions.set(key, session) }, ExpectsVault: configFile != "",
	})
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
	case reason == stream.ReasonMaxTurns:
		return &contract.TaskDone{StopReason: contract.StopReason_STOP_REASON_MAX_TURN_REQUESTS}, nil
	}
	if err := frames.SendMessage(reason); err != nil {
		return nil, err
	}
	return nil, connection.NewStopError(connection.ReasonUnavailable)
}

// Answer runs the program once on the prompt, with the instructions as the system prompt, no tool and
// one turn, and returns the text of its answer. A run that fails returns what the program said of it.
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
	modelName, level := r.chooseModel(call)
	args := arguments.Build(arguments.Options{
		Instructions: instructions, Model: modelName, Reasoning: level, Configuration: "",
		Session: "", Turns: 0, Tools: nil, MayReadHooks: false,
	})
	answer := &collector{text: strings.Builder{}}
	reason, err := r.runProgram(runCtx, program, args, prompt, answer, stream.Options{Record: nil, ExpectsVault: false})
	switch {
	case err != nil:
		return "", err
	case reason != "":
		return "", errors.New(reason)
	case strings.TrimSpace(answer.text.String()) == "":
		return "", errEmptyResponse
	}
	return answer.text.String(), nil
}

func (r *Runner) chooseModel(call connection.Call) (string, string) {
	modelName := call.Options[llm.OptionModel]
	level := r.deps.Reasoning.GetReasoning(modelName, call.Options[llm.OptionReasoning])
	if modelName == model.DefaultName {
		modelName = ""
	}
	return modelName, level
}

// runProgram starts the program, hands what it writes to sink and returns the reason the run failed:
// what the program reported, else what it wrote of an exit that failed, else none. A run whose context
// ends returns no reason.
func (r *Runner) runProgram(
	ctx context.Context, program string, args []string, input string, sink stream.Sink, opts stream.Options,
) (string, error) {
	dir, err := r.getDir()
	if err != nil {
		return "", err
	}
	proc, err := r.deps.Launcher.Launch(ctx, ports.Launch{
		Command: program, Args: args, Dir: dir, Env: r.GetEnvironment(), Input: input,
	})
	if err != nil {
		return "", connection.NewTryNextError(connection.ReasonUnavailable, 0)
	}
	type parsed struct {
		reason string
		err    error
	}
	results := make(chan parsed, 1)
	go func() {
		reason, err := stream.Parse(ctx, proc.Output(), sink, opts)
		results <- parsed{reason: reason, err: err}
	}()
	var result parsed
	var closeErr error
	select {
	case result = <-results:
	case <-ctx.Done():
		closeErr = proc.Output().Close()
		result = <-results
	}
	waitErr := errors.Join(closeErr, proc.Wait())
	switch {
	case result.err != nil:
		return "", result.err
	case ctx.Err() != nil:
		return "", nil
	case result.reason != "":
		return result.reason, nil
	case waitErr != nil:
		return getFailureMessage(waitErr, proc.Stderr()), nil
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

func (r *Runner) serveTools(ctx context.Context, listed []*tools.ToolInfo, request *contract.TaskRequest) (ServedTools, string, error) {
	if len(listed) == 0 {
		return nil, "", nil
	}
	toolContext, vaultID := maps.Clone(request.GetToolContext()), request.GetVaultId()
	served, err := r.deps.ToolServer.Serve(ctx, listed, func(ctx context.Context, name string, arguments []byte) (*tools.CallResponse, error) {
		return r.deps.Tools.InvokeTool(ctx, tools.Call{Name: name, Arguments: arguments, Context: toolContext, VaultID: vaultID})
	})
	if err != nil {
		return nil, "", err
	}
	configFile, err := r.deps.ToolConfig.Write(served.GetURL(), served.GetToken())
	if err != nil {
		return nil, "", errors.Join(err, served.Close(context.WithoutCancel(ctx)))
	}
	return served, configFile, nil
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

func (r *Runner) getDir() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.dir == "" {
		dir, err := os.MkdirTemp("", "numen-claude-*")
		if err != nil {
			return "", fmt.Errorf("create the folder the program runs in: %w", err)
		}
		r.dir = dir
	}
	return r.dir, nil
}

// Close ends every run, waits for it, refuses further runs and removes the folder of the runs.
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
	if r.dir == "" {
		return nil
	}
	if err := os.RemoveAll(r.dir); err != nil {
		return fmt.Errorf("remove the folder of the runs: %w", err)
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

func listNames(listed []*tools.ToolInfo) []string {
	names := make([]string, 0, len(listed))
	for _, one := range listed {
		names = append(names, one.GetName())
	}
	slices.Sort(names)
	return names
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

func writePrompt(request *contract.TaskRequest) string {
	texts := make([]string, 0, len(request.GetContextNotes())+1)
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

func (*collector) SendToolCall(string, string, string) error { return nil }

func (*collector) SendToolResult(string, bool) error { return nil }
