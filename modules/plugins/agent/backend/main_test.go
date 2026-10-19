package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

func TestAgentServerRPC(t *testing.T) {
	agentSvc := &agentServer{
		systemPrompt: "You are an assistant.",
		tools: []*numenv1.AgentToolDefinition{
			{Name: "initial_tool", Description: "Initial tool description"},
		},
	}

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(agentSvc))

	server := httptest.NewServer(mux)
	defer server.Close()

	client := numenv1connect.NewAgentServiceClient(http.DefaultClient, server.URL)
	ctx := context.Background()

	// Test GetAgentState
	stateResp, err := client.GetAgentState(ctx, connect.NewRequest(&numenv1.GetAgentStateRequest{}))
	if err != nil {
		t.Fatalf("GetAgentState failed: %v", err)
	}
	if stateResp.Msg.GetUnreachable() != "" {
		t.Errorf("expected empty unreachable, got %q", stateResp.Msg.GetUnreachable())
	}

	// Test RegisterTool
	regResp, err := client.RegisterTool(ctx, connect.NewRequest(&numenv1.RegisterToolRequest{
		Tool: &numenv1.AgentToolDefinition{
			Name:        "new_tool",
			Description: "New tool description",
		},
	}))
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}
	if !regResp.Msg.GetSuccess() {
		t.Errorf("expected success true")
	}

	// Test ExtendSystemPrompt
	extResp, err := client.ExtendSystemPrompt(ctx, connect.NewRequest(&numenv1.ExtendSystemPromptRequest{
		Extension: " Extra rules.",
	}))
	if err != nil {
		t.Fatalf("ExtendSystemPrompt failed: %v", err)
	}
	if extResp.Msg.GetFullPrompt() != "You are an assistant. Extra rules." {
		t.Errorf("unexpected full prompt: %q", extResp.Msg.GetFullPrompt())
	}

	// Test GetAgentTools
	toolsResp, err := client.GetAgentTools(ctx, connect.NewRequest(&numenv1.GetAgentToolsRequest{}))
	if err != nil {
		t.Fatalf("GetAgentTools failed: %v", err)
	}
	tools := toolsResp.Msg.GetTools()
	if len(tools) != 2 || tools[0] == nil || tools[0].GetName() != "initial_tool" || tools[1] == nil || tools[1].GetName() != "new_tool" {
		t.Errorf("unexpected tools: %+v", tools)
	}
	if toolsResp.Msg.GetSystemPromptExtension() != "You are an assistant. Extra rules." {
		t.Errorf("unexpected prompt extension: %q", toolsResp.Msg.GetSystemPromptExtension())
	}

	// Test AskAgent stream
	stream, err := client.AskAgent(ctx, connect.NewRequest(&numenv1.AskAgentRequest{
		Prompt: "Hello assistant",
	}))
	if err != nil {
		t.Fatalf("AskAgent failed: %v", err)
	}

	var hasModelStart, hasSaid, hasStop bool
	for stream.Receive() {
		msg := stream.Msg()
		switch msg.GetStep().(type) {
		case *numenv1.AskAgentResponse_ModelStarted:
			hasModelStart = true
		case *numenv1.AskAgentResponse_Said:
			hasSaid = true
		case *numenv1.AskAgentResponse_Stopped:
			hasStop = true
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}
	if !hasModelStart {
		t.Errorf("expected ModelStarted step")
	}
	if !hasSaid {
		t.Errorf("expected Said step")
	}
	if !hasStop {
		t.Errorf("expected Stopped step")
	}

	// Test FinishConversation
	finResp, err := client.FinishConversation(ctx, connect.NewRequest(&numenv1.FinishConversationRequest{
		Conversation: "conv-1",
	}))
	if err != nil {
		t.Fatalf("FinishConversation failed: %v", err)
	}
	if finResp == nil {
		t.Errorf("expected non-nil response")
	}
}

func TestLifecycleServer(t *testing.T) {
	shutdownCh := make(chan struct{})
	life := server.NewLifecycleServer(shutdownCh)

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(life))

	server := httptest.NewServer(mux)
	defer server.Close()

	client := numenv1connect.NewPluginLifecycleServiceClient(http.DefaultClient, server.URL)
	ctx := context.Background()

	pingResp, err := client.Ping(ctx, connect.NewRequest(&numenv1.PingRequest{}))
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if pingResp.Msg.GetStatus() != "ok" {
		t.Errorf("unexpected status: %s", pingResp.Msg.GetStatus())
	}
}

func TestToolCallingExecutionLoop(t *testing.T) {
	var requestCount int
	var capturedRequests []chatRequest

	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}

		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		capturedRequests = append(capturedRequests, req)
		requestCount++

		w.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			if len(req.Tools) == 0 {
				t.Errorf("expected tools in request 1, got 0")
			}
			resp := chatResponse{
				Choices: []struct {
					Message      chatMessage `json:"message"`
					FinishReason string      `json:"finish_reason"`
				}{
					{
						FinishReason: "tool_calls",
						Message: chatMessage{
							Role: "assistant",
							ToolCalls: []toolCall{
								{
									ID:   "call_101",
									Type: "function",
									Function: toolCallFunction{
										Name:      "knowledge_search",
										Arguments: `{"query":"quantum physics","path":"notes/quantum.md"}`,
									},
								},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if len(req.Messages) < 4 {
			t.Errorf("expected at least 4 messages in request 2, got %d", len(req.Messages))
		} else {
			lastMsg := req.Messages[len(req.Messages)-1]
			if lastMsg.Role != "tool" {
				t.Errorf("expected last message role to be 'tool', got %q", lastMsg.Role)
			}
			if lastMsg.ToolCallID != "call_101" {
				t.Errorf("expected tool_call_id to be 'call_101', got %q", lastMsg.ToolCallID)
			}
			if lastMsg.Content == nil || *lastMsg.Content == "" {
				t.Errorf("expected non-empty tool message content")
			}
		}

		content := "Quantum physics explains matter at microscopic scale."
		resp := chatResponse{
			Choices: []struct {
				Message      chatMessage `json:"message"`
				FinishReason string      `json:"finish_reason"`
			}{
				{
					FinishReason: "stop",
					Message: chatMessage{
						Role:    "assistant",
						Content: &content,
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockLLM.Close()

	agentSvc := &agentServer{
		systemPrompt: "You are a physics expert.",
		baseURL:      mockLLM.URL,
		apiKey:       "test-key",
		tools: []*numenv1.AgentToolDefinition{
			{
				Name:                 "knowledge_search",
				Description:          "Searches notes in the vault",
				ParametersJsonSchema: `{"type":"object","properties":{"query":{"type":"string"}}}`,
			},
		},
	}

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(agentSvc))

	server := httptest.NewServer(mux)
	defer server.Close()

	client := numenv1connect.NewAgentServiceClient(http.DefaultClient, server.URL)
	ctx := context.Background()

	stream, err := client.AskAgent(ctx, connect.NewRequest(&numenv1.AskAgentRequest{
		Prompt: "Tell me about quantum physics",
	}))
	if err != nil {
		t.Fatalf("AskAgent failed: %v", err)
	}

	var steps []string
	var toolCallReceived *numenv1.ToolCall
	var finalSaid string
	for stream.Receive() {
		msg := stream.Msg()
		switch s := msg.GetStep().(type) {
		case *numenv1.AskAgentResponse_ModelStarted:
			steps = append(steps, "model_started")
		case *numenv1.AskAgentResponse_ToolCall:
			steps = append(steps, "tool_call")
			toolCallReceived = s.ToolCall
		case *numenv1.AskAgentResponse_ToolFinished:
			steps = append(steps, "tool_finished")
		case *numenv1.AskAgentResponse_Said:
			steps = append(steps, "said")
			finalSaid += s.Said
		case *numenv1.AskAgentResponse_Stopped:
			steps = append(steps, "stopped:"+s.Stopped)
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}

	if requestCount != 2 {
		t.Errorf("expected 2 LLM requests, got %d", requestCount)
	}
	if toolCallReceived == nil || toolCallReceived.GetTool() != "knowledge_search" {
		t.Errorf("expected knowledge_search tool call, got %+v", toolCallReceived)
	}
	if toolCallReceived.GetAbout() != "quantum physics" {
		t.Errorf("expected about 'quantum physics', got %q", toolCallReceived.GetAbout())
	}
	if toolCallReceived.GetPath() != "notes/quantum.md" {
		t.Errorf("expected path 'notes/quantum.md', got %q", toolCallReceived.GetPath())
	}
	if finalSaid != "Quantum physics explains matter at microscopic scale." {
		t.Errorf("unexpected reply: %q", finalSaid)
	}

	expectedSteps := []string{"model_started", "tool_call", "tool_finished", "model_started", "said", "stopped:done"}
	if len(steps) != len(expectedSteps) {
		t.Fatalf("got steps %v, want %v", steps, expectedSteps)
	}
	for i, exp := range expectedSteps {
		if steps[i] != exp {
			t.Errorf("step %d: got %s, want %s", i, steps[i], exp)
		}
	}
}

func TestToolCallingCustomExecutor(t *testing.T) {
	var executedTool string
	var executedArgs string

	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		if len(req.Messages) > 0 && req.Messages[len(req.Messages)-1].Role != "tool" {
			resp := chatResponse{
				Choices: []struct {
					Message      chatMessage `json:"message"`
					FinishReason string      `json:"finish_reason"`
				}{
					{
						FinishReason: "tool_calls",
						Message: chatMessage{
							Role: "assistant",
							ToolCalls: []toolCall{
								{
									ID:   "call_custom",
									Type: "function",
									Function: toolCallFunction{
										Name:      "custom_calc",
										Arguments: `{"expr":"2+2"}`,
									},
								},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		content := "The calculation result is 4."
		resp := chatResponse{
			Choices: []struct {
				Message      chatMessage `json:"message"`
				FinishReason string      `json:"finish_reason"`
			}{
				{
					FinishReason: "stop",
					Message: chatMessage{
						Role:    "assistant",
						Content: &content,
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockLLM.Close()

	agentSvc := &agentServer{
		baseURL: mockLLM.URL,
		apiKey:  "test-key",
		tools: []*numenv1.AgentToolDefinition{
			{Name: "custom_calc", Description: "Calculate math expressions"},
		},
		executor: func(_ context.Context, name, args string) (string, error) {
			executedTool = name
			executedArgs = args
			return `{"result": 4}`, nil
		},
	}

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(agentSvc))

	server := httptest.NewServer(mux)
	defer server.Close()

	client := numenv1connect.NewAgentServiceClient(http.DefaultClient, server.URL)
	stream, err := client.AskAgent(context.Background(), connect.NewRequest(&numenv1.AskAgentRequest{
		Prompt: "Calculate 2+2",
	}))
	if err != nil {
		t.Fatalf("AskAgent failed: %v", err)
	}

	var reply string
	for stream.Receive() {
		if said, ok := stream.Msg().GetStep().(*numenv1.AskAgentResponse_Said); ok {
			reply += said.Said
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}

	if executedTool != "custom_calc" {
		t.Errorf("expected executedTool custom_calc, got %q", executedTool)
	}
	if executedArgs != `{"expr":"2+2"}` {
		t.Errorf("expected executedArgs '{\"expr\":\"2+2\"}', got %q", executedArgs)
	}
	if reply != "The calculation result is 4." {
		t.Errorf("unexpected reply: %q", reply)
	}
}

func TestToolCallingMaxTurnsLimit(t *testing.T) {
	var iterations int
	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		iterations++
		w.Header().Set("Content-Type", "application/json")
		resp := chatResponse{
			Choices: []struct {
				Message      chatMessage `json:"message"`
				FinishReason string      `json:"finish_reason"`
			}{
				{
					FinishReason: "tool_calls",
					Message: chatMessage{
						Role: "assistant",
						ToolCalls: []toolCall{
							{
								ID:   "call_loop",
								Type: "function",
								Function: toolCallFunction{
									Name:      "test_loop",
									Arguments: `{}`,
								},
							},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockLLM.Close()

	agentSvc := &agentServer{
		baseURL: mockLLM.URL,
		apiKey:  "test-key",
		tools: []*numenv1.AgentToolDefinition{
			{Name: "test_loop", Description: "Loop tool"},
		},
	}

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(agentSvc))

	server := httptest.NewServer(mux)
	defer server.Close()

	client := numenv1connect.NewAgentServiceClient(http.DefaultClient, server.URL)
	stream, err := client.AskAgent(context.Background(), connect.NewRequest(&numenv1.AskAgentRequest{
		Prompt:   "Start loop",
		MaxSteps: 3,
	}))
	if err != nil {
		t.Fatalf("AskAgent failed: %v", err)
	}

	var stoppedReason string
	for stream.Receive() {
		if stopped, ok := stream.Msg().GetStep().(*numenv1.AskAgentResponse_Stopped); ok {
			stoppedReason = stopped.Stopped
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}

	if iterations != 3 {
		t.Errorf("expected 3 iterations, got %d", iterations)
	}
	if stoppedReason != "max_turns_exceeded" {
		t.Errorf("expected stopped 'max_turns_exceeded', got %q", stoppedReason)
	}
}

func TestToolCallingDefaultMaxTurnsLimit(t *testing.T) {
	var iterations int
	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		iterations++
		w.Header().Set("Content-Type", "application/json")
		resp := chatResponse{
			Choices: []struct {
				Message      chatMessage `json:"message"`
				FinishReason string      `json:"finish_reason"`
			}{
				{
					FinishReason: "tool_calls",
					Message: chatMessage{
						Role: "assistant",
						ToolCalls: []toolCall{
							{
								ID:   "call_loop",
								Type: "function",
								Function: toolCallFunction{
									Name:      "test_loop",
									Arguments: `{}`,
								},
							},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockLLM.Close()

	agentSvc := &agentServer{
		baseURL: mockLLM.URL,
		apiKey:  "test-key",
		tools: []*numenv1.AgentToolDefinition{
			{Name: "test_loop", Description: "Loop tool"},
		},
	}

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(agentSvc))

	server := httptest.NewServer(mux)
	defer server.Close()

	client := numenv1connect.NewAgentServiceClient(http.DefaultClient, server.URL)
	stream, err := client.AskAgent(context.Background(), connect.NewRequest(&numenv1.AskAgentRequest{
		Prompt: "Start loop with default max steps",
	}))
	if err != nil {
		t.Fatalf("AskAgent failed: %v", err)
	}

	var stoppedReason string
	for stream.Receive() {
		if stopped, ok := stream.Msg().GetStep().(*numenv1.AskAgentResponse_Stopped); ok {
			stoppedReason = stopped.Stopped
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}

	if iterations != 10 {
		t.Errorf("expected 10 iterations, got %d", iterations)
	}
	if stoppedReason != "max_turns_exceeded" {
		t.Errorf("expected stopped 'max_turns_exceeded', got %q", stoppedReason)
	}
}

func TestToolCallingMultipleToolsInTurn(t *testing.T) {
	var requestCount int
	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			resp := chatResponse{
				Choices: []struct {
					Message      chatMessage `json:"message"`
					FinishReason string      `json:"finish_reason"`
				}{
					{
						FinishReason: "tool_calls",
						Message: chatMessage{
							Role: "assistant",
							ToolCalls: []toolCall{
								{
									ID:   "call_1",
									Type: "function",
									Function: toolCallFunction{
										Name:      "tool_a",
										Arguments: `{"arg":"a"}`,
									},
								},
								{
									ID:   "call_2",
									Type: "function",
									Function: toolCallFunction{
										Name:      "tool_b",
										Arguments: `{"arg":"b"}`,
									},
								},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Verify both tool responses were provided
		var toolMessages []chatMessage
		for _, m := range req.Messages {
			if m.Role == "tool" {
				toolMessages = append(toolMessages, m)
			}
		}
		if len(toolMessages) != 2 {
			t.Errorf("expected 2 tool messages in request 2, got %d", len(toolMessages))
		}
		if len(toolMessages) == 2 {
			if toolMessages[0].ToolCallID != "call_1" || toolMessages[1].ToolCallID != "call_2" {
				t.Errorf("unexpected tool call IDs: %s, %s", toolMessages[0].ToolCallID, toolMessages[1].ToolCallID)
			}
		}

		content := "Both tools executed successfully."
		resp := chatResponse{
			Choices: []struct {
				Message      chatMessage `json:"message"`
				FinishReason string      `json:"finish_reason"`
			}{
				{
					FinishReason: "stop",
					Message: chatMessage{
						Role:    "assistant",
						Content: &content,
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockLLM.Close()

	agentSvc := &agentServer{
		baseURL: mockLLM.URL,
		apiKey:  "test-key",
		tools: []*numenv1.AgentToolDefinition{
			{Name: "tool_a", Description: "Tool A"},
			{Name: "tool_b", Description: "Tool B"},
		},
	}

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(agentSvc))

	server := httptest.NewServer(mux)
	defer server.Close()

	client := numenv1connect.NewAgentServiceClient(http.DefaultClient, server.URL)
	stream, err := client.AskAgent(context.Background(), connect.NewRequest(&numenv1.AskAgentRequest{
		Prompt: "Run both tools",
	}))
	if err != nil {
		t.Fatalf("AskAgent failed: %v", err)
	}

	var toolCallsCount int
	var toolFinishedCount int
	for stream.Receive() {
		switch stream.Msg().GetStep().(type) {
		case *numenv1.AskAgentResponse_ToolCall:
			toolCallsCount++
		case *numenv1.AskAgentResponse_ToolFinished:
			toolFinishedCount++
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}

	if toolCallsCount != 2 {
		t.Errorf("expected 2 ToolCall events, got %d", toolCallsCount)
	}
	if toolFinishedCount != 2 {
		t.Errorf("expected 2 ToolFinished events, got %d", toolFinishedCount)
	}
}

type mockSearchServiceHandler struct {
	numenv1connect.UnimplementedSearchServiceHandler
}

func (m *mockSearchServiceHandler) SearchPassages(
	_ context.Context,
	req *connect.Request[numenv1.SearchPassagesRequest],
) (*connect.Response[numenv1.SearchPassagesResponse], error) {
	return connect.NewResponse(&numenv1.SearchPassagesResponse{
		Found: []*numenv1.Passage{
			{
				Path: "notes/gravity.md",
				Text: "Gravity is a fundamental interaction: " + req.Msg.GetQuery(),
			},
		},
	}), nil
}

func TestToolCallingConnectRPCSearch(t *testing.T) {
	searchMux := http.NewServeMux()
	searchMux.Handle(numenv1connect.NewSearchServiceHandler(&mockSearchServiceHandler{}))
	searchServer := httptest.NewServer(searchMux)
	defer searchServer.Close()

	t.Setenv("NUMEN_SEARCH_URL", searchServer.URL)

	var searchOutputReceived string
	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		if len(req.Messages) > 0 && req.Messages[len(req.Messages)-1].Role != "tool" {
			resp := chatResponse{
				Choices: []struct {
					Message      chatMessage `json:"message"`
					FinishReason string      `json:"finish_reason"`
				}{
					{
						FinishReason: "tool_calls",
						Message: chatMessage{
							Role: "assistant",
							ToolCalls: []toolCall{
								{
									ID:   "call_search_1",
									Type: "function",
									Function: toolCallFunction{
										Name:      "knowledge_search",
										Arguments: `{"query":"general relativity","limit":3}`,
									},
								},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		for _, m := range req.Messages {
			if m.Role == "tool" && m.Content != nil {
				searchOutputReceived = *m.Content
			}
		}

		content := "Gravity note found."
		resp := chatResponse{
			Choices: []struct {
				Message      chatMessage `json:"message"`
				FinishReason string      `json:"finish_reason"`
			}{
				{
					FinishReason: "stop",
					Message: chatMessage{
						Role:    "assistant",
						Content: &content,
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockLLM.Close()

	agentSvc := &agentServer{
		baseURL: mockLLM.URL,
		apiKey:  "test-key",
		tools: []*numenv1.AgentToolDefinition{
			{Name: "knowledge_search", Description: "Vault search"},
		},
	}

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(agentSvc))

	server := httptest.NewServer(mux)
	defer server.Close()

	client := numenv1connect.NewAgentServiceClient(http.DefaultClient, server.URL)
	stream, err := client.AskAgent(context.Background(), connect.NewRequest(&numenv1.AskAgentRequest{
		Prompt: "Find gravity notes",
	}))
	if err != nil {
		t.Fatalf("AskAgent failed: %v", err)
	}

	for stream.Receive() {
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}

	if !strings.Contains(searchOutputReceived, "notes/gravity.md") {
		t.Errorf("expected search output to contain 'notes/gravity.md', got %q", searchOutputReceived)
	}
	if !strings.Contains(searchOutputReceived, "general relativity") {
		t.Errorf("expected search output to contain query, got %q", searchOutputReceived)
	}
}

func TestChatTimeout(t *testing.T) {
	if chatTimeout != 60*time.Second {
		t.Errorf("chatTimeout = %v, want 60s", chatTimeout)
	}
	client := getHTTPClient(nil)
	if client.Timeout != 60*time.Second {
		t.Errorf("getHTTPClient(nil).Timeout = %v, want 60s", client.Timeout)
	}
	customClient := &http.Client{Timeout: 10 * time.Second}
	if got := getHTTPClient(customClient); got != customClient {
		t.Errorf("getHTTPClient(customClient) = %v, want %v", got, customClient)
	}
}

func TestUpdateConfig(t *testing.T) {
	svc := &agentServer{systemPrompt: "initial-prompt", modelName: "initial-model"}
	// Empty strings or missing keys should not overwrite
	svc.updateConfig(map[string]string{"system_prompt": "", "model": ""})
	if svc.systemPrompt != "initial-prompt" || svc.modelName != "initial-model" {
		t.Errorf("empty string overwrote config: prompt=%q, model=%q", svc.systemPrompt, svc.modelName)
	}
	// Non-empty values should update
	svc.updateConfig(map[string]string{"system_prompt": "new-prompt", "model": "new-model"})
	if svc.systemPrompt != "new-prompt" || svc.modelName != "new-model" {
		t.Errorf("non-empty string failed to update config: prompt=%q, model=%q", svc.systemPrompt, svc.modelName)
	}
}

func TestAskAgentInstructions(t *testing.T) {
	var seenMessages []chatMessage
	chat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		seenMessages = req.Messages
		content := "done"
		_ = json.NewEncoder(w).Encode(chatResponse{Choices: []struct {
			Message      chatMessage `json:"message"`
			FinishReason string      `json:"finish_reason"`
		}{{Message: chatMessage{Role: "assistant", Content: &content}}}})
	}))
	defer chat.Close()

	svc := &agentServer{systemPrompt: "base-sys"}
	svc.updateConfig(map[string]string{"apiKey": "k", "baseUrl": chat.URL})

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(svc))
	server := httptest.NewServer(mux)
	defer server.Close()

	client := numenv1connect.NewAgentServiceClient(http.DefaultClient, server.URL)
	stream, err := client.AskAgent(context.Background(), connect.NewRequest(&numenv1.AskAgentRequest{
		Prompt:       "hi",
		Instructions: "custom-instruction",
	}))
	if err != nil {
		t.Fatal(err)
	}
	for stream.Receive() {
	}
	if len(seenMessages) == 0 || seenMessages[0].Role != "system" || !strings.Contains(*seenMessages[0].Content, "custom-instruction\nbase-sys") {
		t.Fatalf("system message did not include instructions: %+v", seenMessages)
	}
}

func TestAskAgentMaxStepsOne(t *testing.T) {
	turns := 0
	chat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		turns++
		_ = json.NewEncoder(w).Encode(chatResponse{Choices: []struct {
			Message      chatMessage `json:"message"`
			FinishReason string      `json:"finish_reason"`
		}{{
			FinishReason: "tool_calls",
			Message: chatMessage{
				Role: "assistant",
				ToolCalls: []toolCall{{
					ID:       "call_1",
					Type:     "function",
					Function: toolCallFunction{Name: "dummy", Arguments: "{}"},
				}},
			},
		}}})
	}))
	defer chat.Close()

	svc := &agentServer{
		executor: func(ctx context.Context, name string, args string) (string, error) {
			return "ok", nil
		},
	}
	svc.updateConfig(map[string]string{"apiKey": "k", "baseUrl": chat.URL})

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(svc))
	server := httptest.NewServer(mux)
	defer server.Close()

	client := numenv1connect.NewAgentServiceClient(http.DefaultClient, server.URL)
	stream, err := client.AskAgent(context.Background(), connect.NewRequest(&numenv1.AskAgentRequest{
		Prompt:   "loop",
		MaxSteps: 1,
	}))
	if err != nil {
		t.Fatal(err)
	}
	var stoppedReason string
	for stream.Receive() {
		if s := stream.Msg().GetStopped(); s != "" {
			stoppedReason = s
		}
	}
	if turns != 1 {
		t.Fatalf("expected 1 turn with MaxSteps=1, got %d", turns)
	}
	if stoppedReason != "max_turns_exceeded" {
		t.Fatalf("expected stopped reason max_turns_exceeded, got %q", stoppedReason)
	}
}

func TestAskAgentChunkSize64(t *testing.T) {
	reply65 := strings.Repeat("a", 65)
	chat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(chatResponse{Choices: []struct {
			Message      chatMessage `json:"message"`
			FinishReason string      `json:"finish_reason"`
		}{{Message: chatMessage{Role: "assistant", Content: &reply65}}}})
	}))
	defer chat.Close()

	svc := &agentServer{}
	svc.updateConfig(map[string]string{"apiKey": "k", "baseUrl": chat.URL})

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(svc))
	server := httptest.NewServer(mux)
	defer server.Close()

	client := numenv1connect.NewAgentServiceClient(http.DefaultClient, server.URL)
	stream, err := client.AskAgent(context.Background(), connect.NewRequest(&numenv1.AskAgentRequest{Prompt: "hi"}))
	if err != nil {
		t.Fatal(err)
	}
	var chunks []string
	for stream.Receive() {
		if said := stream.Msg().GetSaid(); said != "" {
			chunks = append(chunks, said)
		}
	}
	if len(chunks) != 2 || len(chunks[0]) != 64 || len(chunks[1]) != 1 {
		t.Fatalf("expected chunks of len 64 and 1, got %v", chunks)
	}

	svcNoKey := &agentServer{}
	muxNoKey := http.NewServeMux()
	muxNoKey.Handle(numenv1connect.NewAgentServiceHandler(svcNoKey))
	serverNoKey := httptest.NewServer(muxNoKey)
	defer serverNoKey.Close()

	clientNoKey := numenv1connect.NewAgentServiceClient(http.DefaultClient, serverNoKey.URL)
	streamNoKey, err := clientNoKey.AskAgent(context.Background(), connect.NewRequest(&numenv1.AskAgentRequest{
		Prompt: strings.Repeat("x", 40),
	}))
	if err != nil {
		t.Fatal(err)
	}
	var chunksNoKey []string
	for streamNoKey.Receive() {
		if said := streamNoKey.Msg().GetSaid(); said != "" {
			chunksNoKey = append(chunksNoKey, said)
		}
	}
	if len(chunksNoKey) != 2 || len(chunksNoKey[0]) != 64 || len(chunksNoKey[1]) != 1 {
		t.Fatalf("expected fallback chunks of len 64 and 1, got %v", chunksNoKey)
	}
}

func TestAskAgentFallbackCustomTools(t *testing.T) {
	svc0 := &agentServer{}
	mux0 := http.NewServeMux()
	mux0.Handle(numenv1connect.NewAgentServiceHandler(svc0))
	s0 := httptest.NewServer(mux0)
	defer s0.Close()
	c0 := numenv1connect.NewAgentServiceClient(http.DefaultClient, s0.URL)
	st0, err := c0.AskAgent(context.Background(), connect.NewRequest(&numenv1.AskAgentRequest{Prompt: "test"}))
	if err != nil {
		t.Fatal(err)
	}
	var said0 string
	for st0.Receive() {
		said0 += st0.Msg().GetSaid()
	}
	if strings.Contains(said0, "custom tools available") {
		t.Errorf("expected no custom tools summary, got %q", said0)
	}

	svc1 := &agentServer{
		tools: []*numenv1.AgentToolDefinition{{Name: "t1"}},
	}
	mux1 := http.NewServeMux()
	mux1.Handle(numenv1connect.NewAgentServiceHandler(svc1))
	s1 := httptest.NewServer(mux1)
	defer s1.Close()
	c1 := numenv1connect.NewAgentServiceClient(http.DefaultClient, s1.URL)
	st1, err := c1.AskAgent(context.Background(), connect.NewRequest(&numenv1.AskAgentRequest{Prompt: "test"}))
	if err != nil {
		t.Fatal(err)
	}
	var said1 string
	for st1.Receive() {
		said1 += st1.Msg().GetSaid()
	}
	if !strings.Contains(said1, "(1 custom tools available)") {
		t.Errorf("expected (1 custom tools available), got %q", said1)
	}
}

func TestExecuteToolReleasesReadLock(t *testing.T) {
	svc := &agentServer{}
	_, _ = svc.executeTool(context.Background(), "unknown", "")
	if !svc.mu.TryLock() {
		t.Fatal("read lock was not released")
	}
	svc.mu.Unlock()
}

type testSearchHandler struct {
	numenv1connect.UnimplementedSearchServiceHandler
	gotQuery string
	gotLimit int32
}

func (h *testSearchHandler) SearchPassages(
	_ context.Context,
	req *connect.Request[numenv1.SearchPassagesRequest],
) (*connect.Response[numenv1.SearchPassagesResponse], error) {
	h.gotQuery = req.Msg.GetQuery()
	h.gotLimit = req.Msg.GetLimit()
	return connect.NewResponse(&numenv1.SearchPassagesResponse{
		Found: []*numenv1.Passage{{Text: "found"}},
	}), nil
}

func TestDefaultExecuteToolSearch(t *testing.T) {
	handler := &testSearchHandler{}
	searchMux := http.NewServeMux()
	searchMux.Handle(numenv1connect.NewSearchServiceHandler(handler))
	searchServer := httptest.NewServer(searchMux)
	defer searchServer.Close()

	svc := &agentServer{}

	t.Setenv("NUMEN_SEARCH_URL", "")
	if _, err := svc.defaultExecuteTool(context.Background(), "knowledge_search", `{"query":"q"}`); err == nil {
		t.Fatal("expected error when NUMEN_SEARCH_URL is empty")
	}

	t.Setenv("NUMEN_SEARCH_URL", searchServer.URL)

	_, err := svc.defaultExecuteTool(context.Background(), "search", `{"query":"q1","limit":1}`)
	if err != nil || handler.gotLimit != 1 || handler.gotQuery != "q1" {
		t.Fatalf("search tool failed: err=%v limit=%d query=%s", err, handler.gotLimit, handler.gotQuery)
	}

	_, err = svc.defaultExecuteTool(context.Background(), "knowledge_search", `{"query":"q2","limit":0}`)
	if err != nil || handler.gotLimit != 5 || handler.gotQuery != "q2" {
		t.Fatalf("knowledge_search tool with 0 limit failed: err=%v limit=%d query=%s", err, handler.gotLimit, handler.gotQuery)
	}

	_, err = svc.defaultExecuteTool(context.Background(), "search", "raw query string")
	if err != nil || handler.gotQuery != "raw query string" || handler.gotLimit != 5 {
		t.Fatalf("raw query failed: err=%v limit=%d query=%s", err, handler.gotLimit, handler.gotQuery)
	}

	_, err = svc.defaultExecuteTool(context.Background(), "search", `{"limit":3}`)
	if err != nil || handler.gotQuery != `{"limit":3}` || handler.gotLimit != 5 {
		t.Fatalf("json without query: got query=%q, limit=%d", handler.gotQuery, handler.gotLimit)
	}

	if _, err := svc.defaultExecuteTool(context.Background(), "unknown_tool", `{"query":"q"}`); err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestGetBaseURLAndModel(t *testing.T) {
	t.Setenv("NUMEN_AGENT_URL", "")
	if got := getBaseURL(""); got != defaultBaseURL {
		t.Errorf("getBaseURL(\"\") = %q, want %q", got, defaultBaseURL)
	}
	t.Setenv("NUMEN_AGENT_URL", "http://env.url")
	if got := getBaseURL(""); got != "http://env.url" {
		t.Errorf("getBaseURL(\"\") with env = %q, want http://env.url", got)
	}
	if got := getBaseURL("http://conf.url"); got != "http://conf.url" {
		t.Errorf("getBaseURL(\"http://conf.url\") = %q, want http://conf.url", got)
	}

	if got := getModel("", ""); got != defaultModel {
		t.Errorf("getModel(\"\", \"\") = %q, want %q", got, defaultModel)
	}
	if got := getModel("req", "conf"); got != "req" {
		t.Errorf("getModel(\"req\", \"conf\") = %q, want req", got)
	}
	if got := getModel("", "conf"); got != "conf" {
		t.Errorf("getModel(\"\", \"conf\") = %q, want conf", got)
	}
}

func TestFormatTools(t *testing.T) {
	tools := []*numenv1.AgentToolDefinition{
		{Name: "t1", ParametersJsonSchema: `{"type":"string"}`},
		{Name: "t2", ParametersJsonSchema: `invalid-json`},
		{Name: "t3"},
	}
	res := formatTools(tools, nil)
	if len(res) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(res))
	}
	if string(res[0].Function.Parameters) != `{"type":"string"}` {
		t.Errorf("expected parsed schema, got %s", res[0].Function.Parameters)
	}
	if string(res[1].Function.Parameters) != `{"type":"object","properties":{}}` {
		t.Errorf("expected default schema for invalid json, got %s", res[1].Function.Parameters)
	}

	res = formatTools(tools, []string{"t1"})
	if len(res) != 1 || res[0].Function.Name != "t1" {
		t.Fatalf("expected 1 tool t1, got %+v", res)
	}

	if res := formatTools(nil, nil); res != nil {
		t.Errorf("expected nil, got %+v", res)
	}

	if res := formatTools(tools, []string{"nonexistent"}); res != nil {
		t.Errorf("expected nil, got %+v", res)
	}
}

func TestExtractToolMeta(t *testing.T) {
	about, path := extractToolMeta("invalid json")
	if about != "invalid json" || path != "" {
		t.Errorf("invalid json: got about=%q, path=%q", about, path)
	}
	about, path = extractToolMeta(`{"file":"f.txt","query":"q"}`)
	if about != "q" || path != "f.txt" {
		t.Errorf("file/query: got about=%q, path=%q", about, path)
	}
	about, path = extractToolMeta(`{"path":"p.txt","about":"a"}`)
	if about != "a" || path != "p.txt" {
		t.Errorf("path/about: got about=%q, path=%q", about, path)
	}
	about, path = extractToolMeta(`{"key":"val"}`)
	if about != `{"key":"val"}` || path != "" {
		t.Errorf("other key: got about=%q, path=%q", about, path)
	}
}

type errReaderTransport struct{}

func (errReaderTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(iotest.ErrReader(errors.New("read failed"))),
	}, nil
}

func TestCallChat(t *testing.T) {
	var gotReq chatRequest
	var gotContentType string
	chat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		content := "response text"
		_ = json.NewEncoder(w).Encode(chatResponse{Choices: []struct {
			Message      chatMessage `json:"message"`
			FinishReason string      `json:"finish_reason"`
		}{{Message: chatMessage{Role: "assistant", Content: &content}}}})
	}))
	defer chat.Close()

	msg, err := callChat(context.Background(), http.DefaultClient, chat.URL, "key", "model", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if gotReq.Temperature != 0 {
		t.Errorf("Temperature = %v, want 0", gotReq.Temperature)
	}
	if msg.Content == nil || *msg.Content != "response text" {
		t.Errorf("unexpected message: %+v", msg)
	}

	_, err = callChat(context.Background(), http.DefaultClient, "http://[::1]:namedport", "key", "model", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "create chat request") {
		t.Errorf("expected create chat request error, got %v", err)
	}

	closedServer := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	closedServer.Close()
	_, err = callChat(context.Background(), http.DefaultClient, closedServer.URL, "key", "model", nil, nil)
	if err == nil || errors.Unwrap(err) == nil {
		t.Errorf("expected wrapped send chat request error, got %v", err)
	}

	errClient := &http.Client{Transport: errReaderTransport{}}
	_, err = callChat(context.Background(), errClient, chat.URL, "key", "model", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "read chat response") || errors.Unwrap(err) == nil {
		t.Errorf("expected wrapped read chat response error, got %v", err)
	}

	statusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		content := "err"
		_ = json.NewEncoder(w).Encode(chatResponse{Choices: []struct {
			Message      chatMessage `json:"message"`
			FinishReason string      `json:"finish_reason"`
		}{{Message: chatMessage{Role: "assistant", Content: &content}}}})
	}))
	defer statusServer.Close()
	_, err = callChat(context.Background(), http.DefaultClient, statusServer.URL, "key", "model", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Errorf("expected status 500 error, got %v", err)
	}

	badJSONServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}))
	defer badJSONServer.Close()
	_, err = callChat(context.Background(), http.DefaultClient, badJSONServer.URL, "key", "model", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "unmarshal chat response") || errors.Unwrap(err) == nil {
		t.Errorf("expected wrapped unmarshal chat response error, got %v", err)
	}

	emptyChoicesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer emptyChoicesServer.Close()
	_, err = callChat(context.Background(), http.DefaultClient, emptyChoicesServer.URL, "key", "model", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "no completion returned") {
		t.Errorf("expected no completion returned error, got %v", err)
	}
}

func TestSplitChunks(t *testing.T) {
	if got := splitChunks("", 5); len(got) != 1 || got[0] != "" {
		t.Errorf("splitChunks(\"\", 5) = %v, want [\"\"]", got)
	}
	if got := splitChunks("12345", 5); len(got) != 1 || got[0] != "12345" {
		t.Errorf("splitChunks(\"12345\", 5) = %v, want [\"12345\"]", got)
	}
	if got := splitChunks("123456", 5); len(got) != 2 || got[0] != "12345" || got[1] != "6" {
		t.Errorf("splitChunks(\"123456\", 5) = %v, want [\"12345\", \"6\"]", got)
	}
	if got := splitChunks("1234567890", 5); len(got) != 2 || got[0] != "12345" || got[1] != "67890" {
		t.Errorf("splitChunks(\"1234567890\", 5) = %v, want [\"12345\", \"67890\"]", got)
	}
	if got := splitChunks("12345678901", 5); len(got) != 3 || got[0] != "12345" || got[1] != "67890" || got[2] != "1" {
		t.Errorf("splitChunks(\"12345678901\", 5) = %v, want [\"12345\", \"67890\", \"1\"]", got)
	}
}

type failStreamInterceptor struct {
	failPredicate func(any) bool
	sendErr       error
}

func (i *failStreamInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return next
}

func (i *failStreamInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *failStreamInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return next(ctx, &failingStreamingHandlerConn{
			StreamingHandlerConn: conn,
			interceptor:          i,
		})
	}
}

type failingStreamingHandlerConn struct {
	connect.StreamingHandlerConn
	interceptor *failStreamInterceptor
}

func (c *failingStreamingHandlerConn) Send(msg any) error {
	if c.interceptor.failPredicate != nil && c.interceptor.failPredicate(msg) {
		return c.interceptor.sendErr
	}
	return c.StreamingHandlerConn.Send(msg)
}

func TestAskAgentStreamSendFailure(t *testing.T) {
	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := chatResponse{
			Choices: []struct {
				Message      chatMessage `json:"message"`
				FinishReason string      `json:"finish_reason"`
			}{
				{
					FinishReason: "tool_calls",
					Message: chatMessage{
						Role: "assistant",
						ToolCalls: []toolCall{
							{
								ID:   "call_err_1",
								Type: "function",
								Function: toolCallFunction{
									Name:      "test_tool",
									Arguments: `{"query":"test"}`,
								},
							},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockLLM.Close()

	var executorCalled bool
	agentSvc := &agentServer{
		systemPrompt: "You are a test assistant.",
		baseURL:      mockLLM.URL,
		apiKey:       "test-key",
		tools: []*numenv1.AgentToolDefinition{
			{
				Name:        "test_tool",
				Description: "Test tool description",
			},
		},
		executor: func(ctx context.Context, name string, args string) (string, error) {
			executorCalled = true
			return "executed", nil
		},
	}

	expectedErr := errors.New("simulated stream failure")
	interceptor := &failStreamInterceptor{
		failPredicate: func(msg any) bool {
			resp, ok := msg.(*numenv1.AskAgentResponse)
			if !ok {
				return false
			}
			_, ok = resp.GetStep().(*numenv1.AskAgentResponse_ToolCall)
			return ok
		},
		sendErr: expectedErr,
	}

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(agentSvc, connect.WithInterceptors(interceptor)))

	server := httptest.NewServer(mux)
	defer server.Close()

	client := numenv1connect.NewAgentServiceClient(http.DefaultClient, server.URL)
	stream, err := client.AskAgent(context.Background(), connect.NewRequest(&numenv1.AskAgentRequest{
		Prompt: "Trigger tool call",
	}))
	if err != nil {
		t.Fatalf("AskAgent request failed: %v", err)
	}

	for stream.Receive() {
	}

	if executorCalled {
		t.Fatal("executor was invoked after tool call send failure")
	}
	if streamErr := stream.Err(); streamErr == nil {
		t.Fatal("expected stream error, got nil")
	}
}
