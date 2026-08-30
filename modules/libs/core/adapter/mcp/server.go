// Package mcp is a driving adapter: it turns an agent's tool calls into calls
// on the core, and results into answers an agent can act on.
//
// It knows nothing of the window, of the webview toolkit or of the schema the
// interface is generated from. Serving these tools from a binary of their own
// is an entry point and a transport.
package mcp

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	format "github.com/jiva-studio/numen/modules/libs/core/cards"
	"github.com/jiva-studio/numen/modules/libs/core/check"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// Version is what an agent is told it is talking to.
const Version = "0.1.0"

// Core is everything the tools work through. Every field is a use case or a
// query the rest of the application already has: nothing about a vault is
// decided here.
type Core struct {
	// Showing is the vault the tools work, asked at every call so that they
	// follow the window.
	Showing func() Shown

	Readers port.VaultReaders
	Notes   port.NoteQueries

	// View is the person's window, where there is one. Without it an agent is
	// served the vault and nothing that puts a note in front of anybody.
	View port.View

	// Vaults is the list of vaults this installation holds. Without it an agent
	// is told of the vault it is working and of no other.
	Vaults port.VaultRegistry
	// Choosing puts this machine's own folder picker in front of the person, for
	// a vault added without a path. Without it a folder is named or nothing is
	// added.
	Choosing port.Folders
	// Adding turns a folder into a vault, Renaming is what a person calls one,
	// and Forgetting takes one off the list. Each tool is served where what it
	// works through is here.
	Adding     *usecase.Add
	Renaming   *usecase.Rename
	Forgetting *usecase.Forget
	// Opens puts another vault in the window. The tools are served for the vault
	// that is going, so the session asking for the swap ends with it.
	Opens func(context.Context, domain.Vault) error

	// Sources and Recognise are the documents a vault holds beside its notes.
	// Without them the tools for those documents are not added.
	Sources   port.SourceQueries
	Recognise Recognising
	// Derived is where a reading of a document is kept. Without it a document
	// stands on its own bytes, which for a scan is nothing.
	Derived port.DerivedStores
	// Documents reads a format that needs a library, for a document standing on
	// its own bytes.
	Documents port.Documents

	Search        search.Search
	Neighbourhood note.ShowNeighbourhood
	Links         note.ShowLinks
	Problems      check.Checks

	// Cards reads a deck or a stencil, Stencils lists the stencils the vault
	// holds, Cuts puts a deck back, Cutting makes a stencil and FieldRename
	// gives one of a stencil's fields a different name in every card it cuts.
	Cards       cards.Read
	Stencils    cards.List
	Cuts        cards.Write
	Cutting     cards.Create
	FieldRename cards.RenameField
	// DeckBody is the markdown a deck of cards is written as, and StencilBody
	// the markdown a stencil's faces are. A tool changes cards and hands them
	// back; what the file then reads as is the format's.
	DeckBody    func(d format.Deck) (string, error)
	StencilBody func(preamble string, faces []format.Face, tail string) (string, error)

	Create  note.Create
	Write   note.Write
	Replace note.Replace
	Move    note.Move
	Rename  note.Rename
	Remove  note.Remove
	Linking note.Linking
}

// Shown is the vault a call is answered about: the vault itself, and where it
// sits on this machine. An agent that can open files joins the folder to a
// path itself; one that cannot reads through a tool.
type Shown struct {
	Vault domain.Vault
	Root  string
}

// shown is the vault this call is about. A build that named none answers about
// no vault at all.
func (c Core) shown() Shown {
	if c.Showing == nil {
		return Shown{}
	}
	return c.Showing()
}

// One is a Core working one vault for as long as it is served.
func One(v domain.Vault, root string) func() Shown {
	return func() Shown { return Shown{Vault: v, Root: root} }
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
	addCardTools(server, core)
	addLinkTools(server, core)
	addVaultTools(server, core)
	addVaultsTools(server, core)
	addViewTools(server, core)
	addSourceTools(server, core)
	return server
}

// NewReading builds a server whose every tool reads. Nothing it serves
// reaches a writer, so an agent answering from it changes nothing.
func NewReading(core Core) *sdk.Server {
	server := sdk.NewServer(
		&sdk.Implementation{
			Name:        "numen",
			Title:       "numen",
			Description: "Notes with typed links and spaced repetition.",
			Version:     Version,
		},
		&sdk.ServerOptions{Instructions: readingInstructions(core)},
	)

	addNoteReadingTools(server, core)
	addCardReadingTools(server, core)
	addLinkReadingTools(server, core)
	addSourceReadingTools(server, core)
	addVaultGet(server, core)
	return server
}

// namingOrder is how a note comes by the name it is shown under. The
// instructions and the tool that changes it say it in these words.
const namingOrder = "A note is shown by its title, else by its filename."

// instructions is what an agent is told once, before it calls anything.
func instructions(core Core) string {
	var b strings.Builder
	opening(&b, core)

	b.WriteString("What is worth knowing before changing anything:\n")
	b.WriteString("- " + namingOrder + " `note_rename` brings whichever of the three names it ")
	b.WriteString("into line and files the note under the new name.\n")
	b.WriteString("- A link written as a name finds its note wherever it moves to, so moving ")
	b.WriteString("notes between folders is safe and does not need links rewritten.\n")
	b.WriteString("- Two notes filed under one name make every link written by that name ")
	b.WriteString("ambiguous. `vault_problems` lists them.\n")
	b.WriteString("- Folders are for the person's convenience. The hierarchy the product ")
	b.WriteString("draws is the parent and child links, not the folder tree.\n")
	b.WriteString("- Prose is yours to write; the frontmatter is the person's. Change it with ")
	b.WriteString("the link and rename tools rather than by writing the file yourself.\n")
	b.WriteString("- A removed note goes to the vault's trash rather than being destroyed.\n\n")

	b.WriteString("A vault holds books and papers beside its notes, and asking them is not ")
	b.WriteString("like asking a note:\n")
	b.WriteString("- Search with the person's own words before searching with your own. A ")
	b.WriteString("book's sections are searched by name, and a section named what was asked ")
	b.WriteString("for is what the search answers with.\n")
	b.WriteString("- A search answers with several places of one book. Show every place you ")
	b.WriteString("speak about: `source_show` takes the rest under `also`, and the person is ")
	b.WriteString("taken to the first.\n")
	b.WriteString("- A passage is a window cut to a size and it ends where it was cut, which ")
	b.WriteString("is mid-sentence as often as not. Read on with `source_read` before saying ")
	b.WriteString("a book does not say something.\n")
	b.WriteString("- Name a passage in what you write as a link, so the person can go to it:\n")
	b.WriteString("  `[the Remuna episode](numen:library%2FA%20Book.pdf?start=62690&length=1246)`\n")
	b.WriteString("  The path is percent-encoded, and the start and length are the ones the ")
	b.WriteString("search gave you. Write the link where you speak about the passage, not in ")
	b.WriteString("a list at the end.\n\n")

	b.WriteString("Changes appear immediately in the window the person has open, so work in ")
	b.WriteString("small steps they can follow.\n")
	return b.String()
}

// opening is what every agent is told first: what a vault is, where this one
// is, and how a note is addressed. A note has one address, and an agent that
// joins it to the root pays for the root once.
func opening(b *strings.Builder, core Core) {
	b.WriteString("These tools work on one numen vault: markdown notes in an ordinary folder, ")
	b.WriteString("joined by typed links into a graph.\n\n")

	shown := core.shown()
	fmt.Fprintf(b, "The vault %q is at %s on this machine.\n", shown.Vault.Name, shown.Root)
	b.WriteString("Every note is addressed by its path relative to that folder, with forward ")
	b.WriteString("slashes — `notes/entropy.md`. That path is what every tool takes and returns. ")
	b.WriteString("To open a note as a file, join it to the folder above; if you cannot read ")
	b.WriteString("files, `note_read` gives you the same text.\n\n")
}

// readingInstructions is what an agent served the reading tools is told once,
// before it calls anything.
func readingInstructions(core Core) string {
	var b strings.Builder
	opening(&b, core)

	b.WriteString("Everything you can do here reads. Nothing you can call writes a note, a ")
	b.WriteString("card or a document, and nothing you call moves the person's window.\n\n")

	b.WriteString("Say where each part of an answer came from, in words the person can find ")
	b.WriteString("it by: the note's path and the heading it stands under, the book's name ")
	b.WriteString("and its chapter or page. Write it where you say the thing, not in a list ")
	b.WriteString("at the end. There is nothing here to open a `numen:` link with, so a link ")
	b.WriteString("is not a place a person can go.\n\n")

	b.WriteString("Asking a book is not like asking a note:\n")
	b.WriteString("- Search with the person's own words before searching with your own. A ")
	b.WriteString("book's sections are searched by name, and a section named what was asked ")
	b.WriteString("for is what the search answers with.\n")
	b.WriteString("- A passage is a window cut to a size and it ends where it was cut, which ")
	b.WriteString("is mid-sentence as often as not. Read on with `source_read` before saying ")
	b.WriteString("a book does not say something.\n")
	return b.String()
}
