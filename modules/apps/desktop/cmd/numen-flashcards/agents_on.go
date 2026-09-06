//go:build !nomcp

package main

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sync"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/agents"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	window "github.com/jiva-studio/numen/modules/libs/core/adapter/window/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
)

// This file is the only one that knows a card can be asked about. Built with
// `nomcp`, its counterpart takes its place and nothing of the tools — nor of
// what serves them — is linked in at all.

// unnamed is what the panel is told where the settings name no agent.
const unnamed = "no agent is named in the settings"

// reaching is the agent on the vault a person's session is on.
//
// A session is on one vault and an agent is told which vault it works when it
// starts, so the endpoint is stopped and served again when a session opens on
// another one.
type reaching struct {
	swapping *agents.Endpoint

	mu    sync.Mutex
	vault domain.Vault
}

// Opened is a session opening on a vault. A session opened on the same vault
// again leaves the agent where it is.
func (r *reaching) Opened(_ context.Context, v domain.Vault) {
	r.mu.Lock()
	again := r.vault.ID == v.ID
	r.vault = v
	r.mu.Unlock()

	if again {
		return
	}
	_ = r.swapping.Around(func() error { return nil })
}

func (r *reaching) showing() domain.Vault {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.vault
}

// serveAgents lets a card be asked about, and answers with what takes that
// away again.
//
// The tools are served on a loopback port this machine picks, with a token that
// lives as long as the window. Nothing is written down: the file an agent a
// person runs themselves is configured from names one vault, and the editor
// writes it.
func serveAgents(
	ctx context.Context,
	cfg container.Config,
	db *container.Index,
	notes container.Notes,
	cutting container.Cards,
	api *window.API,
	off bool,
	out io.Writer,
) func() error {
	if off || cfg.Agent.Use != agent.UseClaude {
		api.Unreachable.Store(unnamed)
		return func() error { return nil }
	}

	secret, err := agents.Mint()
	if err != nil {
		api.Unreachable.Store(err.Error())
		fmt.Fprintln(out, "numen-flashcards: no agent:", err)
		return func() error { return nil }
	}

	// The page asks whether a card can be asked about as it opens, and the agent
	// is started when a person sits down to a vault. What is said here is that
	// this window can reach one.
	api.Unreachable.Store("")

	held := &reaching{}
	held.swapping = &agents.Endpoint{
		Serve: func() (func() error, error) {
			v := held.showing()
			root, err := filepath.Abs(v.Path)
			if err != nil {
				return nil, err
			}
			served, err := agents.Serve(ctx, agents.Options{
				Config:  cfg,
				Core:    reviewing(cfg, db, notes, cutting, api, v, root, out),
				Reviews: true,
				Token:   secret,
				Root:    root,
				Out:     out,
			})
			if err != nil {
				return nil, err
			}
			api.Answers(served.Agent)
			return served.Close, nil
		},
		Showing:     held.showing,
		Handler:     api.Answers,
		Unreachable: func(why string) { api.Unreachable.Store(why) },
		Trouble:     func(err error) { fmt.Fprintln(out, "numen-flashcards: agents:", err) },
	}

	api.Opened = held.Opened
	return func() error {
		held.swapping.Off()
		return nil
	}
}

// reviewing is the tools this window serves: everything that reads, and the
// cards of the deck of a person's session.
//
// A card is written here because that is what a person is doing. A deck and a
// stencil are not made here: they are what a vault is arranged into, and
// arranging one is done in the editor.
//
// Nothing embeds behind this window, so a search answers by the words the vault
// holds. Nothing scans either: a card the agent writes is levelled in the index
// by the paths it touched.
func reviewing(
	cfg container.Config,
	db *container.Index,
	notes container.Notes,
	cutting container.Cards,
	api *window.API,
	v domain.Vault,
	root string,
	out io.Writer,
) mcp.Core {
	return mcp.Core{
		Showing: mcp.ShowingOne(v, root),
		Readers: cfg.VaultReaders(),
		// Which card the person is on is a tool's answer and never part of the
		// question, so a deck named by whoever synced it is data and not
		// instruction.
		Reviewing: func() mcp.AskedCard {
			on := api.Current()
			return mcp.AskedCard{Deck: on.Deck, Card: on.Card, Face: on.Face}
		},

		Notes: mcp.Notes{
			Queries:       db.Queries(),
			Neighbourhood: notes.Neighbourhood,
			Links:         notes.Links,
			Search: cfg.SearchingOver(db.Passages(), nil,
				func(err error) { fmt.Fprintln(out, "agents: answering by words alone:", err) }),
		},

		Sources: mcp.Sources{
			Queries:   db.SourcesKnown(),
			Derived:   cfg.DerivedStores(),
			Documents: cfg.TextExtractor(),
		},

		Cards: mcp.Cards{
			Read:     cutting.Read,
			List:     cutting.List,
			Write:    cutting.Write,
			DeckEdit: format.OpenDeckBody,
		},
	}
}
