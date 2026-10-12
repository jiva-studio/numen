// Package mcp is a driving adapter: it turns an agent's tool calls into calls
// on the core, and results into answers an agent can act on.
//
// It knows nothing of the window, of the webview toolkit or of the schema the
// interface is generated from. Serving these tools from a binary of their own
// is an entry point and a transport.
package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/file"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// Version is what an agent is told it is talking to.
const Version = "0.1.0"

// Core is everything the tools work through, in the four things a vault is
// worked as. Every field of every group is a use case or a query the rest of
// the application already has: nothing about a vault is decided here.
type Core struct {
	// Showing is the vault the tools work, asked at every call so that they
	// follow the window.
	Showing func() CurrentVault

	// GetSummary says how much of the vault has been read, and GetProblems what
	// in it could not be acted on. A tool is added where its use case is here.
	GetSummary  *vaults.GetSummary
	GetProblems *vaults.GetProblems

	// Readers is the vault's files, which the notes, the documents and the
	// window are all read out of.
	Readers port.VaultReaders

	// Read reads a range of any file of the vault.
	Read file.Read

	// Attending is what the person has open, asked at every call so that an
	// agent reads the window as it stands. Without it an agent is told nothing
	// of what is in front of anybody.
	Attending func() domain.OpenTabs

	// GetCurrentCard returns the card the person is looking at, asked at every call for
	// the same reason. Without it an agent is told nothing of what card anybody
	// is on.
	GetCurrentCard func() CurrentCard

	Notes   Notes
	Vaults  Vaults
	Cards   Cards
	Sources Sources
}

// Notes is a vault's notes: what is asked of them, and what changes them.
type Notes struct {
	Queries       port.NoteQueries
	Search        search.Search
	Neighbourhood note.ShowNeighbourhood
	Links         note.ShowLinks

	Create   note.Create
	Write    note.Write
	Replace  note.Replace
	Move     note.Move
	Rename   note.Rename
	Linking  note.EditLinks
	AddLinks note.AddLinks

	Read       note.Read
	GetTitles  note.GetTitles
	RemoveNote note.RemoveNote
	// FocusNote puts a note in front of the person. Without it the tool that does is
	// not added.
	FocusNote *note.FocusNote
}

// Vaults is the list of vaults this installation holds, and what a person does
// to it. Without a list an agent is told of the vault it is working and of no
// other, and each tool is served where what it works through is here.
type Vaults struct {
	KnownVaults *vaults.KnownVaults
	GetVault    *vaults.GetVault
	// PickFolder puts this machine's own folder dialog in front of the person,
	// for a vault added without a path.
	PickFolder *vaults.PickFolder
	// Add turns a folder into a vault, Rename is what a person calls one, and
	// Unregister takes one off the list.
	Add        *vaults.Add
	Rename     *vaults.Rename
	Unregister *vaults.Unregister
	// Opens puts another vault in the window. The tools are served for the vault
	// that is going, so the session asking for the swap ends with it.
	Opens func(context.Context, domain.Vault) error
}

// Cards is the decks and stencils a vault is arranged into. Read takes a deck
// or a stencil, List the stencils the vault holds, Write puts a deck back,
// Create makes a stencil and RenameField gives one of a stencil's fields a
// different name in every card it cuts.
type Cards struct {
	Read        cards.Read
	List        cards.List
	Write       cards.Write
	Create      cards.Create
	RenameField cards.RenameField
	// DeckEdit holds a deck's body open so that one card can be changed and every
	// other byte left as it arrived.
	DeckEdit func(body string) *format.DeckFile
}

// Sources holds the use cases of the tools for the documents a vault keeps beside its
// notes. A tool whose use case is left out answers that the vault cannot do it, except
// the artifact and import tools, which are not added.
type Sources struct {
	GetDocumentList *source.GetDocumentList
	GetDocumentText *source.GetDocumentText
	// StartRecognition is held even on a machine holding none of the models: what
	// is missing is fetched behind whoever asked.
	StartRecognition     *source.StartRecognition
	GetRecognitionStatus *source.GetRecognitionStatus
	// StartTranscription queues the transcription of a recording. Without it the
	// tool that asks for one is served and answers that this installation cannot.
	StartTranscription        *source.StartTranscription
	GetTranscriptionReadiness *source.GetTranscriptionReadiness
	// CreateAndImportURL makes the file a web address is kept in and fetches what is
	// at that address. Without it the tool that imports one is not added.
	CreateAndImportURL *source.CreateAndImportURL
	// FocusPassage opens a document in front of the person at a passage. Without it the
	// tool that does is not added.
	FocusPassage *source.FocusPassage
	// The artifacts list, read and correct what was made from a file. Without
	// them the tools for those artifacts are not added.
	GetArtifactList  *source.GetArtifactList
	GetArtifactText  *source.GetArtifactText
	WriteCorrections *source.WriteCorrections
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
	addFileReadingTools(server, core)
	addCardTools(server, core)
	addLinkTools(server, core)
	addVaultTools(server, core)
	addVaultsTools(server, core)
	addViewTools(server, core)
	addWindowTools(server, core)
	addSourceTools(server, core)
	addArtifactTools(server, core)
	addArtifactWriteTool(server, core)
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

	addNoteReadTools(server, core)
	addCardReadTools(server, core)
	addLinkReadTools(server, core)
	addSourceReadTools(server, core)
	addVaultGet(server, core)
	return server
}

// NewReviewServer builds the server the window a person runs their cards in
// serves: everything that reads, and the cards of a deck.
//
// A deck and a stencil are what a vault is arranged into, and nothing here
// makes one. Nothing here writes a note, a link or a document either.
func NewReviewServer(core Core) *sdk.Server {
	server := sdk.NewServer(
		&sdk.Implementation{
			Name:        "numen",
			Title:       "numen",
			Description: "Notes with typed links and spaced repetition.",
			Version:     Version,
		},
		&sdk.ServerOptions{Instructions: getReviewInstructions(core)},
	)

	addNoteReadTools(server, core)
	addCardReadTools(server, core)
	addCardEditTools(server, core)
	addCardFocus(server, core)
	addLinkReadTools(server, core)
	addSourceReadTools(server, core)
	addVaultGet(server, core)
	return server
}
