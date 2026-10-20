// Package service is the backend of the Claude Code agent plugin.
package service

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/toolcall"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// programName is what a task asks for the program by, and what the settings keep its section under.
const programName = "claude"

// readsHooksOption is the option that lets the program read the hooks, skills and standing
// instructions this machine holds for it. What a vault holds is refused either way.
const readsHooksOption = "reads_hooks_and_skills"

// programTitle is what the program is called when shown to a person.
const programTitle = "Claude Code"

type agentServer struct {
	numenv1connect.UnimplementedAgentServiceHandler
	mu           sync.RWMutex
	systemPrompt string
	tools        []*numenv1.AgentToolDefinition
	registry     agentrun.Registry
	defaultModel string

	// environ is the variables a program of the agent is started with; nil is those of this process.
	environ []string

	// models are the models the program listed.
	models modelCache
}

// getEnvironment returns the variables a program of the agent is started with.
func (s *agentServer) getEnvironment() []string {
	if s.environ != nil {
		return s.environ
	}
	return os.Environ()
}

// applyConfig keeps the model that answers a task naming none.
func (s *agentServer) applyConfig(values map[string]string) {
	if model, ok := values["model"]; ok && model != "" {
		s.mu.Lock()
		s.defaultModel = model
		s.mu.Unlock()
	}
}

func (s *agentServer) AskAgent(
	ctx context.Context,
	req *connect.Request[numenv1.AskAgentRequest],
	stream *connect.ServerStream[numenv1.AskAgentResponse],
) error {
	msg := req.Msg
	s.mu.RLock()
	systemPrompt := s.systemPrompt
	defaultModel := s.defaultModel
	s.mu.RUnlock()
	toolNames := msg.GetTools()

	instructions := msg.GetInstructions()
	if systemPrompt != "" {
		if instructions != "" {
			instructions += "\n"
		}
		instructions += systemPrompt
	}

	cmdName, cmdArgs := getCommand(msg.GetCommand())
	if cmdName == "" {
		_ = stream.Send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_Said{
				Said: "Claude Code CLI is not installed or not found on PATH.",
			},
		})
		return stream.Send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_Stopped{
				Stopped: "claude not installed",
			},
		})
	}

	var configFile string
	if len(toolNames) > 0 {
		var err error
		configFile, err = writeMcpConfig(msg.GetToolsUrl(), msg.GetToolsToken())
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("write mcp config: %w", err))
		}
	}
	model := msg.GetModel()
	if model == "" {
		model = defaultModel
	}

	level := s.getReasoning(model, msg.GetReasoning())
	if model == defaultModelName {
		model = ""
	}

	conversation := msg.GetConversation()
	claudeArgs := buildClaudeArguments(claudeArguments{
		instructions:  instructions,
		model:         model,
		reasoning:     level,
		configuration: configFile,
		session:       s.registry.GetSession(conversation),
		turns:         int(msg.GetMaxSteps()),
		tools:         toolNames,
		mayReadHooks:  msg.GetOptions()[readsHooksOption] == "true",
	})

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// gomutants:disable-next-line INTEGER_INCREMENT reason="work.Output is unused because agent-claude writes responses directly to the stream"
	work := agentrun.NewWork(conversation, 0, cancel)
	if !s.registry.Add(work) {
		removeFile(configFile)
		return connect.NewError(connect.CodeUnavailable, errors.New("this agent is closing"))
	}
	defer s.registry.Remove(work)
	defer work.End()
	defer removeFile(configFile)

	// The wait for the model begins with the program, and the stream opens on saying so.
	if err := stream.Send(&numenv1.AskAgentResponse{
		Step: &numenv1.AskAgentResponse_ModelStarted{ModelStarted: &numenv1.ModelStarted{}},
	}); err != nil {
		return err
	}

	cmd := exec.CommandContext(runCtx, cmdName, slices.Concat(cmdArgs, claudeArgs)...)
	cmd.Dir = msg.GetRoot()
	cmd.Env = buildEnvironment(s.getEnvironment())
	cmd.Stdin = strings.NewReader(msg.GetPrompt())
	agentrun.Supervise(cmd)

	out, err := cmd.StdoutPipe()
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("stdout pipe: %w", err))
	}

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("start claude: %w", err))
	}

	recordSession := func(session string) { s.registry.SetSession(conversation, session) }
	reasons := make(chan string, 1)
	go func() {
		reasons <- parseStream(runCtx, out, stream.Send, toolcall.IndexDeclarations(msg.GetToolDeclarations()), recordSession)
	}()

	// A run that is closed ends here, whatever the programs the child started hold of its output.
	var failedReason string
	select {
	case failedReason = <-reasons:
	case <-runCtx.Done():
		_ = out.Close()
		failedReason = <-reasons
	}

	waitErr := cmd.Wait()
	if runCtx.Err() != nil {
		return nil
	}

	stoppedReason := "done"
	if failedReason != "" {
		stoppedReason = failedReason
	} else if waitErr != nil {
		stoppedReason = agentrun.FailureMessage(waitErr, stderrBuf.String())
	}

	return stream.Send(&numenv1.AskAgentResponse{
		Step: &numenv1.AskAgentResponse_Stopped{
			Stopped: stoppedReason,
		},
	})
}

// getCommand is the program to start and what stands before its own arguments: the command line
// the installation set, else the program where it is found. Empty is a program nowhere to be found.
func getCommand(command []string) (string, []string) {
	if len(command) > 0 {
		return command[0], command[1:]
	}
	return findClaude(), nil
}

func (s *agentServer) FinishConversation(
	ctx context.Context,
	req *connect.Request[numenv1.FinishConversationRequest],
) (*connect.Response[numenv1.FinishConversationResponse], error) {
	if err := s.registry.EndConversation(ctx, req.Msg.GetConversation()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&numenv1.FinishConversationResponse{}), nil
}

func (s *agentServer) GetAgentState(
	_ context.Context,
	_ *connect.Request[numenv1.GetAgentStateRequest],
) (*connect.Response[numenv1.GetAgentStateResponse], error) {
	var unreachable string
	if findClaude() == "" {
		unreachable = "Claude Code CLI not found"
	}
	return connect.NewResponse(&numenv1.GetAgentStateResponse{Unreachable: unreachable}), nil
}

// GetAgentPrograms lists Claude Code, whether it is installed under the command line the installation
// set or where it is found, and the models it accepts.
func (s *agentServer) GetAgentPrograms(
	ctx context.Context,
	req *connect.Request[numenv1.GetAgentProgramsRequest],
) (*connect.Response[numenv1.GetAgentProgramsResponse], error) {
	var configured []string
	for _, one := range req.Msg.GetCommands() {
		if one.GetProgram() == programName {
			configured = one.GetCommand()
		}
	}
	command, extra := getCommand(configured)
	program := &numenv1.AgentProgram{
		Name:        programName,
		Title:       programTitle,
		IsInstalled: command != "",
		IsDefault:   true,
		LimitsTools: true,
	}
	program.Models = s.listModels(ctx, command, extra)
	return connect.NewResponse(&numenv1.GetAgentProgramsResponse{
		Programs: []*numenv1.AgentProgram{program},
	}), nil
}

func (s *agentServer) RegisterTool(
	_ context.Context,
	req *connect.Request[numenv1.RegisterToolRequest],
) (*connect.Response[numenv1.RegisterToolResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tools = append(s.tools, req.Msg.GetTool())
	return connect.NewResponse(&numenv1.RegisterToolResponse{
		Success: true,
	}), nil
}

func (s *agentServer) ExtendSystemPrompt(
	_ context.Context,
	req *connect.Request[numenv1.ExtendSystemPromptRequest],
) (*connect.Response[numenv1.ExtendSystemPromptResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.systemPrompt += req.Msg.GetExtension()
	return connect.NewResponse(&numenv1.ExtendSystemPromptResponse{
		Success:    true,
		FullPrompt: s.systemPrompt,
	}), nil
}

func (s *agentServer) GetAgentTools(
	_ context.Context,
	_ *connect.Request[numenv1.GetAgentToolsRequest],
) (*connect.Response[numenv1.GetAgentToolsResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	toolsCopy := make([]*numenv1.AgentToolDefinition, len(s.tools))
	copy(toolsCopy, s.tools)

	return connect.NewResponse(&numenv1.GetAgentToolsResponse{
		Tools:                 toolsCopy,
		SystemPromptExtension: s.systemPrompt,
	}), nil
}

// Mount registers the agent's services on s and hands Configure's settings to the agent.
// The runs the agent started end when s closes.
func Mount(s *server.Server) error {
	return mount(s, &agentServer{})
}

func mount(s *server.Server, agentSvc *agentServer) error {
	s.SetUpdateConfig(agentSvc.applyConfig)
	s.Register(numenv1connect.NewAgentServiceHandler(agentSvc, s.HandlerOptions()...))
	s.OnClose(func() { _ = agentSvc.registry.Close() })
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		return mount(s, &agentServer{environ: getVariables(vars)})
	})
}

// removeFile removes the file, if there is one.
func removeFile(path string) {
	// gomutants:disable-next-line BRANCH_IF reason="os.Remove on an empty path returns ErrNotExist which is ignored"
	if path == "" {
		return
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "remove %s: %v\n", path, err)
	}
}

// listModels returns the models the program offers, as it lists them. The aliases of the fallback
// stand where it is not installed or does not answer.
func (s *agentServer) listModels(ctx context.Context, command string, extra []string) []*numenv1.AgentModel {
	var models []claudeModel
	var err error
	if command == "" {
		err = errors.New("the program is not installed")
	} else {
		models, err = s.models.get(ctx, slices.Concat([]string{command}, extra), s.getEnvironment())
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "list models of claude: %v\n", err)
		models = fallbackModels
	}
	out := make([]*numenv1.AgentModel, 0, len(models))
	for _, one := range models {
		out = append(out, one.toProto())
	}
	return out
}

// getReasoning returns the depth the program is started with: the one asked for, where the model
// supports it. A model the program did not list takes the depth as asked.
func (s *agentServer) getReasoning(model, asked string) string {
	if model == "" {
		model = defaultModelName
	}
	levels, listed := s.models.find(model)
	if listed {
		return reasoning.Pick(levels, asked)
	}
	if reasoning.IsKnown(asked) {
		return asked
	}
	return ""
}
