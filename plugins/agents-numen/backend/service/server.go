package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/toolcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/rpcerror"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
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
	SettingMaxTokens   = "maxTokens"
	SettingMaxSteps    = "maxSteps"
)

// modelsLimit is how long the chat API has to list its models.
const modelsLimit = 15 * time.Second

var errClosing = errors.New("this agent is closing")

// Server is numen's own agent: it works a task by calling an OpenAI-compatible chat API and calling
// the vault's tools over MCP on the model's behalf. It implements AgentService, in a plugin's process
// or linked into the process that hosts the vault. The zero value uses the default chat API with no
// key.
type Server struct {
	agentrun.Unimplemented

	Client *http.Client

	mu            sync.Mutex
	baseURL       string
	key           string
	model         string
	temperature   *float64
	maxTokens     *int
	maxSteps      int
	conversations map[string][]message
	models        []model
	isListed      bool

	registry agentrun.Registry
}

// Configure applies the settings: the base URL of the chat API and the model used when a task names
// none. A setting absent from the values keeps its current value.
func (s *Server) Configure(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if base, ok := values[SettingBaseURL]; ok {
		s.baseURL = base
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
	if val, ok := readPositive(values[SettingMaxSteps]); ok {
		s.maxSteps = val
	}
	s.models, s.isListed = nil, false
}

// ApplySecrets applies the key sent to the chat API: the first secret of the list the host passes for
// the declaration apiKey. An empty list means no key, and a call that does not carry the declaration
// keeps the current key.
func (s *Server) ApplySecrets(secrets server.Secrets) {
	list, carried := server.ReadSecrets(secrets, SettingAPIKey)
	if !carried {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.key = ""
	if len(list) > 0 {
		s.key = list[0].Value
	}
	s.models, s.isListed = nil, false
}

// readPositive reads a whole number above zero from a setting value.
func readPositive(text string) (int, bool) {
	val, err := strconv.Atoi(text)
	return val, err == nil && val > 0
}

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

func (s *Server) getMaxSteps(task *agentrun.Request) int {
	if task.MaxSteps > 0 {
		return int(task.MaxSteps)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.maxSteps
}

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

// Ask works the task and reports each request to the model, its output, each tool call and the
// stop reason to send. A conversation continues from the transcript stored for it.
func (s *Server) Ask(ctx context.Context, task *agentrun.Request, send func(*agentrun.Response) error) (err error) {
	taskCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	conversation := task.Conversation
	work := agentrun.NewWork(conversation, 0, agentrun.StopByCancel(cancel))
	if !s.registry.Add(work) {
		return rpcerror.New(rpcerror.Unavailable, errClosing)
	}
	defer s.registry.Remove(work)
	defer work.End()

	tools, err := openToolbox(taskCtx, task)
	if err != nil {
		if taskCtx.Err() != nil {
			return nil
		}
		return send(&agentrun.Response{Step: &agentrun.StoppedStep{
			Stopped: fmt.Sprintf("the agent could not reach this vault: %v", err),
		}})
	}
	defer func() { err = errors.Join(err, tools.close()) }()

	history := s.getConversation(conversation)
	messages := make([]message, 0, len(history)+2)
	if instructions := task.Instructions; instructions != "" {
		messages = append(messages, message{Role: "system", Content: newText(instructions), ToolCalls: nil, ToolCallID: ""})
	}
	messages = append(messages, history...)
	messages = append(messages, message{Role: "user", Content: newText(task.Prompt), ToolCalls: nil, ToolCallID: ""})

	modelName := s.getModel(task.Model)
	taskRun := &run{
		chat:    s.getChat(),
		model:   modelName,
		effort:  s.getEffort(taskCtx, modelName, task.Reasoning),
		tools:   tools,
		words:   toolcall.IndexDeclarations(task.ToolDeclarations),
		turns:   s.getMaxSteps(task),
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
	return send(&agentrun.Response{Step: &agentrun.StoppedStep{Stopped: reason}})
}

func (s *Server) getConversation(conversation string) []message {
	if conversation == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]message(nil), s.conversations[conversation]...)
}

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
func (s *Server) FinishConversation(ctx context.Context, conversation string) error {
	if err := s.registry.EndConversation(ctx, conversation); err != nil {
		return rpcerror.New(rpcerror.Internal, err)
	}
	s.mu.Lock()
	delete(s.conversations, conversation)
	s.mu.Unlock()
	return nil
}

// GetState is unreachable while the chat API does not list its models.
func (s *Server) GetState(ctx context.Context) (*agentrun.State, error) {
	state := &agentrun.State{Unreachable: ""}
	if _, err := s.listModels(ctx); err != nil {
		state.Unreachable = fmt.Sprintf("the chat API cannot be reached: %v", err)
	}
	return state, nil
}

// GetPrograms lists the one program this agent runs: the chat API the settings name, installed
// while it lists its models, with those models and the configured one as the default. It runs only
// the vault tools a task names.
func (s *Server) GetPrograms(ctx context.Context, _ []agentrun.ProgramCommand) ([]*agentrun.Program, error) {
	models, err := s.listModels(ctx)
	configured := s.getModel("")
	program := &agentrun.Program{
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
		program.Models = append(program.Models, &reasoning.Model{
			Name: one.Name, Title: one.Title, IsDefault: one.Name == configured,
			ReasoningLevels: one.ReasoningLevels, DefaultReasoning: one.DefaultReasoning,
		})
	}
	if !hasConfigured && configured != "" && len(models) > 0 {
		program.Models = append(program.Models, &reasoning.Model{
			Name: configured, Title: configured, IsDefault: true,
			ReasoningLevels: nil, DefaultReasoning: "",
		})
	}
	return []*agentrun.Program{program}, nil
}

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
