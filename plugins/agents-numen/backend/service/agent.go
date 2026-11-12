package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/internal/adapter/vaulttools"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/ports"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/usecase/answer"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/usecase/catalog"
)

// agent is numen's own agent: it works a task by asking the host for the turns of a conversation with a
// model, through the connection its settings name, and calling the vault's tools over MCP on the
// model's behalf. The host holds the address and the key of the connection. It answers the calls of the
// agent service, in a plugin's process or linked into the process that hosts the vault.
type agent struct {
	agentrun.Unimplemented

	settings *settings.Store
	catalog  *catalog.Catalog
	answerer *answer.Answerer
}

func newAgent(chat ports.Chat) (*agent, error) {
	store := settings.NewStore()
	models, err := catalog.New(store)
	if err != nil {
		return nil, err
	}
	answerer, err := answer.New(store, chat, vaulttools.Opener{})
	if err != nil {
		return nil, err
	}
	return &agent{Unimplemented: agentrun.Unimplemented{}, settings: store, catalog: models, answerer: answerer}, nil
}

func (a *agent) Ask(ctx context.Context, task *agentrun.Request, send func(*agentrun.Response) error) error {
	return a.answerer.Ask(ctx, task, send)
}

func (a *agent) FinishConversation(ctx context.Context, conversation string) error {
	return a.answerer.FinishConversation(ctx, conversation)
}

func (a *agent) GetState(context.Context) (*agentrun.State, error) {
	return a.catalog.GetState(), nil
}

func (a *agent) GetPrograms(context.Context, []agentrun.ProgramCommand) ([]*agentrun.Program, error) {
	return a.catalog.GetPrograms(), nil
}

// Close stops every running task and rejects new ones.
func (a *agent) Close() error {
	return a.answerer.Close()
}
