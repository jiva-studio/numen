// Package service is the backend of the Claude Code agent plugin.
package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/toolcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cache"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/rpcerror"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/internal/adapter/process"
)

// programName is the name tasks use for the program, and the settings section key.
const programName = "claude"

// readsHooksSetting is the setting of the plugin that lets the program read the hooks, skills and
// standing instructions of the person.
const readsHooksSetting = "reads_hooks_and_skills"

// programTitle is the display name of the program.
const programTitle = "Claude Code"

type agentServer struct {
	agentrun.Unimplemented
	mu             sync.RWMutex
	systemPrompt   string
	tools          []*agentrun.ToolDefinition
	registry       agentrun.Registry
	defaultModel   string
	executablePath string

	// mayReadHooks is whether the program reads the hooks, skills and standing instructions of the person.
	mayReadHooks bool

	// environ is the variables a program of the agent is started with; nil is those of this process.
	environ []string

	// models are the models the program listed.
	models *cache.Cache[[]claudeModel]

	// fetch lists the models of the program started as command with env; nil starts the program.
	fetch func(ctx context.Context, command, env []string) ([]claudeModel, error)

	// clock paces the reports of a change being written.
	clock timeSource
}

// getEnvironment returns the variables a program of the agent is started with.
func (s *agentServer) getEnvironment() []string {
	if s.environ != nil {
		return s.environ
	}
	return os.Environ()
}

// applyConfig keeps the model that answers a task naming none, the optional executable path and
// whether the program reads the hooks and skills of the person.
func (s *agentServer) applyConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if model, ok := values["model"]; ok && model != "" {
		s.defaultModel = model
	}
	if execPath, ok := values["executable_path"]; ok {
		s.executablePath = strings.TrimSpace(execPath)
	}
	if reads, ok := values[readsHooksSetting]; ok {
		s.mayReadHooks = reads == "true"
	}
}

func (s *agentServer) findClaude() string {
	s.mu.RLock()
	custom := s.executablePath
	s.mu.RUnlock()
	return findClaude(custom)
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

func (s *agentServer) prepareClaudeArgs(
	msg *agentrun.Request, instructions, defaultModel string,
) (claudeArgs []string, configFile string, err error) {
	s.mu.RLock()
	mayReadHooks := s.mayReadHooks
	s.mu.RUnlock()
	toolNames := msg.Tools
	if len(toolNames) > 0 {
		configFile, err = writeMcpConfig(msg.ToolsURL, msg.ToolsToken)
		if err != nil {
			// gomutants:disable-next-line ERRORF_WRAP reason="rpcerror.New extracts only the error message string."
			return nil, "", rpcerror.New(rpcerror.Internal, fmt.Errorf("write mcp config: %w", err))
		}
	}
	model := msg.Model
	if model == "" {
		model = defaultModel
	}
	level := s.getReasoning(model, msg.Reasoning)
	if model == defaultModelName {
		model = ""
	}
	args := buildClaudeArguments(claudeArguments{
		instructions:  instructions,
		model:         model,
		reasoning:     level,
		configuration: configFile,
		session:       s.registry.GetSession(msg.Conversation),
		turns:         int(msg.MaxSteps),
		tools:         toolNames,
		mayReadHooks:  mayReadHooks,
	})
	return args, configFile, nil
}

func (s *agentServer) Ask(ctx context.Context, msg *agentrun.Request, send func(*agentrun.Response) error) error {
	s.mu.RLock()
	systemPrompt := s.systemPrompt
	defaultModel := s.defaultModel
	s.mu.RUnlock()

	instructions := combineInstructions(msg.Instructions, systemPrompt)

	cmdName, cmdArgs := s.getCommand(msg.Command)
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

	claudeArgs, configFile, err := s.prepareClaudeArgs(msg, instructions, defaultModel)
	if err != nil {
		return err
	}

	runErr := s.streamClaude(ctx, send, msg, claudeCommand{name: cmdName, args: cmdArgs, claudeArgs: claudeArgs, configFile: configFile})
	// gomutants:disable-next-line BRANCH_IF reason="agentrun.RemoveFile ignores non-existent paths and fails only on OS permission errors in the temp directory."
	if removeErr := agentrun.RemoveFile(configFile); removeErr != nil {
		return errors.Join(runErr, fmt.Errorf("remove the tool configuration: %w", removeErr))
	}
	return runErr
}

// claudeCommand is the program to start for a run and the files it works with.
type claudeCommand struct {
	name       string
	args       []string
	claudeArgs []string
	configFile string
}

// streamClaude starts the program and streams its responses until it ends.
func (s *agentServer) streamClaude(
	ctx context.Context,
	send func(*agentrun.Response) error,
	msg *agentrun.Request,
	cmd claudeCommand,
) error {
	conversation := msg.Conversation
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// gomutants:disable-next-line INTEGER_INCREMENT reason="work.Output is unused because agent-claude writes responses directly to the stream"
	work := agentrun.NewWork(conversation, 0, agentrun.StopByCancel(cancel))
	if !s.registry.Add(work) {
		return rpcerror.New(rpcerror.Unavailable, errors.New("this agent is closing"))
	}
	defer s.registry.Remove(work)
	defer work.End()

	// The wait for the model begins with the program, and the stream opens on saying so.
	if err := send(&agentrun.Response{
		Step: &agentrun.ModelStartedStep{},
	}); err != nil {
		return err
	}

	proc, out, stderrBuf, err := s.launchAgentCmd(runCtx, cmd.name, cmd.args, cmd.claudeArgs, msg)
	if err != nil {
		return err
	}

	recordSession := func(session string) { s.registry.SetSession(conversation, session) }
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Channel buffer capacity of 1 is a non-blocking send optimization before waitAgentCompletion receives."
	reasons := make(chan string, 1)
	go func() {
		reasons <- parseStream(runCtx, out, send, streamOptions{
			words: toolcall.IndexDeclarations(msg.ToolDeclarations), record: recordSession, clock: s.clock, expectsVault: cmd.configFile != "",
		})
	}()

	stoppedReason, ok := waitAgentCompletion(runCtx, proc, out, stderrBuf, reasons)
	if !ok {
		return nil
	}

	return send(&agentrun.Response{
		Step: &agentrun.StoppedStep{
			Stopped: stoppedReason,
		},
	})
}

func (s *agentServer) launchAgentCmd(
	ctx context.Context,
	cmdName string,
	cmdArgs, claudeArgs []string,
	msg *agentrun.Request,
) (*exec.Cmd, io.ReadCloser, *strings.Builder, error) {
	cmd := exec.CommandContext(ctx, cmdName, slices.Concat(cmdArgs, claudeArgs)...)
	cmd.Dir = msg.Root
	cmd.Env = buildEnvironment(s.getEnvironment())
	cmd.Stdin = strings.NewReader(msg.Prompt)
	process.Supervise(cmd)

	out, err := cmd.StdoutPipe()
	// gomutants:disable-next-line BRANCH_IF reason="cmd.StdoutPipe on a fresh command before Start fails only on OS descriptor exhaustion."
	if err != nil {
		return nil, nil, nil, rpcerror.New(rpcerror.Internal, fmt.Errorf("stdout pipe: %w", err))
	}

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, nil, nil, rpcerror.New(rpcerror.Internal, fmt.Errorf("start claude: %w", err))
	}
	return cmd, out, &stderrBuf, nil
}

func waitAgentCompletion(
	runCtx context.Context,
	cmd *exec.Cmd,
	out io.ReadCloser,
	stderrBuf *strings.Builder,
	reasons <-chan string,
) (string, bool) {
	var failedReason string
	var closeErr error
	select {
	case failedReason = <-reasons:
	case <-runCtx.Done():
		// gomutants:disable-next-line STATEMENT_REMOVE reason="Closing stdout on context cancellation unblocks the reader goroutine while runCtx.Err() returns early below."
		closeErr = out.Close()
		// gomutants:disable-next-line STATEMENT_REMOVE reason="Draining reasons waits for the reader goroutine while runCtx.Err() returns early below."
		failedReason = <-reasons
	}

	waitErr := errors.Join(closeErr, cmd.Wait())
	if runCtx.Err() != nil {
		return "", false
	}

	if failedReason != "" {
		return failedReason, true
	}
	if waitErr != nil {
		return agentrun.FailureMessage(waitErr, stderrBuf.String()), true
	}
	return "done", true
}

// getCommand returns the program to start and the arguments that precede its
// own: the command line set by the installation, or else the program found on
// the system. An empty program means none was found.
func (s *agentServer) getCommand(command []string) (string, []string) {
	if len(command) > 0 {
		return command[0], command[1:]
	}
	return s.findClaude(), nil
}

func (s *agentServer) FinishConversation(ctx context.Context, conversation string) error {
	return s.registry.FinishConversation(ctx, conversation)
}

func (s *agentServer) GetState(_ context.Context) (*agentrun.State, error) {
	var unreachable string
	if s.findClaude() == "" {
		unreachable = "Claude Code CLI not found"
	}
	return &agentrun.State{Unreachable: unreachable}, nil
}

// GetPrograms lists Claude Code, whether it is installed under the command line the installation
// set or where it is found, and the models it accepts.
func (s *agentServer) GetPrograms(ctx context.Context, commands []agentrun.ProgramCommand) ([]*agentrun.Program, error) {
	var configured []string
	for _, one := range commands {
		if one.Program == programName {
			configured = one.Command
		}
	}
	command, extra := s.getCommand(configured)
	program := &agentrun.Program{
		Name:        programName,
		Title:       programTitle,
		IsInstalled: command != "",
		IsDefault:   true,
		LimitsTools: true,
		Models:      nil,
	}
	program.Models = s.listModels(ctx, command, extra)
	return []*agentrun.Program{program}, nil
}

func (s *agentServer) RegisterTool(_ context.Context, tool *agentrun.ToolDefinition) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tools = append(s.tools, tool)
	return nil
}

func (s *agentServer) ExtendSystemPrompt(_ context.Context, extension string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.systemPrompt += extension
	return s.systemPrompt, nil
}

func (s *agentServer) GetTools(_ context.Context) (*agentrun.ToolSet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	toolsCopy := make([]*agentrun.ToolDefinition, len(s.tools))
	copy(toolsCopy, s.tools)

	return &agentrun.ToolSet{
		Tools:                 toolsCopy,
		SystemPromptExtension: s.systemPrompt,
	}, nil
}

// Mount registers the agent's services on s and hands Configure's settings to the agent.
// The runs the agent started end when s closes.
func Mount(s *server.Server) error {
	agent := new(agentServer)
	agent.clock = clock.System{}
	agent.models = newModelCache(true)
	return mount(s, agent)
}

func mount(s *server.Server, agentSvc *agentServer) error {
	s.SetUpdateConfig(agentSvc.applyConfig)
	s.Register(agentrun.NewHandler(agentSvc, s.HandlerOptions()...))
	s.OnClose(agentSvc.registry.Close)
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		agent := new(agentServer)
		agent.clock = clock.System{}
		agent.environ = getVariables(vars)
		agent.models = newModelCache(true)
		return mount(s, agent)
	})
}

// listModels returns the models the program offers, as it lists them. The aliases of the fallback
// stand where it is not installed or does not answer.
func (s *agentServer) listModels(ctx context.Context, command string, extra []string) []*reasoning.Model {
	var models []claudeModel
	var err error
	if command == "" {
		err = errors.New("the program is not installed")
	} else {
		models, err = s.getModels(ctx, slices.Concat([]string{command}, extra), s.getEnvironment())
	}
	if err != nil {
		// gomutants:disable-next-line STATEMENT_REMOVE reason="Diagnostic log to os.Stderr does not affect the returned fallback models."
		fmt.Fprintf(os.Stderr, "list models of claude: %v\n", err)
		models = getFallbackModels()
	}
	out := make([]*reasoning.Model, 0, len(models))
	for _, one := range models {
		out = append(out, one.toProto())
	}
	return out
}

// getReasoning returns the reasoning depth the program starts with: the
// requested depth, where the model supports it. A model the program does not
// list uses the requested depth unchanged.
func (s *agentServer) getReasoning(model, requested string) string {
	if model == "" {
		model = defaultModelName
	}
	levels, items := s.findModelLevels(model)
	if items {
		return reasoning.Pick(levels, requested)
	}
	if reasoning.IsKnown(requested) {
		return requested
	}
	return ""
}
