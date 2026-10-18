package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

const (
	defaultBaseURL = "https://openrouter.ai/api/v1"
	defaultModel   = "openai/gpt-4o-mini"
	chatTimeout    = 60 * time.Second
)

type ToolExecutor func(ctx context.Context, name string, args string) (string, error)

type agentServer struct {
	numenv1connect.UnimplementedAgentServiceHandler
	numenv1connect.UnimplementedCompletionServiceHandler
	mu           sync.RWMutex
	systemPrompt string
	tools        []*numenv1.AgentToolDefinition
	modelName    string
	baseURL      string
	apiKey       string
	httpClient   *http.Client
	executor     ToolExecutor
}

type toolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type toolDefinition struct {
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`
}

type toolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type toolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function toolCallFunction `json:"function"`
}

func (s *agentServer) updateConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if prompt, ok := values["system_prompt"]; ok && prompt != "" {
		s.systemPrompt = prompt
	}
	if model, ok := values["model"]; ok && model != "" {
		s.modelName = model
	}
	if key, ok := server.ReadSecret(values, server.SecretAPIKey); ok {
		s.apiKey = key
	}
	if base, ok := values["baseUrl"]; ok {
		s.baseURL = base
	}
}

type chatMessage struct {
	Role       string     `json:"role"`
	Content    *string    `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type chatRequest struct {
	Model       string           `json:"model"`
	Temperature float64          `json:"temperature"`
	Messages    []chatMessage    `json:"messages"`
	Tools       []toolDefinition `json:"tools,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message      chatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
}

func (s *agentServer) AskAgent(
	ctx context.Context,
	req *connect.Request[numenv1.AskAgentRequest],
	stream *connect.ServerStream[numenv1.AskAgentResponse],
) error {
	if err := stream.Send(&numenv1.AskAgentResponse{
		Step: &numenv1.AskAgentResponse_ModelStarted{
			ModelStarted: &numenv1.ModelStarted{},
		},
	}); err != nil {
		return err
	}

	s.mu.RLock()
	systemPrompt := s.systemPrompt
	toolsCount := len(s.tools)
	toolsList := make([]*numenv1.AgentToolDefinition, len(s.tools))
	copy(toolsList, s.tools)
	serverBaseURL := s.baseURL
	key := s.apiKey
	serverModel := s.modelName
	client := s.httpClient
	s.mu.RUnlock()

	if req.Msg.GetInstructions() != "" {
		systemPrompt = req.Msg.GetInstructions() + "\n" + systemPrompt
	}

	baseURL := getBaseURL(serverBaseURL)
	model := getModel(req.Msg.GetModel(), serverModel)
	client = getHTTPClient(client)

	if key != "" {
		maxTurns := 10
		if req.Msg.GetMaxSteps() > 0 {
			maxTurns = int(req.Msg.GetMaxSteps())
		}

		toolsPayload := formatTools(toolsList, req.Msg.GetTools())

		var messages []chatMessage
		if systemPrompt != "" {
			sp := systemPrompt
			messages = append(messages, chatMessage{
				Role:    "system",
				Content: &sp,
			})
		}
		userPrompt := req.Msg.GetPrompt()
		messages = append(messages, chatMessage{
			Role:    "user",
			Content: &userPrompt,
		})

		var completedNormally bool
		for turn := 0; turn < maxTurns; turn++ {
			if err := ctx.Err(); err != nil {
				return err
			}

			msg, err := callChat(ctx, client, baseURL, key, model, messages, toolsPayload)
			if err != nil {
				return connect.NewError(connect.CodeInternal, fmt.Errorf("llm completion: %w", err))
			}

			if len(msg.ToolCalls) == 0 {
				var reply string
				if msg.Content != nil {
					reply = *msg.Content
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
				completedNormally = true
				break
			}

			messages = append(messages, *msg)

			for _, tc := range msg.ToolCalls {
				about, path := extractToolMeta(tc.Function.Arguments)
				if err := stream.Send(&numenv1.AskAgentResponse{
					Step: &numenv1.AskAgentResponse_ToolCall{
						ToolCall: &numenv1.ToolCall{
							Tool:    tc.Function.Name,
							About:   about,
							Written: int32(len(tc.Function.Arguments)),
							Path:    path,
						},
					},
				}); err != nil {
					return err
				}

				result, err := s.executeTool(ctx, tc.Function.Name, tc.Function.Arguments)
				if err != nil {
					result = fmt.Sprintf("error executing tool %s: %v", tc.Function.Name, err)
				}

				if err := stream.Send(&numenv1.AskAgentResponse{
					Step: &numenv1.AskAgentResponse_ToolFinished{
						ToolFinished: &numenv1.ToolFinished{},
					},
				}); err != nil {
					return err
				}

				messages = append(messages, chatMessage{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    &result,
				})
			}

			if err := stream.Send(&numenv1.AskAgentResponse{
				Step: &numenv1.AskAgentResponse_ModelStarted{
					ModelStarted: &numenv1.ModelStarted{},
				},
			}); err != nil {
				return err
			}
		}

		if !completedNormally {
			warning := "Agent reached maximum tool call iterations without concluding."
			if err := stream.Send(&numenv1.AskAgentResponse{
				Step: &numenv1.AskAgentResponse_Said{
					Said: warning,
				},
			}); err != nil {
				return err
			}
			return stream.Send(&numenv1.AskAgentResponse{
				Step: &numenv1.AskAgentResponse_Stopped{
					Stopped: "max_turns_exceeded",
				},
			})
		}
	} else {
		var toolSummary string
		if toolsCount > 0 {
			toolSummary = fmt.Sprintf(" (%d custom tools available)", toolsCount)
		}
		reply := fmt.Sprintf("Agent received prompt %q%s.", req.Msg.GetPrompt(), toolSummary)

		for _, chunk := range splitChunks(reply, 64) {
			if err := stream.Send(&numenv1.AskAgentResponse{
				Step: &numenv1.AskAgentResponse_Said{
					Said: chunk,
				},
			}); err != nil {
				return err
			}
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

func (s *agentServer) executeTool(ctx context.Context, name, args string) (string, error) {
	s.mu.RLock()
	executor := s.executor
	s.mu.RUnlock()

	if executor != nil {
		return executor(ctx, name, args)
	}

	return s.defaultExecuteTool(ctx, name, args)
}

func (s *agentServer) defaultExecuteTool(ctx context.Context, name, args string) (string, error) {
	searchURL := os.Getenv("NUMEN_SEARCH_URL")
	if searchURL != "" && (name == "knowledge_search" || name == "search") {
		query := args
		limit := 5
		var parsed struct {
			Query string `json:"query"`
			Limit int    `json:"limit"`
		}
		if err := json.Unmarshal([]byte(args), &parsed); err == nil && parsed.Query != "" {
			query = parsed.Query
			if parsed.Limit > 0 {
				limit = parsed.Limit
			}
		}
		searchClient := numenv1connect.NewSearchServiceClient(http.DefaultClient, searchURL)
		resp, err := searchClient.SearchPassages(ctx, connect.NewRequest(&numenv1.SearchPassagesRequest{
			Query: query,
			Limit: int32(limit),
			Mode:  numenv1.SearchMode_SEARCH_MODE_HYBRID,
		}))
		if err == nil {
			out, err := json.Marshal(resp.Msg.GetFound())
			if err == nil {
				return string(out), nil
			}
		}
	}

	return "", fmt.Errorf("tool %q cannot run: it has no executor and no search service", name)
}

// getBaseURL is the endpoint of the chat API: the one configured, the one in NUMEN_AGENT_URL or the default.
func getBaseURL(configured string) string {
	if configured != "" {
		return configured
	}
	if fromEnv := os.Getenv("NUMEN_AGENT_URL"); fromEnv != "" {
		return fromEnv
	}
	return defaultBaseURL
}

// getModel is the model of the request, else the one configured, else the default.
func getModel(requested, configured string) string {
	if requested != "" {
		return requested
	}
	if configured != "" {
		return configured
	}
	return defaultModel
}

// getHTTPClient is the client given, or one with the default timeout.
func getHTTPClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: chatTimeout}
}

func formatTools(allTools []*numenv1.AgentToolDefinition, requestedTools []string) []toolDefinition {
	var selected []*numenv1.AgentToolDefinition
	if len(requestedTools) > 0 {
		allowed := make(map[string]bool, len(requestedTools))
		for _, name := range requestedTools {
			allowed[name] = true
		}
		for _, t := range allTools {
			if allowed[t.GetName()] {
				selected = append(selected, t)
			}
		}
	} else {
		selected = allTools
	}

	if len(selected) == 0 {
		return nil
	}

	result := make([]toolDefinition, 0, len(selected))
	for _, t := range selected {
		params := json.RawMessage(`{"type":"object","properties":{}}`)
		if schema := strings.TrimSpace(t.GetParametersJsonSchema()); schema != "" {
			var js json.RawMessage
			if err := json.Unmarshal([]byte(schema), &js); err == nil {
				params = js
			}
		}
		result = append(result, toolDefinition{
			Type: "function",
			Function: toolFunction{
				Name:        t.GetName(),
				Description: t.GetDescription(),
				Parameters:  params,
			},
		})
	}
	return result
}

func extractToolMeta(args string) (string, string) {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(args), &parsed); err != nil {
		return args, ""
	}
	var about, path string
	if p, ok := parsed["path"].(string); ok {
		path = p
	} else if f, ok := parsed["file"].(string); ok {
		path = f
	}
	if q, ok := parsed["query"].(string); ok {
		about = q
	} else if a, ok := parsed["about"].(string); ok {
		about = a
	} else {
		about = args
	}
	return about, path
}

func callChat(
	ctx context.Context,
	client *http.Client,
	baseURL, key, model string,
	messages []chatMessage,
	tools []toolDefinition,
) (*chatMessage, error) {
	endpoint := strings.TrimSuffix(baseURL, "/") + "/chat/completions"

	body, err := json.Marshal(chatRequest{
		Model:       model,
		Temperature: 0,
		Messages:    messages,
		Tools:       tools,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal chat request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create chat request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+key)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send chat request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read chat response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal chat response: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no completion returned")
	}
	return &chatResp.Choices[0].Message, nil
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
	systemPrompt, tools := loadManifest("manifest.json")

	agentSvc := &agentServer{
		systemPrompt: systemPrompt,
		tools:        tools,
		modelName:    os.Getenv("NUMEN_AGENT_MODEL"),
	}

	s := server.New(server.WithUpdateConfig(agentSvc.updateConfig))
	s.Register(numenv1connect.NewAgentServiceHandler(agentSvc, s.HandlerOptions()...))
	s.Register(numenv1connect.NewCompletionServiceHandler(agentSvc, s.HandlerOptions()...))

	if err := s.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
