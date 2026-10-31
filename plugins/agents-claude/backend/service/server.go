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

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/toolcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cache"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

// programName is the name tasks use for the program, and the settings section key.
const programName = "claude"

// readsHooksOption is the option that lets the program read the hooks, skills and standing
const readsHooksOption = "reads_hooks_and_skills"

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

	// environ is the variables a program of the agent is started with; nil is those of this process.
	environ []string

	// models are the models the program listed.
	models *cache.Cache[[]claudeModel]

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

// applyConfig keeps the model that answers a task naming none, and optional executable path.
func (s *agentServer) applyConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if model, ok := values["model"]; ok && model != "" {
		s.defaultModel = model
	}
	if execPath, ok := values["executable_path"]; ok {
		s.executablePath = strings.TrimSpace(execPath)
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
	toolNames := msg.Tools
	if len(toolNames) > 0 {
		configFile, err = writeMcpConfig(msg.ToolsURL, msg.ToolsToken)
		if err != nil {
			return nil, "", connect.NewError(connect.CodeInternal, fmt.Errorf("write mcp config: %w", err))
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
		mayReadHooks:  msg.Options[readsHooksOption] == "true",
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

	runErr := s.streamClaude(ctx, send, msg, cmdName, cmdArgs, claudeArgs, configFile)
	if removeErr := agentrun.RemoveFile(configFile); removeErr != nil {
		return errors.Join(runErr, fmt.Errorf("remove the tool configuration: %w", removeErr))
	}
	return runErr
}

// streamClaude starts the program and streams its responses until it ends.
func (s *agentServer) streamClaude(
	ctx context.Context,
	send func(*agentrun.Response) error,
	msg *agentrun.Request,
	cmdName string,
	cmdArgs, claudeArgs []string,
	configFile string,
) error {
	conversation := msg.Conversation
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// gomutants:disable-next-line INTEGER_INCREMENT reason="work.Output is unused because agent-claude writes responses directly to the stream"
	work := agentrun.NewWork(conversation, 0, agentrun.StopByCancel(cancel))
	if !s.registry.Add(work) {
		return connect.NewError(connect.CodeUnavailable, errors.New("this agent is closing"))
	}
	defer s.registry.Remove(work)
	defer work.End()

	// The wait for the model begins with the program, and the stream opens on saying so.
	if err := send(&agentrun.Response{
		Step: &agentrun.ModelStartedStep{},
	}); err != nil {
		return err
	}

	cmd, out, stderrBuf, err := s.launchAgentCmd(runCtx, cmdName, cmdArgs, claudeArgs, msg)
	if err != nil {
		return err
	}

	recordSession := func(session string) { s.registry.SetSession(conversation, session) }
	reasons := make(chan string, 1)
	go func() {
		reasons <- parseStream(runCtx, out, send, toolcall.IndexDeclarations(msg.ToolDeclarations), recordSession, s.clock, configFile != "")
	}()

	stoppedReason, ok := waitAgentCompletion(runCtx, cmd, out, stderrBuf, reasons)
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
	supervise(cmd)

	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, connect.NewError(connect.CodeInternal, fmt.Errorf("stdout pipe: %w", err))
	}

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, nil, nil, connect.NewError(connect.CodeInternal, fmt.Errorf("start claude: %w", err))
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
		closeErr = out.Close()
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
		fmt.Fprintf(os.Stderr, "list models of claude: %v\n", err)
		models = fallbackModels
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
