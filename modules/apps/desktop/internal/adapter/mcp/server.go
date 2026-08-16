// Package mcp is a driving adapter: it turns an agent's tool calls into calls
// on the core, and results into answers an agent can act on.
//
// It knows nothing of the window, of the webview toolkit or of the schema the
// interface is generated from. That is what keeps serving these tools from a
// binary of their own a matter of an entry point and a transport rather than a
// redesign.
package mcp

import (
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/lint"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

// Version is what an agent is told it is talking to.
const Version = "0.1.0"

// Core is everything the tools work through. Every field is a use case or a
// query the rest of the application already has: nothing about a vault is
// decided here.
type Core struct {
	Vault domain.Vault
	// Root is where the vault sits on this machine. An agent that can open
	// files joins it to a path itself; one that cannot reads through a tool.
	Root string

	Readers port.VaultReaders
	Notes   port.NoteQueries

	Search        note.Search
	Neighbourhood note.ShowNeighbourhood
	Links         note.ShowLinks
	Problems      lint.Linter

	Create  note.Create
	Write   note.Write
	Move    note.Move
	Remove  note.Remove
	Linking note.Linking
}

// New builds the server an agent connects to.
func New(core Core) *sdk.Server {
	server := sdk.NewServer(
		&sdk.Implementation{
			Name:        "numen",
			Title:       "numen",
			Description: "Notes with typed links and spaced repetition.",
			Version:     Version,
		},
		&sdk.ServerOptions{Instructions: instructions(core)},
	)

	addNoteTools(server, core)
	addLinkTools(server, core)
	addVaultTools(server, core)
	return server
}

// instructions is what an agent is told once, before it calls anything.
//
// The vault's location lives here rather than in every answer. It is the same
// string every time, a note has one address and not two, and an agent that
// joins them itself pays for the root once instead of on every result.
func instructions(core Core) string {
	var b strings.Builder
	b.WriteString("These tools work on one numen vault: markdown notes in an ordinary folder, ")
	b.WriteString("joined by typed links into a graph.\n\n")

	fmt.Fprintf(&b, "The vault %q is at %s on this machine.\n", core.Vault.Name, core.Root)
	b.WriteString("Every note is addressed by its path relative to that folder, with forward ")
	b.WriteString("slashes — `notes/entropy.md`. That path is what every tool takes and returns. ")
	b.WriteString("To open a note as a file, join it to the folder above; if you cannot read ")
	b.WriteString("files, `note_read` gives you the same text.\n\n")

	b.WriteString("What is worth knowing before changing anything:\n")
	b.WriteString("- A note is named by its file. Renaming a note renames the file.\n")
	b.WriteString("- A link written as a name finds its note wherever it moves to, so moving ")
	b.WriteString("notes between folders is safe and does not need links rewritten.\n")
	b.WriteString("- Two notes filed under one name make every link written by that name ")
	b.WriteString("ambiguous. `vault_problems` lists them.\n")
	b.WriteString("- Folders are for the person's convenience. The hierarchy the product ")
	b.WriteString("draws is the parent and child links, not the folder tree.\n")
	b.WriteString("- Prose is yours to write; the frontmatter is the person's. Change it with ")
	b.WriteString("the link tools rather than by writing the file yourself.\n")
	b.WriteString("- A removed note goes to the vault's trash rather than being destroyed.\n\n")

	b.WriteString("Changes appear immediately in the window the person has open, so work in ")
	b.WriteString("small steps they can follow.\n")
	return b.String()
}
