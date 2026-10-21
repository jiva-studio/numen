package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/review"
)

// Mount registers the services of the plugin on s, working the vault through the host the
// environment names.
func Mount(s *server.Server) error {
	client, err := host.NewClientFromEnv()
	if err != nil {
		return err
	}
	return mount(s, client)
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given. It works the vault through the host
// those name.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		client, err := host.NewClientFromVars(vars)
		if err != nil {
			return err
		}
		return mount(s, client)
	})
}

func mount(s *server.Server, vault host.Vault) error {
	backend := New(Config{Vault: vault})
	s.SetConfigure(func(_ context.Context, values map[string]string) error {
		backend.Configure(values)
		return nil
	})
	s.SetUpdateConfig(backend.Configure)
	return backend.Mount(s)
}

// settingDayStart is the setting that says when a day of review begins.
const settingDayStart = "day_start_time"

// Configure takes the settings the host hands over. A day that begins at an hour that is none stands
// at the default.
func (b *Backend) Configure(values map[string]string) {
	starts, err := review.ParseDayStart(values[settingDayStart])
	if err != nil {
		fmt.Fprintln(os.Stderr, "flashcards-studio:", settingDayStart, err)
	}
	b.Reconfigure(func() time.Duration { return starts })
}

// Mount registers the services and the tools on s.
func (b *Backend) Mount(s *server.Server) error {
	options := s.HandlerOptions()
	s.Register(numenv1connect.NewCardsServiceHandler(b.Cards(), options...))
	s.Register(numenv1connect.NewPresetsServiceHandler(b.Presets(), options...))
	s.Register(numenv1connect.NewFlashcardsServiceHandler(b.Review(), options...))
	registry, err := b.Tools()
	if err != nil {
		return err
	}
	registry.Mount(s)
	return nil
}

// Cards answers CardsService: the decks and stencils a vault is arranged into.
func (b *Backend) Cards() numenv1connect.CardsServiceHandler { return &cardsHandler{Backend: b} }

// Presets answers PresetsService: the presets decks are scheduled by.
func (b *Backend) Presets() numenv1connect.PresetsServiceHandler { return &presetsHandler{Backend: b} }

// Review answers FlashcardsService: what a vault owes, what to ask next and what an answer is written to.
func (b *Backend) Review() numenv1connect.FlashcardsServiceHandler { return &reviewHandler{Backend: b} }
