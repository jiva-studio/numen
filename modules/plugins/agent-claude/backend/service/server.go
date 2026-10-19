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
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/completion"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/toolcall"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// programName is what a task asks for the program by, and what the settings keep its section under.
const programName = "claude"

// readsHooksOption is the option that lets the program read the hooks, skills and standing
// instructions this machine holds for it. What a vault carries is refused either way.
const readsHooksOption = "reads_hooks_and_skills"

// programTitle is what the program is called when shown to a person.
const programTitle = "Claude Code"

type agentServer struct {
	numenv1connect.UnimplementedAgentServiceHandler
	numenv1connect.UnimplementedCompletionServiceHandler
	mu              sync.RWMutex
	systemPrompt    string
	tools           []*numenv1.AgentToolDefinition
	registry        agentrun.Registry
	catalogue       modelCatalogue
	completionModel string
}

// applyConfig keeps the key and the endpoint the models are asked with, dropping the models asked
// with the ones before, and the model that answers a completion.
func (s *agentServer) applyConfig(values map[string]string) {
	if model, ok := values["model"]; ok && model != "" {
		s.mu.Lock()
		s.completionModel = model
		s.mu.Unlock()
	}
	s.catalogue.update(values)
}

func (s *agentServer) AskAgent(
	ctx context.Context,
	req *connect.Request[numenv1.AskAgentRequest],
	stream *connect.ServerStream[numenv1.AskAgentResponse],
) error {
	msg := req.Msg
	s.mu.RLock()
	systemPrompt := s.systemPrompt
	toolNames := msg.GetTools()
	if len(toolNames) == 0 {
		for _, t := range s.tools {
			toolNames = append(toolNames, t.GetName())
		}
	}
	s.mu.RUnlock()

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

	configFile, err := writeMcpConfig(msg.GetToolsUrl(), msg.GetToolsToken())
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("write mcp config: %w", err))
	}

	conversation := msg.GetConversation()
	claudeArgs := buildClaudeArguments(claudeArguments{
		instructions:  instructions,
		model:         msg.GetModel(),
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
	cmd.Env = buildEnvironment(os.Environ())
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
		reasons <- parseStream(runCtx, out, stream.Send, toolcall.ReadWords(msg.GetToolDeclarations()), recordSession)
	}()

	// A run let go of ends here, whatever the programs the child started hold of its output.
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
		stoppedReason = completion.FailureMessage(waitErr, stderrBuf.String())
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
// set or where it is found, and the models the endpoint lists.
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
	command, _ := getCommand(configured)
	program := &numenv1.AgentProgram{
		Name:        programName,
		Title:       programTitle,
		IsInstalled: command != "",
		IsDefault:   true,
		LimitsTools: true,
	}
	for _, one := range s.catalogue.get(ctx) {
		program.Models = append(program.Models, &numenv1.AgentModel{Name: one.Name, Title: one.Title})
	}
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

// Mount registers the agent's services on s and hands Configure's settings to the model catalogue.
// The runs the agent started end when s closes.
func Mount(s *server.Server) error {
	agentSvc := &agentServer{}
	s.SetUpdateConfig(agentSvc.applyConfig)
	s.Register(numenv1connect.NewAgentServiceHandler(agentSvc, s.HandlerOptions()...))
	s.Register(numenv1connect.NewCompletionServiceHandler(agentSvc, s.HandlerOptions()...))
	s.OnClose(func() { _ = agentSvc.registry.Close() })
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, Mount)
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
