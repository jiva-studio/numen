package container

import "github.com/jiva-studio/numen/libs/core/kernel/usecase/note"

// noteUseCases is everything that changes the notes of a vault, grouped; the container builds each
// use case on its own. The window and the agent tools use the same set, so both enforce the same
// rules and a dependency is configured once for both.
type noteUseCases struct {
	Edit       noteTextUseCases
	Properties noteProperties
	Place      notePlacementUseCases
	Connect    noteRelationUseCases
}

// noteTextUseCases reads and changes the text of a note.
type noteTextUseCases struct {
	Read    note.Read
	Write   note.Write
	Create  note.Create
	Linking note.EditLinks
}

// notePlacementUseCases moves, renames and removes notes.
type notePlacementUseCases struct {
	Move       note.Move
	Rename     note.Rename
	Remove     note.Remove
	RemoveNote note.RemoveNote
	EmptyTrash note.EmptyTrash
}

// noteRelationUseCases changes the relationships between notes.
type noteRelationUseCases struct {
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
