package harness

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/completion"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/toolcall"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// ProgramName is what a task asks for the agent by, and what the settings keep its section under.
const ProgramName = "numen"

// programTitle is what the agent is called when shown to a person.
const programTitle = "Numen Agent"

// The settings Configure reads.
const (
	SettingBaseURL = "baseUrl"
	SettingModel   = "model"
	SettingAPIKey  = "apiKey"
)

// modelsLimit is how long the chat API has to list its models.
const modelsLimit = 15 * time.Second

var errClosing = errors.New("this agent is closing")

// Server is numen's own agent: it works a task by asking an OpenAI-compatible chat API and calling
// the vault's tools over MCP on the model's behalf. It answers AgentService and CompletionService,
// in a plugin's process or linked into the one that hosts the vault. The zero value asks the
// default chat API with no key.
type Server struct {
	numenv1connect.UnimplementedAgentServiceHandler
	numenv1connect.UnimplementedCompletionServiceHandler

	// Client sends the requests to the chat API. Nil sends them with a client of its own.
	Client *http.Client

	mu            sync.Mutex
	baseURL       string
	key           string
	model         string
	conversations map[string][]message
	models        []model
	isListed      bool

	registry agentrun.Registry
}

// Configure takes the settings: the base URL of the chat API, the key presented to it and the model
// asked where a task names none. A setting the values do not carry is left as it was, and an empty
// key is no key.
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
	s.models, s.isListed = nil, false
}

// getChat is the chat API as the settings name it.
func (s *Server) getChat() chat {
	s.mu.Lock()
	defer s.mu.Unlock()
	return chat{baseURL: s.baseURL, key: s.key, client: s.Client}
}

// getModel is the model asked: the one named, else the configured one, else the default.
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

// AskAgent works the task and reports each request to the model, what it says, each tool it calls
// and why it stopped. A conversation goes on from what was said in it before.
func (s *Server) AskAgent(
	ctx context.Context,
	req *connect.Request[numenv1.AskAgentRequest],
	stream *connect.ServerStream[numenv1.AskAgentResponse],
) error {
	return s.Ask(ctx, req.Msg, stream.Send)
}

// Ask is AskAgent with the steps going to send.
func (s *Server) Ask(ctx context.Context, task *numenv1.AskAgentRequest, send func(*numenv1.AskAgentResponse) error) error {
	running, cancel := context.WithCancel(ctx)
	defer cancel()

	conversation := task.GetConversation()
	work := agentrun.NewWork(conversation, 0, cancel)
	if !s.registry.Add(work) {
		return connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer s.registry.Remove(work)
	defer work.End()

	tools, err := openToolbox(running, task.GetToolsUrl(), task.GetToolsToken(), task.GetTools())
	if err != nil {
		if running.Err() != nil {
			return nil
		}
		return send(&numenv1.AskAgentResponse{Step: &numenv1.AskAgentResponse_Stopped{
			Stopped: fmt.Sprintf("the agent could not reach this vault: %v", err),
		}})
	}
	defer tools.close()

	held := s.getConversation(conversation)
	messages := make([]message, 0, len(held)+2)
	if instructions := task.GetInstructions(); instructions != "" {
		messages = append(messages, message{Role: "system", Content: newText(instructions)})
	}
	messages = append(messages, held...)
	messages = append(messages, message{Role: "user", Content: newText(task.GetPrompt())})

	worked := &run{
		chat:  s.getChat(),
		model: s.getModel(task.GetModel()),
		tools: tools,
		words: toolcall.ReadWords(task.GetToolDeclarations()),
		turns: int(task.GetMaxSteps()),
		send:  send,
	}
	said, reason, err := worked.answer(running, messages)
	if running.Err() != nil {
		return nil
	}
	if failed := (*chatFailure)(nil); errors.As(err, &failed) {
		reason = failed.Error()
	} else if err != nil {
		return err
	}
	if reason == reasonDone || reason == reasonTurns || reason == reasonCut {
		s.keepConversation(conversation, said[len(messages)-1:])
	}
	return send(&numenv1.AskAgentResponse{Step: &numenv1.AskAgentResponse_Stopped{Stopped: reason}})
}

// getConversation is what was said in the conversation so far, oldest first.
func (s *Server) getConversation(conversation string) []message {
	if conversation == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]message(nil), s.conversations[conversation]...)
}

// keepConversation adds what was said in one task to the conversation. A task that named no
// conversation keeps nothing.
func (s *Server) keepConversation(conversation string, said []message) {
	if conversation == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conversations == nil {
		s.conversations = map[string][]message{}
	}
	s.conversations[conversation] = append(s.conversations[conversation], said...)
}

// FinishConversation stops what is still being answered in the conversation and forgets what was
// said in it.
func (s *Server) FinishConversation(
	ctx context.Context,
	req *connect.Request[numenv1.FinishConversationRequest],
) (*connect.Response[numenv1.FinishConversationResponse], error) {
	conversation := req.Msg.GetConversation()
	if err := s.registry.EndConversation(ctx, conversation); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.mu.Lock()
	delete(s.conversations, conversation)
	s.mu.Unlock()
	return connect.NewResponse(&numenv1.FinishConversationResponse{}), nil
}

// GetAgentState is unreachable while the chat API does not list its models.
func (s *Server) GetAgentState(
	ctx context.Context,
	_ *connect.Request[numenv1.GetAgentStateRequest],
) (*connect.Response[numenv1.GetAgentStateResponse], error) {
	res := &numenv1.GetAgentStateResponse{}
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
	_ *connect.Request[numenv1.GetAgentProgramsRequest],
) (*connect.Response[numenv1.GetAgentProgramsResponse], error) {
	models, err := s.listModels(ctx)
	configured := s.getModel("")
	program := &numenv1.AgentProgram{
		Name:        ProgramName,
		Title:       programTitle,
		IsInstalled: err == nil,
		LimitsTools: true,
	}
	for _, one := range models {
		program.Models = append(program.Models, &numenv1.AgentModel{
			Name: one.Name, Title: one.Title, IsDefault: one.Name == configured,
		})
	}
	return connect.NewResponse(&numenv1.GetAgentProgramsResponse{Programs: []*numenv1.AgentProgram{program}}), nil
}

// listModels is what the chat API lists: what it listed under the present settings, or what it
// lists now.
func (s *Server) listModels(ctx context.Context) ([]model, error) {
	s.mu.Lock()
	if s.isListed {
		defer s.mu.Unlock()
		return s.models, nil
	}
	asked := chat{baseURL: s.baseURL, key: s.key, client: s.Client}
	s.mu.Unlock()

	listing, stop := context.WithTimeout(ctx, modelsLimit)
	defer stop()
	models, err := asked.listModels(listing)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// A failure is asked again, and what was listed under settings that have changed since is not kept.
	s.mu.Lock()
	defer s.mu.Unlock()
	if err == nil && s.baseURL == asked.baseURL && s.key == asked.key {
		s.models, s.isListed = models, true
	}
	return models, err
}

// Complete answers one question from the model the host names, else the configured one, with no tools, no vault and no conversation.
func (s *Server) Complete(
	ctx context.Context,
	req *connect.Request[numenv1.CompleteRequest],
) (*connect.Response[numenv1.CompleteResponse], error) {
	asked, name := s.getChat(), s.getModel(req.Msg.GetModel())
	return completion.Answer(ctx, req.Msg.GetQuestion(), func(ctx context.Context, request completion.Request) (string, error) {
		var messages []message
		if request.System != "" {
			messages = append(messages, message{Role: "system", Content: newText(request.System)})
		}
		messages = append(messages, message{Role: "user", Content: newText(request.Prompt)})
		reply, _, err := asked.send(ctx, name, messages, nil)
		if err != nil || reply.Content == nil {
			return "", err
		}
		return *reply.Content, nil
	})
}

// Close stops every task still being worked and refuses new ones.
func (s *Server) Close() error {
	return s.registry.Close()
}
