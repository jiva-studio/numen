package container

import "github.com/jiva-studio/numen/libs/core/kernel/usecase/note"

// noteUseCases is everything that changes the notes of a vault, grouped; the container builds each
// use case on its own. The window and the agent tools use the same set, so both enforce the same
// rules and a dependency is configured once for both.
type noteUseCases struct {
	Edit       noteEditing
	Properties noteProperties
	Place      notePlacing
	Connect    noteConnecting
}

// noteEditing reads and changes the text of a note.
type noteEditing struct {
	Read    note.Read
	Write   note.Write
	Create  note.Create
	Linking note.EditLinks
}

// notePlacing moves, renames and removes notes.
type notePlacing struct {
	Move       note.Move
	Rename     note.Rename
	Remove     note.Remove
	RemoveNote note.RemoveNote
	EmptyTrash note.EmptyTrash
}

// noteConnecting changes the relationships between notes.
type noteConnecting struct {
	SaveConnection       note.SaveConnection
	DeleteConnectionSide note.DeleteConnectionSide
	DeleteConnection     note.DeleteConnection
	Join                 note.Join
	CreateInRelation     note.CreateInRelation
}

// noteProperties changes the frontmatter keys of notes.
type noteProperties struct {
	Frontmatter note.WriteFrontmatter
}
