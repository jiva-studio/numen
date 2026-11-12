package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/internal/adapter/chatapi"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/internal/adapter/vaulttools"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/usecase/answer"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/usecase/catalog"
)

// agent is numen's own agent: it works a task by calling an OpenAI-compatible chat API and calling
// the vault's tools over MCP on the model's behalf. It answers the calls of the agent service, in a
// plugin's process or linked into the process that hosts the vault. The key sent to the chat API is the first
// secret of the list the host passes for the declaration apiKey; an empty list means no key, and a call
// that does not carry the declaration keeps the key.
type agent struct {
	agentrun.Unimplemented

	settings *settings.Store
	catalog  *catalog.Catalog
	answerer *answer.Answerer
}

func newAgent(opts ...chatapi.Option) (*agent, error) {
	store := settings.NewStore()
	chat := chatapi.New(opts...)
	models, err := catalog.New(store, chat)
	if err != nil {
		return nil, err
	}
	answerer, err := answer.New(store, chat, vaulttools.Opener{}, models)
	if err != nil {
		return nil, err
	}
	return &agent{Unimplemented: agentrun.Unimplemented{}, settings: store, catalog: models, answerer: answerer}, nil
}

func (a *agent) configure(values map[string]string) {
	a.settings.Configure(values)
	a.catalog.Reset()
}

func (a *agent) applySecrets(secrets server.Secrets) {
	list, carried := server.ReadSecrets(secrets, settings.KeyAPIKey)
	if !carried {
		return
	}
	key := ""
	if len(list) > 0 {
		key = list[0].Value
	}
	a.settings.SetKey(key)
	a.catalog.Reset()
}

func (a *agent) Ask(ctx context.Context, task *agentrun.Request, send func(*agentrun.Response) error) error {
	return a.answerer.Ask(ctx, task, send)
}

func (a *agent) FinishConversation(ctx context.Context, conversation string) error {
	return a.answerer.FinishConversation(ctx, conversation)
}

func (a *agent) GetState(ctx context.Context) (*agentrun.State, error) {
	return a.catalog.GetState(ctx), nil
}

func (a *agent) GetPrograms(ctx context.Context, _ []agentrun.ProgramCommand) ([]*agentrun.Program, error) {
	return a.catalog.GetPrograms(ctx), nil
}

// Close stops every running task and rejects new ones.
func (a *agent) Close() error {
	return a.answerer.Close()
}
