package service

import (
	"context"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/hostclient"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/rpc"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/settings"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/wire/cardcommands"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/wire/cardtools"
)

// Mount registers the services of the plugin on s, working the vault and the plugin's folder of its
// store through the host the environment names.
func Mount(s *server.Server) error {
	clients, err := hostclient.NewFromEnv()
	if err != nil {
		return err
	}
	return mount(s, clients)
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given. It works the vault through the host
// those name.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		clients, err := hostclient.NewFromVars(vars)
		if err != nil {
			return err
		}
		return mount(s, clients)
	})
}

func mount(s *server.Server, clients hostclient.Clients) error {
	backend := New(Config{Vaults: clients.Vaults, Store: clients.Folders, DayStart: nil, Now: clock.System{}, Cores: 0, Scheduler: nil})
	s.SetConfigure(func(_ context.Context, values map[string]string, _ server.Secrets) error {
		backend.Configure(values)
		return nil
	})
	s.SetUpdateConfig(func(values map[string]string, _ server.Secrets) { backend.Configure(values) })
	return backend.Mount(s)
}

// ServiceName is the name of the API service the backend serves; the manifest declares it.
const ServiceName = "md.numen.views.flashcards.studio"

// Mount registers the API, the tools and the commands on s.
func (b *Backend) Mount(s *server.Server) error {
	// gomutants:disable-next-line BRANCH_IF reason="Static service and method names are always valid."
	if err := rpc.Mount(s.NewAPI(ServiceName), b.Handlers()); err != nil {
		return err
	}
	registry, err := b.Tools()
	// gomutants:disable-next-line BRANCH_IF reason="Static tool schemas are always valid."
	if err != nil {
		return err
	}
	registry.Mount(s)
	commands, err := b.Commands()
	// gomutants:disable-next-line BRANCH_IF reason="Static command registrations are always valid."
	if err != nil {
		return err
	}
	command.Mount(s, commands)
	return nil
}

// Commands runs the commands the manifest declares.
func (b *Backend) Commands() (*command.Registry, error) {
	return cardcommands.New(func() cards.Create { return b.scenarios().Cards.Create }, b.vaults("").Files, b.createCardNote)
}

// Tools provides the MCP tools for interacting with flashcards and decks.
func (b *Backend) Tools() (*tools.Registry, error) {
	selector := func(id domain.VaultID) host.VaultSelector { return b.vaults(id).Selector }
	return cardtools.New(selector, b.scenarios().Cards, b.scenarios().DeckEdit).Registry()
}

// Handlers returns the handlers of the JSON methods.
func (b *Backend) Handlers() rpc.Handlers {
	cards, presets, reviews := b.newCardsHandler(), b.newPresetsHandler(), b.newReviewHandler()
	return rpc.Handlers{
		Models: cards, Decks: cards, PresetReads: presets, PresetWrites: presets,
		Sessions: reviews, Watches: reviews, Stats: reviews,
	}
}

// Configure applies the settings supplied by the host, and makes the scenarios again with the day
// they set. Calls already running finish under the settings they began under.
func (b *Backend) Configure(values map[string]string) {
	starts := settings.GetDayStart(values)
	b.current.SetDayStart(starts)
}
