// Package answer carries out acp.task: it asks the model for each message of a conversation through the
// rows of llm.chat of the plugin, calls the tools of the host the model requests, and sends each update of
// the turn.
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
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/ports"
)

// Answerer carries out the tasks of the agent. A conversation continues from the transcript kept for it.
type Answerer struct {
	store   *settings.Store
	chat    ports.Chat
	tools   ports.Tools
	history *history
	running *running
}

var _ acp.Runner = (*Answerer)(nil)

// New returns an answerer that reads its settings from store and keeps conversations by clock.
func New(store *settings.Store, chat ports.Chat, tools ports.Tools, clock ports.Clock) (*Answerer, error) {
	if store == nil || chat == nil || tools == nil || clock == nil {
		return nil, errors.New("an answerer needs settings, a chat, the tools and a clock")
	}
	return &Answerer{
		store: store, chat: chat, tools: tools,
		history: newHistory(clock), running: &running{mu: sync.Mutex{}, cancels: map[*context.CancelFunc]bool{}, isClosed: false},
	}, nil
}

// RunTask carries out one turn of the task and sends its updates. A task whose model cannot be asked,
// because no row of llm.chat is set or every row failed, stops for a reason of unavailable.
func (a *Answerer) RunTask(ctx context.Context, _ connection.Call, task *acp.TaskRequest, send func(*acp.TaskFrame) error) (*acp.TaskDone, error) {
	taskCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if !a.running.add(&cancel) {
		return nil, connection.NewStopError(connection.ReasonUnavailable)
	}
	defer a.running.remove(&cancel)

	cfg := a.store.Get()
	if !cfg.HasRow {
		return nil, connection.NewStopError(connection.ReasonUnavailable)
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

	id := task.GetConversation()
	messages := a.getMessages(id, task, offered)
	taskRun := &run{
		chat:        a.chat,
		temperature: cfg.Temperature,
		maxTokens:   cfg.GetMaxTokens(),
		tools:       a.tools,
		offered:     offered,
		listed:      listed,
		scope:       conversation.Scope{Context: task.GetToolContext(), VaultID: task.GetVaultId()},
		turns:       cfg.GetMaxSteps(),
		send:        send,
	}
	transcript, reason, err := taskRun.answer(taskCtx, messages)
	if ctxErr := taskCtx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if _, isChat := errors.AsType[*chatError](err); isChat {
		return nil, connection.NewStopError(connection.ReasonUnavailable)
	}
	if err != nil {
		return nil, err
	}
	a.history.append(id, transcript[len(messages)-1:])
	return &acp.TaskDone{StopReason: reason}, nil
}

// getMessages is the conversation the model is asked: the instructions of the tools, the transcript kept
// for the conversation and the prompt with the notes attached to it.
func (a *Answerer) getMessages(id string, task *acp.TaskRequest, offered []conversation.Tool) []conversation.Message {
	kept := a.history.get(id)
	messages := make([]conversation.Message, 0, len(kept)+2)
	if instructions := joinInstructions(offered); instructions != "" {
		messages = append(messages, conversation.Message{Role: "system", Text: instructions, ToolCalls: nil, ToolCallID: ""})
	}
	messages = append(messages, kept...)
	return append(messages, conversation.Message{Role: "user", Text: writePrompt(task), ToolCalls: nil, ToolCallID: ""})
}

// joinInstructions joins the distinct instruction sentences of the tools, in the order of the listing.
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
