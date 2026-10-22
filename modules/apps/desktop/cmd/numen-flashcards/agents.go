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
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// This file is the only one that knows a card can be asked about.

// errNoAgentNamed is what the panel is told where the settings name no agent.
const errNoAgentNamed = "no agent is named in the settings"

// vaultTracker is the endpoint on the vault a person's session is on.
//
// A session is on one vault and an endpoint is served for the vault it works when it
// starts, so the endpoint is stopped and served again when a session opens on
// another one.
type vaultTracker struct {
	endpoint *agents.Endpoint

	mu    sync.Mutex
	vault domain.Vault
	url   string
}

func (r *vaultTracker) getURL() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.url
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
	api *window.API,
	off bool,
	out io.Writer,
) func() error {
	_, away := serveAgentsTracker(ctx, cfg, db, notes, api, off, out)
	return away
}

func serveAgentsTracker(
	ctx context.Context,
	cfg container.Config,
	db *container.Index,
	notes container.Notes,
	api *window.API,
	off bool,
	out io.Writer,
) (*vaultTracker, func() error) {
	if off || !cfg.Agent.IsNamed() {
		api.Unreachable.Store(errNoAgentNamed)
		return &vaultTracker{}, func() error { return nil }
	}

	secret, err := agents.CreateToken()
	if err != nil {
		api.Unreachable.Store(err.Error())
		fmt.Fprintln(out, "numen-flashcards: no agent:", err)
		return &vaultTracker{}, func() error { return nil }
	}

	// The page asks whether a card can be asked about as it opens, and the agent
	// this window can reach one.
	api.Unreachable.Store("")

	target := &vaultTracker{}
	target.endpoint = &agents.Endpoint{
		Serve: func() (func() error, error) {
			v := target.getVault()
			root, err := filepath.Abs(v.Path)
			if err != nil {
				return nil, err
			}
			served, err := agents.Serve(ctx, agents.Options{
				Config:       cfg,
				Core:         makeReviewCore(cfg, db, notes, api, v, root, out),
				ShouldReview: true,
				Token:        secret,
				Root:         root,
				Out:          out,
			})
			if err != nil {
				return nil, err
			}
			target.mu.Lock()
			target.url = served.URL
			target.mu.Unlock()
			return func() error {
				target.mu.Lock()
				target.url = ""
				target.mu.Unlock()
				return served.Close()
			}, nil
		},
		GetVault:       target.getVault,
		SetUnreachable: func(why string) { api.Unreachable.Store(why) },
		ErrorHandler:   func(err error) { fmt.Fprintln(out, "numen-flashcards: agents:", err) },
	}

	api.Opened = target.SetSessionVault
	return target, func() error {
		target.endpoint.Stop()
		return nil
	}
}

// reviewing is the tools this window serves: everything that reads, and what the plugins offer here.
//
// Nothing embeds behind this window, so a search answers by the words the vault
// holds. Nothing scans either: a card the agent writes is levelled in the index
// by the paths it touched.
func makeReviewCore(
	cfg container.Config,
	db *container.Index,
	notes container.Notes,
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
		Plugins:     cfg.GetPluginTools(),
		// What the person is looking at is data the plugins are told with each call and never part of the
		// question, so a deck named by whoever synced it is data and not instruction.
		GetAsked: api.GetAsked,

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
	}
}
