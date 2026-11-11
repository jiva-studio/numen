package container

import "github.com/jiva-studio/numen/libs/core/kernel/usecase/note"

// NoteScenarios is everything that changes the notes of a vault, grouped; the container builds each
// use case on its own. The window and the agent tools use the same set, so both enforce the same
// rules and a dependency is configured once for both.
type NoteScenarios struct {
	Edit    NoteEditing
	Place   NotePlacing
	Connect NoteConnecting
	Refer   NoteReferring
}

// NoteEditing reads and changes the text of a note.
type NoteEditing struct {
	Read    note.Read
	Write   note.Write
	Create  note.Create
	Replace note.Replace
	Linking note.EditLinks
}

// NotePlacing moves, renames and removes notes.
type NotePlacing struct {
	Move       note.Move
	Rename     note.Rename
	Remove     note.Remove
	RemoveNote note.RemoveNote
	EmptyTrash note.EmptyTrash
}

// NoteConnecting changes the relationships between notes.
type NoteConnecting struct {
	SaveConnection   note.SaveConnection
	RemoveSide       note.RemoveSide
	RemoveConnection note.RemoveConnection
	Join             note.Join
	CreateInRelation note.CreateInRelation
}

// NoteReferring adds the links of notes.
type NoteReferring struct {
	AddLinks note.AddLinks
}

func trackConnections(c NoteConnecting, gate note.WriteGate, follow note.IndexFollow) NoteConnecting {
	track := note.NewTrackWrites(gate, follow)
	c.SaveConnection = note.TrackSaveConnection(c.SaveConnection, track)
	c.RemoveSide = note.TrackRemoveSide(c.RemoveSide, track)
	c.RemoveConnection = note.TrackRemoveConnection(c.RemoveConnection, track)
	c.Join = note.TrackJoin(c.Join, track)
	return c
}
