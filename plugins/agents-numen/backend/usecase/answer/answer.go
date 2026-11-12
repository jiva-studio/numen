// Package answer works a task: it asks the model for each message of a conversation, calls the
// vault's tools the model requests, and reports each step and why the work stopped.
package answer

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/toolcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/rpcerror"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/ports"
)

var errClosing = errors.New("this agent is closing")

// Answerer works the tasks of the agent. A conversation continues from the transcript stored for it.
type Answerer struct {
	store    *settings.Store
	chat     ports.Chat
	tools    ports.ToolOpener
	history  *history
	registry agentrun.Registry
}

// New returns an answerer that reads its settings from store.
func New(store *settings.Store, chat ports.Chat, tools ports.ToolOpener) (*Answerer, error) {
	if store == nil || chat == nil || tools == nil {
		return nil, errors.New("an answerer needs settings, a chat and a tool opener")
	}
	return &Answerer{
		store: store, chat: chat, tools: tools,
		history: newHistory(), registry: agentrun.Registry{},
	}, nil
}

// Ask works the task and reports each request to the model, its output, each tool call and the
// stop reason to send. A task stops at once while the settings name no connection.
func (a *Answerer) Ask(ctx context.Context, task *agentrun.Request, send func(*agentrun.Response) error) (err error) {
	taskCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	id := task.Conversation
	work := agentrun.NewWork(id, 0, agentrun.StopByCancel(cancel))
	if !a.registry.Add(work) {
		return rpcerror.New(rpcerror.Unavailable, errClosing)
	}
	defer a.registry.Remove(work)
	defer work.End()

	cfg := a.store.Get()
	if !cfg.HasRow {
		return send(&agentrun.Response{Step: &agentrun.StoppedStep{Stopped: conversation.ReasonNoConnection}})
	}
	toolServer := conversation.ToolServer{URL: task.ToolsURL, Token: task.ToolsToken, Names: task.Tools, Meta: task.Context}
	tools, err := a.tools.OpenTools(taskCtx, toolServer)
	if err != nil {
		if taskCtx.Err() != nil {
			return nil
		}
		return send(&agentrun.Response{Step: &agentrun.StoppedStep{
			Stopped: fmt.Sprintf("the agent could not reach this vault: %v", err),
		}})
	}
	defer func() { err = errors.Join(err, tools.Close()) }()

	messages := a.getMessages(id, task)
	taskRun := &run{
		chat:        a.chat,
		temperature: cfg.Temperature,
		maxTokens:   cfg.GetMaxTokens(),
		tools:       tools,
		words:       toolcall.IndexDeclarations(task.ToolDeclarations),
		turns:       cfg.GetMaxSteps(int(task.MaxSteps)),
		send:        send,
	}
	transcript, reason, err := taskRun.answer(taskCtx, messages)
	if taskCtx.Err() != nil {
		return nil
	}
	if failed := (*chatError)(nil); errors.As(err, &failed) {
		reason = failed.Error()
	} else if err != nil {
		return err
	}
	if reason == ReasonDone || reason == ReasonTurns || reason == ReasonCut {
		a.history.append(id, transcript[len(messages)-1:])
	}
	return send(&agentrun.Response{Step: &agentrun.StoppedStep{Stopped: reason}})
}

func (a *Answerer) getMessages(id string, task *agentrun.Request) []conversation.Message {
	history := a.history.get(id)
	messages := make([]conversation.Message, 0, len(history)+2)
	if instructions := task.Instructions; instructions != "" {
		messages = append(messages, conversation.Message{Role: "system", Text: instructions, ToolCalls: nil, ToolCallID: ""})
	}
	messages = append(messages, history...)
	return append(messages, conversation.Message{Role: "user", Text: task.Prompt, ToolCalls: nil, ToolCallID: ""})
}

// FinishConversation stops what is still running in the conversation and discards its history.
func (a *Answerer) FinishConversation(ctx context.Context, id string) error {
	if err := a.registry.EndConversation(ctx, id); err != nil {
		return rpcerror.New(rpcerror.Internal, err)
	}
	a.history.discard(id)
	return nil
}

// Close stops every running task and rejects new ones.
func (a *Answerer) Close() error {
	return a.registry.Close()
}
