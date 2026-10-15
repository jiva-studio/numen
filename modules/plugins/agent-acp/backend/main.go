package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

type lifecycleServer struct {
	numenv1connect.UnimplementedPluginLifecycleServiceHandler
	shutdownTrigger chan struct{}
}

func (s *lifecycleServer) Ping(
	_ context.Context,
	_ *connect.Request[numenv1.PingRequest],
) (*connect.Response[numenv1.PingResponse], error) {
	return connect.NewResponse(&numenv1.PingResponse{
		Status: "ok",
	}), nil
}

func (s *lifecycleServer) Shutdown(
	_ context.Context,
	_ *connect.Request[numenv1.ShutdownRequest],
) (*connect.Response[numenv1.ShutdownResponse], error) {
	go func() {
		time.Sleep(100 * time.Millisecond)
		close(s.shutdownTrigger)
	}()
	return connect.NewResponse(&numenv1.ShutdownResponse{
		Success: true,
	}), nil
}

type agentServer struct {
	numenv1connect.UnimplementedAgentServiceHandler
	runner       *AgentRunner
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

func authCheck(expectedToken string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if expectedToken == "" {
				return next(ctx, req)
			}
			auth := req.Header().Get("Authorization")
			token := strings.TrimPrefix(auth, "Bearer ")
			if token != expectedToken {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("unauthorized"))
			}
			return next(ctx, req)
		}
	}
}

func main() {
	var portFlag int
	var tokenFlag string

	flag.IntVar(&portFlag, "port", 0, "Port to listen on")
	flag.StringVar(&tokenFlag, "token", "", "Authentication token")
	flag.Parse()

	if portFlag == 0 {
		if envPort := os.Getenv("NUMEN_PLUGIN_PORT"); envPort != "" {
			if p, err := strconv.Atoi(envPort); err == nil {
				portFlag = p
			}
		}
	}
	if tokenFlag == "" {
		tokenFlag = os.Getenv("NUMEN_PLUGIN_TOKEN")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", portFlag))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to listen: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	actualPort := listener.Addr().(*net.TCPAddr).Port

	shutdownCh := make(chan struct{})
	reg := NewRegistry()
	runner := &AgentRunner{Registry: reg}

	life := &lifecycleServer{shutdownTrigger: shutdownCh}
	agentSvc := &agentServer{
		runner: runner,
	}

	interceptors := connect.WithInterceptors(authCheck(tokenFlag))

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(life, interceptors))
	mux.Handle(numenv1connect.NewAgentServiceHandler(agentSvc, interceptors))

	handler := h2c.NewHandler(mux, &http2.Server{})
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		}
	}()

	fmt.Printf("NUMEN_READY port=%d\n", actualPort)
	_ = os.Stdout.Sync()

	stdinEOF := make(chan struct{})
	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				close(stdinEOF)
				return
			}
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
	case <-shutdownCh:
	case <-stdinEOF:
	}

	_ = reg.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
