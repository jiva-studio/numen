package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

type agentServer struct {
	numenv1connect.UnimplementedAgentServiceHandler
	mu           sync.RWMutex
	systemPrompt string
	tools        []*numenv1.AgentToolDefinition
	modelName    string
	models       []AgentModel
	command      []string
	mayReadHooks bool
	sessions     map[string]string
	activeRuns   map[string]context.CancelFunc
}

func (s *agentServer) applyConfig(ctx context.Context, values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := values["model"]; ok && m != "" {
		s.modelName = m
	}
	if v, ok := values["readsHooksAndSkills"]; ok {
		s.mayReadHooks = v == "true"
	}
	apiKey := values["apiKey"]
	baseURL := values["baseUrl"]
	if apiKey != "" || os.Getenv(anthropicKeyVar) != "" {
		if models, err := fetchAnthropicModels(ctx, apiKey, baseURL); err == nil && len(models) > 0 {
			s.models = models
		}
	}
}

func (s *agentServer) AskAgent(
	ctx context.Context,
	req *connect.Request[numenv1.AskAgentRequest],
	stream *connect.ServerStream[numenv1.AskAgentResponse],
) error {
	s.mu.RLock()
	systemPrompt := s.systemPrompt
	model := req.Msg.GetModel()
	if model == "" {
		model = s.modelName
	}
	customCmd := s.command
	mayReadHooks := s.mayReadHooks
	conversation := req.Msg.GetConversation()
	session := s.sessions[conversation]
	var toolNames []string
	if len(req.Msg.GetTools()) > 0 {
		toolNames = req.Msg.GetTools()
	} else {
		for _, t := range s.tools {
			toolNames = append(toolNames, t.GetName())
		}
	}
	s.mu.RUnlock()

	instructions := req.Msg.GetInstructions()
	if systemPrompt != "" {
		if instructions != "" {
			instructions = instructions + "\n" + systemPrompt
		} else {
			instructions = systemPrompt
		}
	}

	cmdName := ""
	var cmdArgs []string
	if len(customCmd) > 0 {
		cmdName = customCmd[0]
		cmdArgs = customCmd[1:]
	} else {
		cmdName = findClaude()
	}

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

	toolsURL, toolsToken := getToolsEndpoint()
	configFile, err := writeMcpConfig(toolsURL, toolsToken)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("write mcp config: %w", err))
	}
	if configFile != "" {
		defer os.Remove(configFile)
	}

	turns := int(req.Msg.GetMaxSteps())
	claudeArgs := buildClaudeArguments(instructions, model, configFile, session, turns, toolNames, mayReadHooks)
	fullArgs := append(cmdArgs, claudeArgs...)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	if conversation != "" {
		s.mu.Lock()
		if s.activeRuns == nil {
			s.activeRuns = make(map[string]context.CancelFunc)
		}
		s.activeRuns[conversation] = cancel
		s.mu.Unlock()

		defer func() {
			s.mu.Lock()
			delete(s.activeRuns, conversation)
			s.mu.Unlock()
		}()
	}

	cmd := exec.CommandContext(runCtx, cmdName, fullArgs...)
	cmd.Dir = req.Msg.GetRoot()
	cmd.Env = buildEnvironment(os.Environ())
	cmd.Stdin = strings.NewReader(req.Msg.GetPrompt())

	out, err := cmd.StdoutPipe()
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("stdout pipe: %w", err))
	}

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("start claude: %w", err))
	}

	recordSession := func(sess string) {
		if conversation != "" {
			s.mu.Lock()
			if s.sessions == nil {
				s.sessions = make(map[string]string)
			}
			s.sessions[conversation] = sess
			s.mu.Unlock()
		}
	}

	failedReason := parseStream(runCtx, out, stream.Send, getToolWords(req.Msg.GetToolDeclarations()), recordSession)

	waitErr := cmd.Wait()
	if runCtx.Err() != nil {
		return nil
	}

	stoppedReason := "done"
	if failedReason != "" {
		stoppedReason = failedReason
	} else if waitErr != nil {
		stoppedReason = getFailureMessage(waitErr, stderrBuf.String())
	}

	return stream.Send(&numenv1.AskAgentResponse{
		Step: &numenv1.AskAgentResponse_Stopped{
			Stopped: stoppedReason,
		},
	})
}

func (s *agentServer) FinishConversation(
	_ context.Context,
	req *connect.Request[numenv1.FinishConversationRequest],
) (*connect.Response[numenv1.FinishConversationResponse], error) {
	conv := req.Msg.GetConversation()
	s.mu.Lock()
	if cancel, ok := s.activeRuns[conv]; ok {
		cancel()
		delete(s.activeRuns, conv)
	}
	if s.sessions != nil {
		delete(s.sessions, conv)
	}
	s.mu.Unlock()
	return connect.NewResponse(&numenv1.FinishConversationResponse{}), nil
}

func (s *agentServer) GetAgentState(
	_ context.Context,
	_ *connect.Request[numenv1.GetAgentStateRequest],
) (*connect.Response[numenv1.GetAgentStateResponse], error) {
	s.mu.RLock()
	customCmd := s.command
	s.mu.RUnlock()

	var unreachable string
	if len(customCmd) == 0 && findClaude() == "" {
		unreachable = "Claude Code CLI not found"
	}

	return connect.NewResponse(&numenv1.GetAgentStateResponse{
		Unreachable: unreachable,
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

// getToolWords indexes the declarations of the vault's tools by tool name.
func getToolWords(declarations []*numenv1.ToolDeclaration) map[string]toolWords {
	words := make(map[string]toolWords, len(declarations))
	for _, d := range declarations {
		words[d.GetName()] = toolWords{
			title:   d.GetTitle(),
			kind:    d.GetKind(),
			about:   d.GetAbout(),
			element: d.GetElement(),
			match:   d.GetMatch(),
			text:    d.GetText(),
		}
	}
	return words
}

func getToolsEndpoint() (string, string) {
	if u := os.Getenv("NUMEN_TOOLS_URL"); u != "" {
		return u, os.Getenv("NUMEN_TOOLS_TOKEN")
	}
	if u := os.Getenv("NUMEN_MCP_URL"); u != "" {
		return u, os.Getenv("NUMEN_MCP_TOKEN")
	}

	configDir, err := os.UserConfigDir()
	if err == nil {
		path := filepath.Join(configDir, "numen", "agents.json")
		if data, err := os.ReadFile(path); err == nil {
			var disc struct {
				URL   string `json:"url"`
				Token string `json:"token"`
			}
			if err := json.Unmarshal(data, &disc); err == nil {
				return disc.URL, disc.Token
			}
		}
	}
	return "", ""
}

func main() {
	agentSvc := &agentServer{
		modelName:  os.Getenv("NUMEN_CLAUDE_MODEL"),
		sessions:   make(map[string]string),
		activeRuns: make(map[string]context.CancelFunc),
	}

	srv := server.New(server.WithConfigure(func(ctx context.Context, values map[string]string) error {
		agentSvc.applyConfig(ctx, values)
		return nil
	}))

	srv.Register(numenv1connect.NewAgentServiceHandler(agentSvc, srv.HandlerOptions()...))

	if err := srv.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
