package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/toolcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/rpcerror"
)

// ProgramName is the identifier a task uses to select the agent, and the key of its settings section.
const ProgramName = "numen"

// programTitle is the agent's display name.
const programTitle = "Numen Agent"

// The settings Configure reads.
const (
	SettingConnections = "connections"
	SettingTemperature = "temperature"
	SettingMaxTokens   = "maxTokens"
	SettingMaxSteps    = "maxSteps"
)

// reasonNoConnection is why a run stops, and why the agent is unreachable, while no row names a connection.
const reasonNoConnection = "the Numen Agent has no connection: choose one in its settings"

var errClosing = errors.New("this agent is closing")

// chatStream is the events of one turn, as the stream of connection.Client gives them.
type chatStream interface {
	Next() (connection.ChatEvent, bool)
	Err() error
	Close() error
}

// askModel asks the core for the next turn of a conversation.
type askModel func(ctx context.Context, request connection.ChatRequest) (chatStream, error)

// Server is numen's own agent: it works a task by asking the core for the turns of a conversation with a
// model, through the connection its settings name, and calling the vault's tools over MCP on the model's
// behalf. The core holds the address and the key of the connection. It implements AgentService, in a
// plugin's process or linked into the process that hosts the vault.
type Server struct {
	agentrun.Unimplemented

	chat askModel

	mu            sync.Mutex
	row           *connection.Row
	temperature   *float64
	maxTokens     *int
	maxSteps      int
	conversations map[string][]connection.ChatMessage

	registry agentrun.Registry
}

// Configure applies the settings: the row that names the connection, the model and its reasoning
// level, and the options of a request. A setting absent from the values keeps its current value, and a
// row setting that is no list of rows is left out.
func (s *Server) Configure(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rows, err := connection.ReadRows(values, SettingConnections); err == nil && values[SettingConnections] != "" {
		s.row = nil
		if len(rows) > 0 {
			s.row = &rows[0]
		}
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
}

func readPositive(text string) (int, bool) {
	val, err := strconv.Atoi(text)
	return val, err == nil && val > 0
}

func (s *Server) getRow() (connection.Row, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.row == nil {
		return connection.Row{}, false
	}
	return *s.row, true
}

func (s *Server) getOptions() (*float64, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	maxTokens := 0
	if s.maxTokens != nil {
		maxTokens = *s.maxTokens
	}
	return s.temperature, maxTokens
}

func (s *Server) getMaxSteps(task *agentrun.Request) int {
	if task.MaxSteps > 0 {
		return int(task.MaxSteps)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.maxSteps
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

	if _, hasRow := s.getRow(); !hasRow {
		return send(&agentrun.Response{Step: &agentrun.StoppedStep{Stopped: reasonNoConnection}})
	}
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
	messages := make([]connection.ChatMessage, 0, len(history)+2)
	if instructions := task.Instructions; instructions != "" {
		messages = append(messages, connection.ChatMessage{Role: "system", Text: instructions})
	}
	messages = append(messages, history...)
	messages = append(messages, connection.ChatMessage{Role: "user", Text: task.Prompt})

	temperature, maxTokens := s.getOptions()
	taskRun := &run{
		ask:         s.chat,
		temperature: temperature,
		maxTokens:   maxTokens,
		tools:       tools,
		words:       toolcall.IndexDeclarations(task.ToolDeclarations),
		turns:       s.getMaxSteps(task),
		send:        send,
	}
	transcript, reason, err := taskRun.answer(taskCtx, messages)
	if taskCtx.Err() != nil {
		return nil
	}
	if failed := (*failure)(nil); errors.As(err, &failed) {
		reason = failed.Error()
	} else if err != nil {
		return err
	}
	if reason == reasonDone || reason == reasonTurns || reason == reasonCut {
		s.appendConversation(conversation, transcript[len(messages)-1:])
	}
	return send(&agentrun.Response{Step: &agentrun.StoppedStep{Stopped: reason}})
}

func (s *Server) getConversation(conversation string) []connection.ChatMessage {
	if conversation == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]connection.ChatMessage(nil), s.conversations[conversation]...)
}

func (s *Server) appendConversation(conversation string, transcript []connection.ChatMessage) {
	if conversation == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conversations == nil {
		s.conversations = map[string][]connection.ChatMessage{}
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

// GetState is unreachable while no row names a connection.
func (s *Server) GetState(context.Context) (*agentrun.State, error) {
	state := &agentrun.State{Unreachable: ""}
	if _, hasRow := s.getRow(); !hasRow {
		state.Unreachable = reasonNoConnection
	}
	return state, nil
}

// GetPrograms lists the one program this agent runs. It is installed while a row names a connection,
// and the model of the row is the one model it offers. It runs only the vault tools a task names.
func (s *Server) GetPrograms(context.Context, []agentrun.ProgramCommand) ([]*agentrun.Program, error) {
	row, hasRow := s.getRow()
	program := &agentrun.Program{
		Name:        ProgramName,
		Title:       programTitle,
		IsInstalled: hasRow,
		LimitsTools: true,
		IsDefault:   false,
		Models:      nil,
	}
	if hasRow {
		program.Models = []*reasoning.Model{{
			Name: row.Model, Title: row.Model, IsDefault: true, ReasoningLevels: nil, DefaultReasoning: "",
		}}
	}
	return []*agentrun.Program{program}, nil
}

// Close stops every running task and rejects new ones.
func (s *Server) Close() error {
	return s.registry.Close()
}

// failure is a run stopped by a turn that failed. It stops the run and is reported as why it stopped.
type failure struct{ err error }

func (f *failure) Error() string { return describeFailure(f.err) }

func (f *failure) Unwrap() error { return f.err }

func describeFailure(err error) string {
	var allFailed *connection.AllFailedError
	var rejected *connection.RejectedError
	switch {
	case errors.Is(err, connection.ErrNoConnection):
		return reasonNoConnection
	case errors.As(err, &allFailed):
		parts := make([]string, len(allFailed.Failures))
		for i, one := range allFailed.Failures {
			parts[i] = one.ConnectionName + ": " + one.Reason
		}
		return strings.Join(parts, "; ")
	case errors.As(err, &rejected):
		return rejected.ConnectionName + ": " + rejected.Reason
	}
	return err.Error()
}

func newAgent(chat askModel) *Server {
	return &Server{
		Unimplemented: agentrun.Unimplemented{},
		chat:          chat,
		mu:            sync.Mutex{},
		row:           nil,
		temperature:   nil,
		maxTokens:     nil,
		maxSteps:      0,
		conversations: nil,
		registry:      agentrun.Registry{},
	}
}
