// Package note serves a window's NoteService: what the notes of the shown vault hold, the writes that
// change them, and the stream of edits being made to them.
package note

import (
	"errors"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/broadcast"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Vault names the vault the window shows. A window showing none returns a vault with no ID.
type Vault interface {
	GetCurrentVault() domain.Vault
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
	proseCalls
	outlineCalls
	linksCalls
	connectionsCalls

	edits Edits
}

// NewHandler is the handler of NoteService made of its parts.
func NewHandler(edits Edits, prose *Prose, outline *Outline, links *Links, connections *Connections) *Handler {
	return &Handler{proseCalls: prose, outlineCalls: outline, linksCalls: links, connectionsCalls: connections, edits: edits}
}

// Prose reads, writes, creates and renames notes.
type Prose struct {
	vault  Vault
	read   *note.Read
	save   *note.Write
	create *note.Create
	rename *note.Rename
	writes Writes
}

// NewProse is the part of NoteService that reads and changes the prose of notes.
func NewProse(
	window Vault, read *note.Read, save *note.Write, create *note.Create, rename *note.Rename, writes Writes,
) *Prose {
	return &Prose{vault: window, read: read, save: save, create: create, rename: rename, writes: writes}
}

// Outline answers what a note stands among: the note a vault opens on, its related notes and the
// headings of notes.
type Outline struct {
	vault           Vault
	getOpeningNote  *query.GetOpeningNote
	getRelatedNotes *query.GetRelatedNotesView
	listHeadings    *query.ListHeadings
}

// NewOutline is the part of NoteService that answers what a note stands among.
func NewOutline(
	window Vault, opening *query.GetOpeningNote, relatedNotes *query.GetRelatedNotesView, headings *query.ListHeadings,
) *Outline {
	return &Outline{vault: window, getOpeningNote: opening, getRelatedNotes: relatedNotes, listHeadings: headings}
}

// Links writes the links of notes and answers where links land and which notes link to a note.
type Links struct {
	vault         Vault
	resolveLinks  *query.ResolveLinks
	createRelated *note.CreateInRelation
	getBacklinks  *query.GetBacklinks
	writes        Writes
}

// NewLinks is the part of NoteService that works the links of notes.
func NewLinks(
	window Vault,
	resolve *query.ResolveLinks, create *note.CreateInRelation, backlinks *query.GetBacklinks,
	writes Writes,
) *Links {
	return &Links{
		vault: window, resolveLinks: resolve, createRelated: create, getBacklinks: backlinks, writes: writes,
	}
}

// Connections reads and changes the relationship between two notes.
type Connections struct {
	vault            Vault
	getConnection    *query.GetConnection
	saveConnection   *note.SaveConnection
	removeSide       *note.RemoveSide
	removeConnection *note.RemoveConnection
	join             *note.Join
}

// NewConnections is the part of NoteService that works the connections between notes.
func NewConnections(
	window Vault,
	get *query.GetConnection, save *note.SaveConnection, removeSide *note.RemoveSide,
	remove *note.RemoveConnection, join *note.Join,
) *Connections {
	return &Connections{
		vault: window, getConnection: get, saveConnection: save, removeSide: removeSide,
		removeConnection: remove, join: join,
	}
}
