//go:build !nomcp

package main

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sync"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/agents"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	window "github.com/jiva-studio/numen/modules/libs/core/adapter/window/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// This file is the only one that knows a card can be asked about. Built with
// `nomcp`, its counterpart takes its place and nothing of the tools — nor of
// what serves them — is linked in at all.

// errNoAgentNamed is what the panel is told where the settings name no agent.
const errNoAgentNamed = "no agent is named in the settings"

// vaultTracker is the agent on the vault a person's session is on.
//
// A session is on one vault and an agent is told which vault it works when it
// starts, so the endpoint is stopped and served again when a session opens on
// another one.
type vaultTracker struct {
	endpoint *agents.Endpoint

	mu    sync.Mutex
	vault domain.Vault
}

// SetSessionVault is a session opening on a vault. A session opened on the same
// vault again leaves the agent where it is.
func (r *vaultTracker) SetSessionVault(_ context.Context, v domain.Vault) {
	r.mu.Lock()
	again := r.vault.ID == v.ID
	r.vault = v
	r.mu.Unlock()

	if again {
		return
	}
	_ = r.endpoint.RunWhileStopped(func() error { return nil })
}

func (r *vaultTracker) getVault() domain.Vault {
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
	cards container.Cards,
	api *window.API,
	off bool,
	out io.Writer,
) func() error {
	if off || !cfg.Agent.IsNamed() {
		api.Unreachable.Store(errNoAgentNamed)
		return func() error { return nil }
	}

	secret, err := agents.CreateToken()
	if err != nil {
		api.Unreachable.Store(err.Error())
		fmt.Fprintln(out, "numen-flashcards: no agent:", err)
		return func() error { return nil }
	}

	// The page asks whether a card can be asked about as it opens, and the agent
	// is started when a person sits down to a vault. What is said here is that
	// this window can reach one.
	api.Unreachable.Store("")

	held := &vaultTracker{}
	held.endpoint = &agents.Endpoint{
		Serve: func() (func() error, error) {
			v := held.getVault()
			root, err := filepath.Abs(v.Path)
			if err != nil {
				return nil, err
			}
			served, err := agents.Serve(ctx, agents.Options{
				Config:           cfg,
				Core:             makeReviewCore(cfg, db, notes, cards, api, v, root, out),
				ShouldReview:     true,
				LimitedToolsOnly: true,
				Token:            secret,
				Root:             root,
				Out:              out,
			})
			if err != nil {
				return nil, err
			}
			api.SetAgent(served.Agent)
			return served.Close, nil
		},
		GetVault:       held.getVault,
		SetAgent:       api.SetAgent,
		SetUnreachable: func(why string) { api.Unreachable.Store(why) },
		ErrorHandler:   func(err error) { fmt.Fprintln(out, "numen-flashcards: agents:", err) },
	}

	api.Opened = held.SetSessionVault
	return func() error {
		held.endpoint.Stop()
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
func makeReviewCore(
	cfg container.Config,
	db *container.Index,
	notes container.Notes,
	cards container.Cards,
	api *window.API,
	v domain.Vault,
	root string,
	out io.Writer,
) mcp.Core {
	return mcp.Core{
		Showing:     mcp.ShowOneVault(v, root),
		GetProblems: cfg.GetProblems(db.Problems()),
		GetSummary:  cfg.GetSummary(db.Queries()),
		Read:        cfg.Read(),
		// Which card the person is on is a tool's answer and never part of the
		// question, so a deck named by whoever synced it is data and not
		// instruction.
		GetCurrentCard: func() mcp.CurrentCard {
			on := api.GetCurrentCard()
			return mcp.CurrentCard{Deck: on.Deck, Card: on.Card, Face: on.Face}
		},

		Notes: mcp.Notes{
			GetNamedPaths: cfg.GetNamedPaths(db.Queries()),
			Neighbourhood: notes.Neighbourhood,
			Links:         notes.Links,
			Read:          notes.Read,
			GetTitles:     notes.GetTitles,
			Search: cfg.NewSearchOver(db.Passages(), nil,
				func(err error) { fmt.Fprintln(out, "agents: answering by words alone:", err) }),
		},

		Sources: mcp.Sources{
			GetDocumentList: cfg.GetDocumentList(db.SourcesKnown()),
			GetDocumentText: cfg.GetDocumentText(db.SourcesKnown()),
		},

		Cards: mcp.Cards{
			Read:            cards.Read,
			Write:           cards.Write,
			GetCardPage:     cards.GetCardPage,
			AddCard:         cards.AddCard,
			EditCard:        cards.EditCard,
			RemoveCardValue: cards.RemoveCardValue,
			RemoveCard:      cards.RemoveCard,
			AddSection:      cards.AddSection,
			RenameSection:   cards.RenameSection,
			RemoveSection:   cards.RemoveSection,
			List:            cards.List,
		},
	}
}
