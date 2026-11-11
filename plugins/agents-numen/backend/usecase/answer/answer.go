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
	models   ports.ModelCatalog
	history  *history
	registry agentrun.Registry
}

// New returns an answerer that reads its settings from store.
func New(store *settings.Store, chat ports.Chat, tools ports.ToolOpener, models ports.ModelCatalog) (*Answerer, error) {
	if store == nil || chat == nil || tools == nil || models == nil {
		return nil, errors.New("an answerer needs settings, a chat API, a tool opener and a model catalog")
	}
	return &Answerer{
		store: store, chat: chat, tools: tools, models: models,
		history: newHistory(), registry: agentrun.Registry{},
	}, nil
}

// Ask works the task and reports each request to the model, its output, each tool call and the
// stop reason to send.
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
	cfg := a.store.Get()
	modelName := cfg.GetModel(task.Model)
	taskRun := &run{
		chat:    a.chat,
		cfg:     cfg,
		model:   modelName,
		effort:  a.getEffort(taskCtx, modelName, task.Reasoning),
		tools:   tools,
		words:   toolcall.IndexDeclarations(task.ToolDeclarations),
		turns:   cfg.GetMaxSteps(int(task.MaxSteps)),
		send:    send,
		isPlain: false,
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
		messages = append(messages, conversation.Message{Role: "system", Content: new(instructions), ToolCalls: nil, ToolCallID: ""})
	}
	messages = append(messages, history...)
	return append(messages, conversation.Message{Role: "user", Content: new(task.Prompt), ToolCalls: nil, ToolCallID: ""})
}

func (a *Answerer) getEffort(ctx context.Context, name, level string) conversation.Effort {
	none := conversation.Effort{Field: "", Level: ""}
	if level == "" {
		return none
	}
	models, err := a.models.ListModels(ctx)
	if err != nil {
		return none
	}
	for _, one := range models {
		if one.Name == name {
			return conversation.NewEffort(one, level)
		}
	}
	return none
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
