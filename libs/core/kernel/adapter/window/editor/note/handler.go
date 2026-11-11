// Package note serves a window's NoteService: what the notes of the shown vault hold, the writes that
// change them, and the stream of edits being made to them.
package note

import (
	"errors"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/broadcast"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/outline"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Vault names the vault the window shows. A window showing none returns a vault with no ID.
type Vault interface {
	GetCurrentVault() domain.Vault
}

// Writes counts the writes the window waits for before it closes, and reports whether the index
// followed a write.
type Writes interface {
	BeginWrite() bool
	FinishWrite()
	IsIndexOutdated(err error) bool
	NoteWritten()
}

// Session is the window a handler serves: the vault it shows and the writes it waits for.
type Session interface {
	Vault
	Writes
}

// Edits is the stream of edits being made to the notes of the window.
type Edits interface {
	SubscribeEdits() (<-chan domain.Edit, func())
}

var (
	errClosing = broadcast.ErrClosing
	errNoVault = errors.New("the window has no vault")
)

func getShownVault(window Vault) (domain.Vault, error) {
	v := window.GetCurrentVault()
	if v.ID == "" {
		return domain.Vault{}, connect.NewError(connect.CodeFailedPrecondition, errNoVault)
	}
	return v, nil
}

// Handler answers NoteService. Each question it answers is held by one of the parts it is made of.
type Handler struct {
	*Prose
	*Outline
	*Links
	*Connections

	edits Edits
}

// NewHandler is the handler of NoteService made of its parts.
func NewHandler(edits Edits, prose *Prose, outline *Outline, links *Links, connections *Connections) *Handler {
	return &Handler{Prose: prose, Outline: outline, Links: links, Connections: connections, edits: edits}
}

// Addresses turns the links a request names into the addresses of notes in the shown vault.
type Addresses struct {
	vault Vault
	get   *query.GetLinkAddress
}

// NewAddresses is the addresses of the vault the window shows.
func NewAddresses(window Vault, get *query.GetLinkAddress) *Addresses {
	return &Addresses{vault: window, get: get}
}

// Prose reads, writes, creates and renames notes.
type Prose struct {
	vault     Vault
	writes    Writes
	addresses *Addresses
	read      *note.Read
	save      *note.SaveProse
	create    *note.CreateNote
	rename    *note.Rename
}

// NewProse is the part of NoteService that reads and changes the prose of notes.
func NewProse(
	session Session, addresses *Addresses,
	read *note.Read, save *note.SaveProse, create *note.CreateNote, rename *note.Rename,
) *Prose {
	return &Prose{
		vault: session, writes: session, addresses: addresses, read: read, save: save, create: create, rename: rename,
	}
}

// Outline answers what a note stands among: the note a vault opens on, its neighborhood and the
// headings of notes.
type Outline struct {
	vault           Vault
	getOpeningNote  *query.GetOpeningNote
	getNeighborhood *outline.GetNeighborhood
	listHeadings    *outline.ListHeadings
}

// NewOutline is the part of NoteService that answers what a note stands among.
func NewOutline(
	window Vault, opening *query.GetOpeningNote, neighborhood *outline.GetNeighborhood, headings *outline.ListHeadings,
) *Outline {
	return &Outline{vault: window, getOpeningNote: opening, getNeighborhood: neighborhood, listHeadings: headings}
}

// Links writes the links of notes and answers where links land and which notes link to a note.
type Links struct {
	vault            Vault
	writes           Writes
	addresses        *Addresses
	resolveLinks     *query.ResolveLinks
	editLinks        *note.EditLinks
	createInRelation *note.CreateInRelation
	getBacklinks     *query.GetBacklinks
}

// NewLinks is the part of NoteService that works the links of notes.
func NewLinks(
	session Session, addresses *Addresses,
	resolve *query.ResolveLinks, edit *note.EditLinks, create *note.CreateInRelation, backlinks *query.GetBacklinks,
) *Links {
	return &Links{
		vault: session, writes: session, addresses: addresses, resolveLinks: resolve, editLinks: edit,
		createInRelation: create, getBacklinks: backlinks,
	}
}

// Connections reads and changes the relationship between two notes.
type Connections struct {
	vault            Vault
	writes           Writes
	getConnection    *query.GetConnection
	saveConnection   *note.SaveConnection
	removeSide       *note.RemoveSide
	removeConnection *note.RemoveConnection
	join             *note.Join
}

// NewConnections is the part of NoteService that works the connections between notes.
func NewConnections(
	session Session,
	get *query.GetConnection, save *note.SaveConnection, removeSide *note.RemoveSide,
	remove *note.RemoveConnection, join *note.Join,
) *Connections {
	return &Connections{
		vault: session, writes: session, getConnection: get, saveConnection: save, removeSide: removeSide,
		removeConnection: remove, join: join,
	}
}
