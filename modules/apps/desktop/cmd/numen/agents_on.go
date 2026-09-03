//go:build !nomcp

package main

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/claudecode"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/agents"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/webui"
	format "github.com/jiva-studio/numen/modules/libs/core/cards"
	"github.com/jiva-studio/numen/modules/libs/core/check"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// This file is the only one that knows an agent can reach the vault. Built with
// `nomcp`, its counterpart takes its place and nothing of the tools — nor of
// what serves them — is linked in at all.

const defaultAgentAddr = mcp.DefaultAddr

// unnamed is what the panel is told where the settings name no agent.
const unnamed = "no agent is named in the settings"

func serveAgents(ctx context.Context, cfg container.Config, opened *webui.Opened, opts agentOptions, out io.Writer) (func() error, error) {
	if opts.off {
		return func() error { return nil }, nil
	}

	// The settings say whether the tools go on a port: an agent named for the
	// panel puts them there, and so does a person asking for the port itself.
	// An installation asking for neither opens no port and mints no token.
	if !cfg.Agent.Serving() {
		opened.API.Unreachable.Store(unnamed)
		return func() error { return nil }, nil
	}

	root, err := filepath.Abs(opened.Showing().Path)
	if err != nil {
		return nil, err
	}

	// An address cleared on the command line is where an agent looks by
	// default: this is the window an agent is configured against.
	addr := opts.addr
	if addr == "" {
		addr = mcp.DefaultAddr
	}

	served, err := agents.Serve(ctx, agents.Options{
		Config:     cfg,
		Core:       agentCore(cfg, opened, root, out),
		Addr:       addr,
		Announcing: true,
		Root:       root,
		Drafting:   drafting(cfg, opened),
		Out:        out,
	})
	if err != nil {
		return nil, err
	}

	// The panel answers with the agent the settings name. An installation
	// naming none is served the tools alone, and the panel says so.
	if served.Agent == nil {
		opened.API.Unreachable.Store(unnamed)
	} else {
		opened.API.Answers(served.Agent)
	}
	return served.Close, nil
}

// drafting is how a change the agent is making reaches the window before it
// lands. Where a stretch stands is the vault's to say.
func drafting(cfg container.Config, opened *webui.Opened) claudecode.Drafting {
	reading := note.Read{Readers: cfg.VaultReaders()}
	return claudecode.Drafting{
		Tell: func(ctx context.Context, said domain.Editing) {
			_ = opened.API.Viewing().Editing(ctx, said)
		},
		Where: func(ctx context.Context, path, stood string) (int, int, bool) {
			contents, err := reading.Execute(ctx, opened.Showing(), path)
			if err != nil || contents.Outcome != note.Ok {
				return 0, 0, false
			}
			at, _ := markdown.Where(contents.Body, stood)
			if len(at) != 1 {
				return 0, 0, false
			}
			return markdown.Counted(contents.Body, at[0].From),
				markdown.Counted(contents.Body, at[0].To), true
		},
	}
}

// agentCore wires the tools to the same use cases everything else uses. The
// index is brought level by the same refresh the watcher drives, so a tool that
// writes a note leaves it findable.
func agentCore(cfg container.Config, opened *webui.Opened, root string, out io.Writer) mcp.Core {
	index := func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := opened.Refresh().Execute(ctx, v, paths)
		return err
	}
	readers := cfg.VaultReaders()
	writers := cfg.VaultWriters()
	queries := opened.Index.Queries()
	viewing := opened.API.Viewing()
	// What a write is doing reaches the window the way a note put in front of
	// the person does. A build with no window draws nothing and is told nothing.
	tells := note.Telling(func(ctx context.Context, said domain.Editing) {
		if viewing == nil {
			return
		}
		_ = viewing.Editing(ctx, said)
	})
	went := note.Moving(func(ctx context.Context, gone domain.Went) {
		if viewing == nil {
			return
		}
		_ = viewing.Moved(ctx, gone)
	})
	moves := note.Move{
		Readers: readers, Writers: writers, Links: opened.Index.Links(),
		Names:   queries,
		Sources: opened.Index.Sources(), Index: index,
		Moving: went,
		Sync:   cfg.Syncing(),
	}

	cutting := cfg.Cards(queries, opened.Index.Links(), index)

	return mcp.Core{
		Showing:   mcp.One(opened.Showing(), root),
		Readers:   readers,
		View:      opened.API.Viewing(),
		Attending: opened.API.Attended,
		Notes:     queries,

		Vaults:     opened.API.Vaults,
		Renaming:   opened.API.Renaming,
		Forgetting: opened.API.Forgetting,

		Sources:    opened.Index.SourcesKnown(),
		Recognise:  recogniser(opened),
		Transcribe: opened.Transcribing(),
		Derived:    cfg.DerivedStores(),
		Documents:  cfg.Documents(),

		Search: cfg.Searching(opened.Index, opened.Asking,
			func(err error) { fmt.Fprintln(out, "agents: answering by words alone:", err) }),
		Neighbourhood: note.ShowNeighbourhood{Links: opened.Index.Links(), Notes: queries},
		Links:         note.ShowLinks{Links: opened.Index.Links()},
		Problems:      check.Standard(opened.Index.Problems()),

		Cards:       cutting.Read,
		Stencils:    cutting.List,
		Cuts:        cutting.Write,
		Cutting:     cutting.Create,
		FieldRename: cutting.Rename,
		DeckBody:    format.DeckBody,
		StencilBody: container.StencilBody,

		Create:  note.Create{Writers: writers, Names: queries, Index: index},
		Write:   note.Write{Readers: readers, Writers: writers, Index: index, Telling: tells},
		Replace: note.Replace{Readers: readers, Writers: writers, Index: index, Telling: tells},
		Move:    moves,
		Rename:  note.Rename{Move: moves},
		Remove:  note.Remove{Writers: writers, Links: opened.Index.Links(), Known: opened.Index.SourcesKnown(), Index: index},
		Linking: note.Linking{Readers: readers, Writers: writers, Index: index},
	}
}

// recogniser is what reads a scanned document for an agent that asks.
//
// It is always served, even on a machine holding none of the models: what is
// missing is fetched behind whoever asked, and the tool says so. A tool that is
// not served at all leaves an agent saying the vault cannot do a thing it can.
func recogniser(opened *webui.Opened) mcp.Recognising {
	return opened.Recognising()
}
