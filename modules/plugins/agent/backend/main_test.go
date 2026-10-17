package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	if len(toolsResp.Msg.GetTools()) != 2 {
		t.Errorf("expected 2 tools, got %d", len(toolsResp.Msg.GetTools()))
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
