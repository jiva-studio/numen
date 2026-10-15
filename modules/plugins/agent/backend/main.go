package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
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
	mu           sync.RWMutex
	systemPrompt string
	tools        []*numenv1.AgentToolDefinition
	modelName    string
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Temperature float64       `json:"temperature"`
	Messages    []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
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

	s.mu.RLock()
	systemPrompt := s.systemPrompt
	toolsCount := len(s.tools)
	s.mu.RUnlock()

	if req.Msg.GetInstructions() != "" {
		systemPrompt = req.Msg.GetInstructions() + "\n" + systemPrompt
	}

	key := os.Getenv("NUMEN_AGENT_KEY")
	if key == "" {
		key = os.Getenv("OPENAI_API_KEY")
	}

	baseURL := os.Getenv("NUMEN_AGENT_URL")
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}

	model := req.Msg.GetModel()
	if model == "" {
		model = s.modelName
	}
	if model == "" {
		model = "openai/gpt-4o-mini"
	}

	var reply string
	if key != "" {
		client := &http.Client{Timeout: 60 * time.Second}
		var err error
		reply, err = callLLM(ctx, client, baseURL, key, model, systemPrompt, req.Msg.GetPrompt())
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("llm completion: %w", err))
		}
	} else {
		var toolSummary string
		if toolsCount > 0 {
			toolSummary = fmt.Sprintf(" (%d custom tools available)", toolsCount)
		}
		reply = fmt.Sprintf("Agent received prompt %q%s.", req.Msg.GetPrompt(), toolSummary)
	}

	for _, chunk := range splitChunks(reply, 64) {
		if err := stream.Send(&numenv1.AskAgentResponse{
			Step: &numenv1.AskAgentResponse_Said{
				Said: chunk,
			},
		}); err != nil {
			return err
		}
	}

	return stream.Send(&numenv1.AskAgentResponse{
		Step: &numenv1.AskAgentResponse_Stopped{
			Stopped: "done",
		},
	})
}

func (s *agentServer) FinishConversation(
	_ context.Context,
	_ *connect.Request[numenv1.FinishConversationRequest],
) (*connect.Response[numenv1.FinishConversationResponse], error) {
	return connect.NewResponse(&numenv1.FinishConversationResponse{}), nil
}

func (s *agentServer) GetAgentState(
	_ context.Context,
	_ *connect.Request[numenv1.GetAgentStateRequest],
) (*connect.Response[numenv1.GetAgentStateResponse], error) {
	return connect.NewResponse(&numenv1.GetAgentStateResponse{
		Unreachable: "",
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

func callLLM(
	ctx context.Context,
	client *http.Client,
	baseURL, key, model, systemPrompt, prompt string,
) (string, error) {
	endpoint := strings.TrimSuffix(baseURL, "/") + "/chat/completions"

	body, err := json.Marshal(chatRequest{
		Model:       model,
		Temperature: 0,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+key)

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", err
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no completion returned")
	}
	return chatResp.Choices[0].Message.Content, nil
}

func splitChunks(s string, chunkSize int) []string {
	if len(s) <= chunkSize {
		return []string{s}
	}
	var chunks []string
	for len(s) > chunkSize {
		chunks = append(chunks, s[:chunkSize])
		s = s[chunkSize:]
	}
	if len(s) > 0 {
		chunks = append(chunks, s)
	}
	return chunks
}

func authCheck(expectedToken string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if expectedToken != "" {
				authHeader := req.Header().Get("Authorization")
				expected := "Bearer " + expectedToken
				if authHeader != expected {
					return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token"))
				}
			}
			return next(ctx, req)
		}
	}
}

type manifestPayload struct {
	Agent struct {
		SystemPrompt string `json:"systemPrompt"`
		Tools        []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"tools"`
	} `json:"agent"`
}

func loadManifest(path string) (string, []*numenv1.AgentToolDefinition) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil
	}
	var m manifestPayload
	if err := json.Unmarshal(data, &m); err != nil {
		return "", nil
	}

	var tools []*numenv1.AgentToolDefinition
	for _, t := range m.Agent.Tools {
		tools = append(tools, &numenv1.AgentToolDefinition{
			Name:        t.Name,
			Description: t.Description,
		})
	}
	return m.Agent.SystemPrompt, tools
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

	systemPrompt, tools := loadManifest("manifest.json")

	shutdownCh := make(chan struct{})
	life := &lifecycleServer{shutdownTrigger: shutdownCh}
	agentSvc := &agentServer{
		systemPrompt: systemPrompt,
		tools:        tools,
		modelName:    os.Getenv("NUMEN_AGENT_MODEL"),
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

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
	case <-shutdownCh:
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
