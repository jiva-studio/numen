//go:build !nomcp

package app

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/agents"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// This file is the only one that knows an agent can reach the vault. Built with
// `nomcp`, its counterpart takes its place and nothing of the tools — nor of
// what serves them — is linked in at all.

const DefaultAgentAddr = mcp.DefaultAddr

// unnamed is what the panel is told where the settings name no agent.
const unnamed = "no agent is named in the settings"

func serveAgents(ctx context.Context, cfg container.Config, opened *editor.Installation, opts agentOptions, out io.Writer) (func() error, error) {
	if opts.isOff {
		return func() error { return nil }, nil
	}

	// The settings say whether the tools go on a port: an agent named for the
	// panel puts them there, and so does a person asking for the port itself.
	// An installation asking for neither opens no port and mints no token.
	if !cfg.Agent.IsServingTools() {
		opened.State.Unreachable.Store(unnamed)
		return func() error { return nil }, nil
	}

	root, err := filepath.Abs(opened.GetCurrentVault().Path)
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
		Config:       cfg,
		Core:         makeAgentCore(cfg, opened, root, out),
		Addr:         addr,
		IsAnnouncing: true,
		Root:         root,
		Preview:      makeEditPreview(opened),
		Out:          out,
	})
	if err != nil && opts.addr == "" {
		served, err = agents.Serve(ctx, agents.Options{
			Config:       cfg,
			Core:         makeAgentCore(cfg, opened, root, out),
			Addr:         "127.0.0.1:0",
			IsAnnouncing: false,
			Root:         root,
			Preview:      makeEditPreview(opened),
			Out:          out,
		})
	}
	if err != nil {
		return nil, err
	}

	// The panel answers with the agent the settings name. An installation
	// naming none is served the tools alone, and the panel says so.
	if served.Agent == nil {
		opened.State.Unreachable.Store(unnamed)
	} else {
		opened.State.SetAgent(served.Agent)
	}
	return served.Close, nil
}

// makeEditPreview is how a change the agent is making reaches the window before it
// lands. Where a stretch stands is the vault's to say.
func makeEditPreview(opened *editor.Installation) agents.EditPreview {
	return agents.EditPreview{
		Emit: func(ctx context.Context, said domain.Edit) {
			_ = opened.State.GetWindow().ShowEdit(ctx, said)
		},
		Locate: opened.DraftingLocation,
	}
}

// makeAgentCore wires the tools to the use cases the window works this vault
// through. They are built once, where the window was put together, so a tool
// and the person reach the vault through the one set: the index is brought
// level by the same refresh the watcher drives, and a note a tool writes is
// findable and drawn wherever it is shown.
//
// The one difference is Drawing, and it is why: a note a tool writes is drawn
// as the stretch that changed, and a note the person writes is not, because
// they are looking at the text they typed.
func makeAgentCore(cfg container.Config, opened *editor.Installation, root string, out io.Writer) mcp.Core {
	notes := opened.Notes().ShowEdits(opened.State.GetWindow())
	cards := opened.Cards()

	return mcp.Core{
		Showing:     mcp.ShowOneVault(opened.GetCurrentVault(), root),
		GetProblems: cfg.GetProblems(opened.Problems()),
		GetSummary:  cfg.GetSummary(opened.Queries()),
		Read:        cfg.Read(),
		Attending:   opened.State.GetOpenTabs,

		// The list is the window's own, and an agent does not erase a vault:
		// the folder that goes is a person's to ask for.
		Vaults: mcp.Vaults{
			GetKnownVaults: cfg.GetKnownVaults(opened.Vaults().Registry),
			Find:           &opened.Vaults().Find,
			PickFolder:     opened.Services.GetPickFolder(),
			Add:            &opened.Vaults().Add,
			CheckRoot:      cfg.CheckRoot(),
			Rename:         &opened.Vaults().Rename,
			Unregister:     &opened.Vaults().Unregister,
			Opens:          makeOpenVault(opened, out),
		},

		Sources: mcp.Sources{
			GetDocumentList:           cfg.GetDocumentList(opened.SourcesKnown()),
			GetDocumentText:           cfg.GetDocumentText(opened.SourcesKnown()),
			StartRecognition:          cfg.StartRecognition(opened.GetRecognitionWorker()),
			GetRecognitionStatus:      cfg.GetRecognitionStatus(opened.GetRecognitionWorker()),
			StartTranscription:        cfg.StartTranscription(opened.GetTranscriptionWorker()),
			GetTranscriptionReadiness: cfg.GetTranscriptionReadiness(opened.GetTranscriptionWorker()),
			CreateAndImportURL:        cfg.CreateAndImportURL(opened.GetCreateURL(), opened.GetImportURL()),
			GetArtifactList:           cfg.GetArtifactList(opened.SourcesKnown()),
			GetArtifactText:           cfg.GetArtifactText(opened.SourcesKnown()),
			WriteCorrections:          cfg.WriteCorrections(opened.SourcesKnown(), opened.Services.Artifact.ReportArtifactChange),
			FocusPassage:              cfg.FocusPassage(opened.State.GetWindow()),
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
			Create:          cards.Create,
			RenameField:     cards.BoundedRenameField,
		},

		Notes: mcp.Notes{
			GetNamedPaths: cfg.GetNamedPaths(opened.Queries()),
			// The one search the window offers. A model that could not be
			// fitted is said where the person is, and not twice.
			Search:        *opened.GetSearch(),
			Neighbourhood: notes.Neighbourhood,
			Links:         notes.Links,

			Create:   notes.Create,
			Write:    notes.Write,
			Replace:  notes.Replace,
			Move:     notes.Move,
			Rename:   notes.Rename,
			Linking:  notes.Linking,
			AddLinks: notes.AddLinks,

			Read:       notes.Read,
			GetTitles:  notes.GetTitles,
			RemoveNote: notes.RemoveNote,
			FocusNote:  cfg.FocusNote(opened.Queries(), opened.State.GetWindow()),
		},
	}
}

// makeOpenVault is the window moved to another vault, as a tool asks for it.
//
// The session that asked is served for the vault that is going and ends with
// it, so what went wrong is said here. A window that cannot be moved serves no
// tool that would move it.
func makeOpenVault(opened *editor.Installation, out io.Writer) func(context.Context, domain.Vault) error {
	if opened.State.OpenVault == nil {
		return nil
	}
	return func(ctx context.Context, v domain.Vault) error {
		err := opened.State.OpenVault(ctx, v)
		if err != nil {
			fmt.Fprintln(out, "agents:", err)
		}
		return err
	}
}
