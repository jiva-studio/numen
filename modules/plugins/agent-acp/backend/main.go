package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

type agentServer struct {
	numenv1connect.UnimplementedAgentServiceHandler
	numenv1connect.UnimplementedCompletionServiceHandler
	runner       *AgentRunner
	command      []string
	mu           sync.RWMutex
	systemPrompt string
	tools        []*numenv1.AgentToolDefinition
}

func (s *agentServer) AskAgent(
	ctx context.Context,
	req *connect.Request[numenv1.AskAgentRequest],
	stream *connect.ServerStream[numenv1.AskAgentResponse],
) error {
	_ = stream.Send(&numenv1.AskAgentResponse{
		Step: &numenv1.AskAgentResponse_ModelStarted{
			ModelStarted: &numenv1.ModelStarted{},
		},
	})

	agentName := req.Msg.GetAgent()
	if agentName == "" {
		installed := GetInstalledAgents()
		if len(installed) > 0 {
			agentName = installed[0]
		} else {
			agentName = "antigravity"
		}
	}

	endpoint := resolveMCPEndpoint()

	params := RunParams{
		Program:      agentName,
		Model:        req.Msg.GetModel(),
		Prompt:       req.Msg.GetPrompt(),
		Conversation: req.Msg.GetConversation(),
		Tools:        endpoint,
		MaxSteps:     int(req.Msg.GetMaxSteps()),
	}

	work, err := s.runner.StartRun(ctx, params)
	if err != nil {
		return fmt.Errorf("start agent run: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			work.Stop()
			return ctx.Err()
		case resp, ok := <-work.Output:
			if !ok {
				return nil
			}
			if err := stream.Send(resp); err != nil {
				work.Stop()
				return err
			}
		}
	}
}

func (s *agentServer) FinishConversation(
	ctx context.Context,
	req *connect.Request[numenv1.FinishConversationRequest],
) (*connect.Response[numenv1.FinishConversationResponse], error) {
	conv := req.Msg.GetConversation()
	if conv != "" {
		_ = s.runner.Registry.EndConversation(ctx, conv)
	}
	return connect.NewResponse(&numenv1.FinishConversationResponse{}), nil
}

func (s *agentServer) GetAgentState(
	_ context.Context,
	_ *connect.Request[numenv1.GetAgentStateRequest],
) (*connect.Response[numenv1.GetAgentStateResponse], error) {
	installed := GetInstalledAgents()
	if len(installed) == 0 {
		return connect.NewResponse(&numenv1.GetAgentStateResponse{
			Unreachable: "No external coding agents (Claude, Antigravity, Codex, Cursor) found on this machine",
		}), nil
	}
	return connect.NewResponse(&numenv1.GetAgentStateResponse{}), nil
}

func (s *agentServer) RegisterTool(
	_ context.Context,
	req *connect.Request[numenv1.RegisterToolRequest],
) (*connect.Response[numenv1.RegisterToolResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tool := req.Msg.GetTool(); tool != nil {
		s.tools = append(s.tools, tool)
	}
	return connect.NewResponse(&numenv1.RegisterToolResponse{Success: true}), nil
}

func (s *agentServer) ExtendSystemPrompt(
	_ context.Context,
	req *connect.Request[numenv1.ExtendSystemPromptRequest],
) (*connect.Response[numenv1.ExtendSystemPromptResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ext := req.Msg.GetExtension(); ext != "" {
		if s.systemPrompt != "" {
			s.systemPrompt += "\n" + ext
		} else {
			s.systemPrompt = ext
		}
	}
	return connect.NewResponse(&numenv1.ExtendSystemPromptResponse{
		FullPrompt: s.systemPrompt,
	}), nil
}

func (s *agentServer) GetAgentTools(
	_ context.Context,
	_ *connect.Request[numenv1.GetAgentToolsRequest],
) (*connect.Response[numenv1.GetAgentToolsResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return connect.NewResponse(&numenv1.GetAgentToolsResponse{
		Tools:                 s.tools,
		SystemPromptExtension: s.systemPrompt,
	}), nil
}

func resolveMCPEndpoint() Endpoint {
	url := os.Getenv("NUMEN_MCP_URL")
	token := os.Getenv("NUMEN_TOKEN")
	if url != "" {
		return Endpoint{URL: url, Token: token}
	}

	home, err := os.UserHomeDir()
	if err == nil {
		discPath := filepath.Join(home, ".config", "numen", "agents.json")
		if data, err := os.ReadFile(discPath); err == nil {
			var disc struct {
				URL   string `json:"url"`
				Token string `json:"token"`
			}
			if err := json.Unmarshal(data, &disc); err == nil && disc.URL != "" {
				return Endpoint{URL: disc.URL, Token: disc.Token}
			}
		}
	}
	return Endpoint{}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	s := server.New()

	reg := NewRegistry()
	defer reg.Close()

	agentSvc := &agentServer{
		runner: &AgentRunner{Registry: reg},
	}

	s.Register(numenv1connect.NewAgentServiceHandler(agentSvc, s.HandlerOptions()...))
	s.Register(numenv1connect.NewCompletionServiceHandler(agentSvc, s.HandlerOptions()...))

	return s.Serve()
}
