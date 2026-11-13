// Package answer carries out acp.task: it asks the model the row of the task names for each message of a
// conversation, calls the tools of the host the model requests, and sends each update of the turn.
package answer

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/ports"
)

// NoModelText is what a task whose row names no model of a connection that serves llm.chat answers.
const NoModelText = "Choose a model."

// Parts are what an answerer works with: its settings, the model, the connections of llm.chat, the tools
// of the host and the clock that ages conversations.
type Parts struct {
	Store  *settings.Store
	Chat   ports.Chat
	Models ports.Models
	Tools  ports.Tools
	Clock  ports.Clock
}

// Answerer carries out the tasks of the agent. A conversation continues from the transcript kept for it.
type Answerer struct {
	store   *settings.Store
	chat    ports.Chat
	models  ports.Models
	tools   ports.Tools
	history *history
	running *running
}

var _ acp.Runner = (*Answerer)(nil)

// New returns an answerer over the parts.
func New(parts Parts) (*Answerer, error) {
	if parts.Store == nil || parts.Chat == nil || parts.Models == nil || parts.Tools == nil || parts.Clock == nil {
		return nil, errors.New("an answerer needs settings, a chat, the connections of llm.chat, the tools and a clock")
	}
	return &Answerer{
		store: parts.Store, chat: parts.Chat, models: parts.Models, tools: parts.Tools,
		history: newHistory(parts.Clock), running: &running{mu: sync.Mutex{}, cancels: map[*context.CancelFunc]bool{}, isClosed: false},
	}, nil
}

// RunTask carries out one turn of the task with the model the row of the task names in llm.OptionModel, a
// model of a connection that serves llm.chat. The model is asked the distinct instruction sentences of the
// tools, in the order of the listing, then the transcript kept for the conversation, then the prompt with
// its attached notes; it is asked again after each tool call until it answers without one, its answer is
// cut off or the steps of the settings are used up. A tool call frame shows the title of the tool, its kind
// by the effect, and the query, about or title of the arguments, else their path, with the path of the
// note. A row that names no model, or a model of a connection that no longer serves llm.chat, sends
// NoModelText and stops for a reason of invalid request. A task whose model cannot be asked stops for the
// reason its connection failed with: no other connection of llm.chat and no other row of acp.task is asked.
// A closed answerer stops for a reason of unavailable.
func (a *Answerer) RunTask(ctx context.Context, call connection.Call, task *acp.TaskRequest, send func(*acp.TaskFrame) error) (*acp.TaskDone, error) {
	taskCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if !a.running.add(&cancel) {
		return nil, connection.NewStopError(connection.ReasonUnavailable)
	}
	defer a.running.remove(&cancel)

	model, err := a.findModel(taskCtx, call.Options[llm.OptionModel])
	if err != nil {
		return nil, a.readModelError(taskCtx, err, send)
	}
	offered, err := a.tools.ListTools(taskCtx)
	if err != nil {
		if ctxErr := taskCtx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("the tools of the host: %w", err)
	}
	listed := make(map[string]conversation.Tool, len(offered))
	for _, one := range offered {
		listed[one.Name] = one
	}

	cfg := a.store.Get()
	id := task.GetConversation()
	messages := a.getMessages(id, task, offered)
	taskRun := &run{
		chat:        a.chat,
		model:       model,
		temperature: cfg.Temperature,
		maxTokens:   cfg.GetMaxTokens(),
		tools:       a.tools,
		offered:     offered,
		listed:      listed,
		scope:       conversation.Scope{Context: task.GetToolContext(), VaultID: task.GetVaultId(), ConnectionID: call.ConnectionID},
		turns:       cfg.GetMaxSteps(),
		send:        send,
	}
	transcript, reason, err := taskRun.answer(taskCtx, messages)
	if ctxErr := taskCtx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if errors.Is(err, errChat) && errors.Is(err, connection.ErrNoConnection) {
		return nil, a.readModelError(taskCtx, errNoModel, send)
	}
	if errors.Is(err, errChat) {
		return nil, connection.NewStopError(readChatReason(err))
	}
	if err != nil {
		return nil, err
	}
	a.history.append(id, transcript[len(messages)-1:])
	return &acp.TaskDone{StopReason: reason}, nil
}

// errNoModel is a row that names no model of a connection that serves llm.chat.
var errNoModel = errors.New("the row names no model")

// errConnections is a listing of the connections of llm.chat that failed.
var errConnections = errors.New("the connections of llm.chat could not be listed")

// readChatReason returns the reason the connection of a failed turn gave, unavailable where it gave none.
func readChatReason(err error) connection.Reason {
	var failed *connection.AllFailedError
	var stopped *connection.StoppedError
	given := ""
	switch {
	case errors.As(err, &failed) && len(failed.Failures) > 0:
		given = failed.Failures[len(failed.Failures)-1].Reason
	case errors.As(err, &stopped):
		given = stopped.Reason
	}
	for _, reason := range []connection.Reason{
		connection.ReasonRejectedKey, connection.ReasonPaymentRequired, connection.ReasonRateLimited, connection.ReasonInvalidRequest,
	} {
		if reason.String() == given {
			return reason
		}
	}
	return connection.ReasonUnavailable
}

func (a *Answerer) findModel(ctx context.Context, value string) (conversation.Model, error) {
	none := conversation.Model{ConnectionID: "", Field: "", ID: ""}
	choice, isChoice := conversation.ParseChoice(value)
	if !isChoice {
		return none, errNoModel
	}
	connections, err := a.models.ListConnections(ctx)
	if err != nil {
		return none, fmt.Errorf("%w: %w", errConnections, err)
	}
	index := slices.IndexFunc(connections, func(one conversation.ChatConnection) bool {
		return one.ID == choice.ConnectionID && one.ModelField != ""
	})
	if index < 0 {
		return none, errNoModel
	}
	return conversation.Model{ConnectionID: choice.ConnectionID, Field: connections[index].ModelField, ID: choice.ModelID}, nil
}

func (a *Answerer) readModelError(ctx context.Context, err error, send func(*acp.TaskFrame) error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if !errors.Is(err, errNoModel) {
		return connection.NewTryNextError(connection.ReasonUnavailable, 0)
	}
	if sendErr := send(&acp.TaskFrame{Update: &acp.TaskFrame_AgentMessageChunk{
		AgentMessageChunk: &acp.MessageChunk{MessageId: "message-1", Text: NoModelText},
	}}); sendErr != nil {
		return sendErr
	}
	return connection.NewStopError(connection.ReasonInvalidRequest)
}

func (a *Answerer) getMessages(id string, task *acp.TaskRequest, offered []conversation.Tool) []conversation.Message {
	kept := a.history.get(id)
	messages := make([]conversation.Message, 0, len(kept)+2)
	if instructions := joinInstructions(offered); instructions != "" {
		messages = append(messages, conversation.Message{Role: "system", Text: instructions, ToolCalls: nil, ToolCallID: ""})
	}
	messages = append(messages, kept...)
	return append(messages, conversation.Message{Role: "user", Text: writePrompt(task), ToolCalls: nil, ToolCallID: ""})
}

func joinInstructions(offered []conversation.Tool) string {
	var sentences []string
	for _, one := range offered {
		for _, sentence := range one.Instructions {
			if !slices.Contains(sentences, sentence) {
				sentences = append(sentences, sentence)
			}
		}
	}
	return strings.Join(sentences, " ")
}

func writePrompt(task *acp.TaskRequest) string {
	notes := task.GetContextNotes()
	if len(notes) == 0 {
		return task.GetPrompt()
	}
	var text strings.Builder
	text.WriteString(task.GetPrompt())
	for _, note := range notes {
		fmt.Fprintf(&text, "\n\nAttached note %q (%s):\n%s", note.GetTitle(), note.GetPath(), note.GetText())
	}
	return text.String()
}

// Close stops every running task and rejects new ones.
func (a *Answerer) Close() error {
	a.running.close()
	return nil
}

// running holds the cancel of every task that has not ended.
type running struct {
	mu       sync.Mutex
	cancels  map[*context.CancelFunc]bool
	isClosed bool
}

func (r *running) add(cancel *context.CancelFunc) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.isClosed {
		return false
	}
	r.cancels[cancel] = true
	return true
}

func (r *running) remove(cancel *context.CancelFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.cancels, cancel)
}

func (r *running) close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.isClosed = true
	for cancel := range r.cancels {
		(*cancel)()
	}
	r.cancels = map[*context.CancelFunc]bool{}
}
