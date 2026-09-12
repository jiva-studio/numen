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
	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/modules/libs/core/check"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// This file is the only one that knows an agent can reach the vault. Built with
// `nomcp`, its counterpart takes its place and nothing of the tools — nor of
// what serves them — is linked in at all.

const defaultAgentAddr = mcp.DefaultAddr

// unnamed is what the panel is told where the settings name no agent.
const unnamed = "no agent is named in the settings"

func serveAgents(ctx context.Context, cfg container.Config, opened *editor.Installation, opts agentOptions, out io.Writer) (func() error, error) {
	if opts.off {
		return func() error { return nil }, nil
	}

	// The settings say whether the tools go on a port: an agent named for the
	// panel puts them there, and so does a person asking for the port itself.
	// An installation asking for neither opens no port and mints no token.
	if !cfg.Agent.IsServingTools() {
		opened.API.Unreachable.Store(unnamed)
		return func() error { return nil }, nil
	}

	root, err := filepath.Abs(opened.GetShownVault().Path)
	if err != nil {
		return nil, err
	}

	// An address cleared on the command line is where an agent looks by
	// default: this is the window an agent is configured against.
	addr := opts.addr
	if addr == "" {
		addr = mcp.DefaultAddr
	}

	// The whole surface, and this is the one endpoint an agent a person runs
	// themselves reaches too, announced with the token it presents. Serving a
	// narrow one here would take the vault's writers off that agent, which is
	// not what a person configured it for. What the panel's own child may call
	// is that child's allowance and is narrowed there.
	served, err := agents.Serve(ctx, agents.Options{
		Config:     cfg,
		Core:       agentCore(cfg, opened, root, out),
		Addr:       addr,
		Announcing: true,
		Root:       root,
		Drafting:   makeDrafting(opened),
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

// makeDrafting is how a change the agent is making reaches the window before it
// lands. Where a stretch stands is the vault's to say.
func makeDrafting(opened *editor.Installation) claudecode.Drafting {
	reading := opened.Notes().Read
	return claudecode.Drafting{
		Report: func(ctx context.Context, said domain.Edit) {
			_ = opened.API.GetWindow().ShowEdit(ctx, said)
		},
		Location: func(ctx context.Context, path, stood string) (int, int, bool) {
			contents, err := reading.Execute(ctx, opened.GetShownVault(), path)
			if err != nil || contents.Outcome != note.Ok {
				return 0, 0, false
			}
			at, _ := markdown.Where(contents.Body, stood)
			if len(at) != 1 {
				return 0, 0, false
			}
			return markdown.CountUTF16(contents.Body, at[0].From),
				markdown.CountUTF16(contents.Body, at[0].To), true
		},
	}
}

// agentCore wires the tools to the use cases the window works this vault
// through. They are built once, where the window was put together, so a tool
// and the person reach the vault through the one set: the index is brought
// level by the same refresh the watcher drives, and a note a tool writes is
// findable and drawn wherever it is shown.
//
// The one difference is Drawing, and it is why: a note a tool writes is drawn
// as the stretch that changed, and a note the person writes is not, because
// they are looking at the text they typed.
func agentCore(cfg container.Config, opened *editor.Installation, root string, out io.Writer) mcp.Core {
	notes := opened.Notes().Drawing(opened.API.GetWindow())
	cutting := opened.Cards()

	return mcp.Core{
		Showing:   mcp.ShowOneVault(opened.GetShownVault(), root),
		Readers:   cfg.VaultReaders(),
		View:      opened.API.GetWindow(),
		Attending: opened.API.GetOpenTabs,

		// The list is the window's own, and an agent does not erase a vault:
		// the folder that goes is a person's to ask for.
		Vaults: mcp.Vaults{
			Registry:     opened.API.Vaults.Registry,
			FolderDialog: opened.API.Vaults.FolderDialog,
			Add:          opened.API.Vaults.Add,
			Rename:       opened.API.Vaults.Rename,
			Forget:       opened.API.Vaults.Forget,
			Opens:        opening(opened, out),
		},

		Sources: mcp.Sources{
			Queries:    opened.Index.SourcesKnown(),
			Recognise:  recogniser(opened),
			Transcribe: opened.GetTranscriptionWorker(),
			Derived:    cfg.GetDerivedStores(),
			Documents:  cfg.TextExtractor(),
			URLs:       opened.API.Files.URLs,
			Import:     opened.API.Imports,
			Changed:    opened.API.ReportArtifactChange,
		},

		Cards: mcp.Cards{
			Read:        cutting.Read,
			List:        cutting.List,
			Write:       cutting.Write,
			Create:      cutting.Create,
			RenameField: cutting.Rename,
			DeckEdit:    format.OpenDeckBody,
			StencilBody: format.StencilBody,
		},

		Notes: mcp.Notes{
			Queries: opened.Index.Queries(),
			// The one search the window offers. A model that could not be
			// fitted is said where the person is, and not twice.
			Search:        *opened.API.Finds,
			Neighbourhood: notes.Neighbourhood,
			Links:         notes.Links,
			Problems:      check.Standard(opened.Index.Problems()),

			Create:  notes.Create,
			Write:   notes.Write,
			Replace: notes.Replace,
			Move:    notes.Move,
			Rename:  notes.Rename,
			Remove:  notes.Remove,
			Linking: notes.Linking,
		},
	}
}

// opening is the window moved to another vault, as a tool asks for it.
//
// The session that asked is served for the vault that is going and ends with
// it, so what went wrong is said here. A window that cannot be moved serves no
// tool that would move it.
func opening(opened *editor.Installation, out io.Writer) func(context.Context, domain.Vault) error {
	if opened.API.Opens == nil {
		return nil
	}
	return func(ctx context.Context, v domain.Vault) error {
		err := opened.API.Opens(ctx, v)
		if err != nil {
			fmt.Fprintln(out, "agents:", err)
		}
		return err
	}
}

// recogniser is what reads a scanned document for an agent that asks.
//
// It is always served, even on a machine holding none of the models: what is
// missing is fetched behind whoever asked, and the tool says so. A tool that is
// not served at all leaves an agent saying the vault cannot do a thing it can.
func recogniser(opened *editor.Installation) mcp.Recogniser {
	return opened.GetRecognitionWorker()
}
