// Package run answers a task by running the Claude Code program on it and reporting what the
// program does.
package run

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/toolcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/rpcerror"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/arguments"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/domain/environment"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/model"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/ports"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/stream"
)

// Clock gets the current time.
type Clock interface {
	Now() time.Time
}

// Commands finds the command line of the program for a task.
type Commands interface {
	// ResolveCommand returns the program and its leading arguments: the command the task names, else
	// the program found on the machine, which is none when it is not installed.
	ResolveCommand(command []string) (string, []string)
}

// Reasoning chooses the depth of reasoning of a run.
type Reasoning interface {
	GetReasoning(model, requested string) string
}

// Prompt gives the system prompt the host registered.
type Prompt interface {
	GetPrompt() string
}

// Dependencies are what a Run needs.
type Dependencies struct {
	Settings    *settings.Settings
	Prompt      Prompt
	Commands    Commands
	Reasoning   Reasoning
	Launcher    ports.Launcher
	ToolConfig  ports.ToolConfig
	Environment ports.Environment
	Clock       Clock
}

// Run starts the program for each task and keeps the runs that are answering, and the session of
// each conversation, until it is closed.
type Run struct {
	deps     Dependencies
	registry agentrun.Registry
}

// New creates a Run, and refuses dependencies that are missing.
func New(deps Dependencies) (*Run, error) {
	if deps.Settings == nil || deps.Prompt == nil || deps.Commands == nil || deps.Reasoning == nil ||
		deps.Launcher == nil || deps.ToolConfig == nil || deps.Environment == nil || deps.Clock == nil {
		return nil, errors.New("run: every dependency is required")
	}
	return &Run{deps: deps, registry: agentrun.Registry{}}, nil
}

// Close ends every run still answering and refuses further tasks.
func (r *Run) Close() error {
	return r.registry.Close()
}

// FinishConversation ends the runs of the conversation and forgets its session.
func (r *Run) FinishConversation(ctx context.Context, conversation string) error {
	return r.registry.FinishConversation(ctx, conversation)
}

func combineInstructions(instructions, systemPrompt string) string {
	if systemPrompt == "" {
		return instructions
	}
	if instructions != "" {
		return instructions + "\n" + systemPrompt
	}
	return systemPrompt
}

func (r *Run) prepareArguments(
	msg *agentrun.Request, instructions string, values settings.Values,
) (claudeArgs []string, configFile string, err error) {
	if len(msg.Tools) > 0 {
		configFile, err = r.deps.ToolConfig.Write(msg.ToolsURL, msg.ToolsToken)
		if err != nil {
			// gomutants:disable-next-line ERRORF_WRAP reason="rpcerror.New extracts only the error message string."
			return nil, "", rpcerror.New(rpcerror.Internal, fmt.Errorf("write mcp config: %w", err))
		}
	}
	modelName := msg.Model
	if modelName == "" {
		modelName = values.DefaultModel
	}
	level := r.deps.Reasoning.GetReasoning(modelName, msg.Reasoning)
	if modelName == model.DefaultName {
		modelName = ""
	}
	args := arguments.Build(arguments.Options{
		Instructions:  instructions,
		Model:         modelName,
		Reasoning:     level,
		Configuration: configFile,
		Session:       r.registry.GetSession(msg.Conversation),
		Turns:         int(msg.MaxSteps),
		Tools:         msg.Tools,
		MayReadHooks:  values.MayReadHooks,
	})
	return args, configFile, nil
}

// Ask runs the program on the task and sends each step it reports. The configuration file of the
// tools is removed when the run ends.
func (r *Run) Ask(ctx context.Context, msg *agentrun.Request, send func(*agentrun.Response) error) error {
	values := r.deps.Settings.Get()
	instructions := combineInstructions(msg.Instructions, r.deps.Prompt.GetPrompt())

	cmdName, cmdArgs := r.deps.Commands.ResolveCommand(msg.Command)
	if cmdName == "" {
		if err := send(&agentrun.Response{
			Step: &agentrun.TextStep{
				Text: "Claude Code CLI is not installed or not found on PATH.",
			},
		}); err != nil {
			return err
		}
		return send(&agentrun.Response{
			Step: &agentrun.StoppedStep{
				Stopped: "claude not installed",
			},
		})
	}

	claudeArgs, configFile, err := r.prepareArguments(msg, instructions, values)
	if err != nil {
		return err
	}

	runErr := r.streamProgram(ctx, send, msg, command{name: cmdName, args: cmdArgs, claudeArgs: claudeArgs, configFile: configFile})
	// gomutants:disable-next-line BRANCH_IF reason="agentrun.RemoveFile ignores non-existent paths and fails only on OS permission errors in the temp directory."
	if removeErr := r.deps.ToolConfig.Remove(configFile); removeErr != nil {
		return errors.Join(runErr, fmt.Errorf("remove the tool configuration: %w", removeErr))
	}
	return runErr
}

// command is the program to start for a run and the files it works with.
type command struct {
	name       string
	args       []string
	claudeArgs []string
	configFile string
}

func (r *Run) streamProgram(
	ctx context.Context,
	send func(*agentrun.Response) error,
	msg *agentrun.Request,
	cmd command,
) error {
	conversation := msg.Conversation
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// gomutants:disable-next-line INTEGER_INCREMENT reason="work.Output is unused because agent-claude writes responses directly to the stream"
	work := agentrun.NewWork(conversation, 0, agentrun.StopByCancel(cancel))
	if !r.registry.Add(work) {
		return rpcerror.New(rpcerror.Unavailable, errors.New("this agent is closing"))
	}
	defer r.registry.Remove(work)
	defer work.End()

	// The wait for the model begins with the program, and the stream opens on saying so.
	if err := send(&agentrun.Response{
		Step: &agentrun.ModelStartedStep{},
	}); err != nil {
		return err
	}

	proc, err := r.deps.Launcher.Launch(runCtx, ports.Launch{
		Command: cmd.name,
		Args:    slices.Concat(cmd.args, cmd.claudeArgs),
		Dir:     msg.Root,
		Env:     environment.DropSession(r.deps.Environment.GetVariables()),
		Input:   msg.Prompt,
	})
	if err != nil {
		return rpcerror.New(rpcerror.Internal, err)
	}

	recordSession := func(session string) { r.registry.SetSession(conversation, session) }
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Channel buffer capacity of 1 is a non-blocking send optimization before waitCompletion receives."
	reasons := make(chan string, 1)
	go func() {
		reasons <- stream.Parse(runCtx, proc.Output(), send, stream.Options{
			Words: toolcall.IndexDeclarations(msg.ToolDeclarations), Record: recordSession, Clock: r.deps.Clock, ExpectsVault: cmd.configFile != "",
		})
	}()

	stoppedReason, ok := waitCompletion(runCtx, proc, reasons)
	if !ok {
		return nil
	}

	return send(&agentrun.Response{
		Step: &agentrun.StoppedStep{
			Stopped: stoppedReason,
		},
	})
}

func waitCompletion(runCtx context.Context, proc ports.Process, reasons <-chan string) (string, bool) {
	var failedReason string
	var closeErr error
	select {
	case failedReason = <-reasons:
	case <-runCtx.Done():
		// gomutants:disable-next-line STATEMENT_REMOVE reason="Closing stdout on context cancellation unblocks the reader goroutine while runCtx.Err() returns early below."
		closeErr = proc.Output().Close()
		// gomutants:disable-next-line STATEMENT_REMOVE reason="Draining reasons waits for the reader goroutine while runCtx.Err() returns early below."
		failedReason = <-reasons
	}

	waitErr := errors.Join(closeErr, proc.Wait())
	if runCtx.Err() != nil {
		return "", false
	}

	if failedReason != "" {
		return failedReason, true
	}
	if waitErr != nil {
		return agentrun.FailureMessage(waitErr, proc.Stderr()), true
	}
	return "done", true
}
