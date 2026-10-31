package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/toolcall"
)

// ProgramName is the identifier a task uses to select the agent, and the key of its settings section.
const ProgramName = "numen"

// programTitle is the agent's display name.
const programTitle = "Numen Agent"

// The settings Configure reads.
const (
	SettingBaseURL     = "baseUrl"
	SettingModel       = "model"
	SettingAPIKey      = "apiKey"
	SettingTemperature = "temperature"
	SettingMaxTokens   = "max_tokens"
)

// modelsLimit is how long the chat API has to list its models.
const modelsLimit = 15 * time.Second

var errClosing = errors.New("this agent is closing")

// Server is numen's own agent: it works a task by calling an OpenAI-compatible chat API and calling
// the vault's tools over MCP on the model's behalf. It implements AgentService, in a plugin's process
// or linked into the process that hosts the vault. The zero value uses the default chat API with no
// key.
type Server struct {
	agentrun.UnimplementedAgentServiceHandler

	// Client sends the requests to the chat API. Nil sends them with a client of its own.
	Client *http.Client

	mu            sync.Mutex
	baseURL       string
	key           string
	model         string
	temperature   *float64
	maxTokens     *int
	conversations map[string][]message
	models        []model
	isListed      bool

	registry agentrun.Registry
}

// Configure applies the settings: the base URL of the chat API, the key sent to it and the model
// used when a task names none. A setting absent from the values keeps its current value, and an empty
// key means no key.
func (s *Server) Configure(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if base, ok := values[SettingBaseURL]; ok {
		s.baseURL = base
	}
	if key, ok := values[SettingAPIKey]; ok {
		s.key = key
	}
	if name, ok := values[SettingModel]; ok {
		s.model = name
	}
	if tempStr, ok := values[SettingTemperature]; ok && tempStr != "" {
		if val, err := strconv.ParseFloat(tempStr, 64); err == nil {
			s.temperature = &val
		}
	}
	if maxStr, ok := values[SettingMaxTokens]; ok && maxStr != "" {
		if val, err := strconv.Atoi(maxStr); err == nil {
			s.maxTokens = &val
		}
	}
	s.models, s.isListed = nil, false
}

// getChat is the chat API as the settings name it.
func (s *Server) getChat() chat {
	s.mu.Lock()
	defer s.mu.Unlock()
	return chat{
		baseURL:     s.baseURL,
		key:         s.key,
		client:      s.Client,
		temperature: s.temperature,
		maxTokens:   s.maxTokens,
	}
}

// getModel returns the model to use: the one named, else the configured one, else the default.
func (s *Server) getModel(named string) string {
	if named != "" {
		return named
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.model != "" {
		return s.model
	}
	return defaultModel
}

// AskAgent works the task and reports each request to the model, its output, each tool call
// and the stop reason. A conversation continues from the transcript stored for it.
func (s *Server) AskAgent(
	ctx context.Context,
	req *connect.Request[agentrun.AskAgentRequest],
	stream *connect.ServerStream[agentrun.AskAgentResponse],
) error {
	return s.Ask(ctx, req.Msg, stream.Send)
}

// Ask is AskAgent with the steps going to send.
func (s *Server) Ask(ctx context.Context, task *agentrun.AskAgentRequest, send func(*agentrun.AskAgentResponse) error) (err error) {
	taskCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	conversation := task.GetConversation()
	work := agentrun.NewWork(conversation, 0, cancel)
	if !s.registry.Add(work) {
		return connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer s.registry.Remove(work)
	defer work.End()

	tools, err := openToolbox(taskCtx, task)
	if err != nil {
		if taskCtx.Err() != nil {
			return nil
		}
		return send(&agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponseStopped{
			Stopped: fmt.Sprintf("the agent could not reach this vault: %v", err),
		}})
	}
	defer func() { err = errors.Join(err, tools.close()) }()

	history := s.getConversation(conversation)
	messages := make([]message, 0, len(history)+2)
	if instructions := task.GetInstructions(); instructions != "" {
		messages = append(messages, message{Role: "system", Content: newText(instructions), ToolCalls: nil, ToolCallID: ""})
	}
	messages = append(messages, history...)
	messages = append(messages, message{Role: "user", Content: newText(task.GetPrompt()), ToolCalls: nil, ToolCallID: ""})

	modelName := s.getModel(task.GetModel())
	taskRun := &run{
		chat:    s.getChat(),
		model:   modelName,
		effort:  s.getEffort(taskCtx, modelName, task.GetReasoning()),
		tools:   tools,
		words:   toolcall.IndexDeclarations(task.GetToolDeclarations()),
		turns:   int(task.GetMaxSteps()),
		send:    send,
		isPlain: false,
	}
	transcript, reason, err := taskRun.answer(taskCtx, messages)
	if taskCtx.Err() != nil {
		return nil
	}
	if failed := (*chatFailure)(nil); errors.As(err, &failed) {
		reason = failed.Error()
	} else if err != nil {
		return err
	}
	if reason == reasonDone || reason == reasonTurns || reason == reasonCut {
		s.appendConversation(conversation, transcript[len(messages)-1:])
	}
	return send(&agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponseStopped{Stopped: reason}})
}

// getConversation is the transcript in the conversation so far, oldest first.
func (s *Server) getConversation(conversation string) []message {
	if conversation == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]message(nil), s.conversations[conversation]...)
}

// appendConversation adds the transcript in one task to the conversation. A task that named no
// conversation keeps nothing.
func (s *Server) appendConversation(conversation string, transcript []message) {
	if conversation == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conversations == nil {
		s.conversations = map[string][]message{}
	}
	s.conversations[conversation] = append(s.conversations[conversation], transcript...)
}

// FinishConversation stops what is still running in the conversation and discards its
// history.
func (s *Server) FinishConversation(
	ctx context.Context,
	req *connect.Request[agentrun.FinishConversationRequest],
) (*connect.Response[agentrun.FinishConversationResponse], error) {
	conversation := req.Msg.GetConversation()
	if err := s.registry.EndConversation(ctx, conversation); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.mu.Lock()
	delete(s.conversations, conversation)
	s.mu.Unlock()
	return connect.NewResponse(&agentrun.FinishConversationResponse{}), nil
}

// GetAgentState is unreachable while the chat API does not list its models.
func (s *Server) GetAgentState(
	ctx context.Context,
	_ *connect.Request[agentrun.GetAgentStateRequest],
) (*connect.Response[agentrun.GetAgentStateResponse], error) {
	res := &agentrun.GetAgentStateResponse{Unreachable: ""}
	if _, err := s.listModels(ctx); err != nil {
		res.Unreachable = fmt.Sprintf("the chat API cannot be reached: %v", err)
	}
	return connect.NewResponse(res), nil
}

// GetAgentPrograms lists the one program this agent runs: the chat API the settings name, installed
// while it lists its models, with those models and the configured one as the default. It runs only
// the vault tools a task names.
func (s *Server) GetAgentPrograms(
	ctx context.Context,
	_ *connect.Request[agentrun.GetAgentProgramsRequest],
) (*connect.Response[agentrun.GetAgentProgramsResponse], error) {
	models, err := s.listModels(ctx)
	configured := s.getModel("")
	program := &agentrun.AgentProgram{
		Name:        ProgramName,
		Title:       programTitle,
		IsInstalled: err == nil,
		LimitsTools: true,
		IsDefault:   false,
		Models:      nil,
	}
	hasConfigured := false
	for _, one := range models {
		if one.Name == configured {
			hasConfigured = true
		}
		program.Models = append(program.Models, &agentrun.AgentModel{
			Name: one.Name, Title: one.Title, IsDefault: one.Name == configured,
			ReasoningLevels: reasoning.ToProto(one.ReasoningLevels), DefaultReasoning: one.DefaultReasoning,
		})
	}
	if !hasConfigured && configured != "" && len(models) > 0 {
		program.Models = append(program.Models, &agentrun.AgentModel{
			Name: configured, Title: configured, IsDefault: true,
			ReasoningLevels: nil, DefaultReasoning: "",
		})
	}
	return connect.NewResponse(&agentrun.GetAgentProgramsResponse{Programs: []*agentrun.AgentProgram{program}}), nil
}

// listModels is what the chat API lists: what it listed under the present settings, or what it
// lists now.
func (s *Server) listModels(ctx context.Context) ([]model, error) {
	s.mu.Lock()
	if s.isListed {
		defer s.mu.Unlock()
		return s.models, nil
	}
	request := chat{baseURL: s.baseURL, key: s.key, client: s.Client, temperature: nil, maxTokens: nil}
	s.mu.Unlock()

	listing, stop := context.WithTimeout(ctx, modelsLimit)
	defer stop()
	models, err := request.listModels(listing)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// A failure triggers a new request, and a tool list fetched under settings that have since changed is not cached.
	s.mu.Lock()
	defer s.mu.Unlock()
	if err == nil && s.baseURL == request.baseURL && s.key == request.key {
		s.models, s.isListed = models, true
	}
	return models, err
}

// Close stops every running task and rejects new ones.
func (s *Server) Close() error {
	return s.registry.Close()
}

// getEffort returns the reasoning depth requested from the model the chat API lists: none when the task requests
// none, the API does not list the model, or the model takes no depth.
func (s *Server) getEffort(ctx context.Context, name, level string) effort {
	if level == "" {
		return effort{field: "", level: ""}
	}
	models, err := s.listModels(ctx)
	if err != nil {
		return effort{field: "", level: ""}
	}
	for _, one := range models {
		if one.Name == name {
			return newEffort(one, level)
		}
	}
	return effort{field: "", level: ""}
}
