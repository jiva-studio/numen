package note

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/naming"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/shared/note/edit"
)

// UseCases is everything that changes the notes of a vault, grouped. The window and the agent tools
// use the same set, so both enforce the same rules and a dependency is configured once for both.
type UseCases struct {
	Edit       Editing
	Properties Properties
	Place      Placing
	Connect    Connecting
}

// Editing reads and changes the text of a note.
type Editing struct {
	Read    Read
	Write   Write
	Create  Create
	Linking EditLinks
}

// Placing moves, renames and removes notes.
type Placing struct {
	Move       Move
	Rename     Rename
	Remove     Remove
	RemoveNote RemoveNote
	EmptyTrash EmptyTrash
}

// Connecting changes the relationships between notes.
type Connecting struct {
	SaveConnection       SaveConnection
	DeleteConnectionSide DeleteConnectionSide
	DeleteConnection     DeleteConnection
	Join                 Join
	CreateInRelation     CreateInRelation
}

// Properties changes the frontmatter keys of notes.
type Properties struct {
	Frontmatter WriteFrontmatter
}

// VaultWriters opens a vault to write notes, move files and empty the trash.
type VaultWriters interface {
	MoveWriting
	TrashEmptiers
}

// IndexSources is the sources the index holds: they are read, searched under a path and moved with their versions.
type IndexSources interface {
	sourceport.SourceRepository
	SourcesUnder
	VersionMover
}

// Deps is what NewUseCases takes. Moves, Admission, Follow, Absent and Sync are optional.
type Deps struct {
	Readers   vaultport.VaultReaders
	Writers   VaultWriters
	Links     noteport.LinkQueries
	Names     edit.NameQueries
	Sources   IndexSources
	Index     edit.IndexUpdater
	Now       machineport.Clock
	Trash     TrashedVersions
	Absent    vaultport.AbsentFiles
	Sync      naming.SyncSetting
	Moves     TellMove
	Admission vaultport.WriteAdmission
	Follow    IndexFollow
}

func NewUseCases(d Deps) UseCases {
	moveDeps := MoveDeps{
		Readers: d.Readers, Writers: d.Writers, Links: d.Links, Names: d.Names,
		Sources: d.Sources, Index: d.Index, Now: d.Now, Drawing: d.Moves, Sync: d.Sync,
	}
	mover := NewMove(moveDeps)
	remove := NewRemove(d.Writers, d.Links, d.Sources, d.Sources, d.Index, d.Admission).WithAbsent(d.Absent)
	create := NewCreate(d.Writers, d.Names, d.Index, d.Now, d.Admission)
	connecting := ConnectionDeps{
		Readers: d.Readers, Writers: d.Writers, Index: d.Index, Now: d.Now, Links: d.Links, Names: d.Names,
		Admission: d.Admission, Follow: d.Follow,
	}
	return UseCases{
		Edit: Editing{
			Read:    NewRead(d.Readers),
			Write:   NewWrite(d.Readers, d.Writers, d.Index, d.Now, d.Admission),
			Create:  create,
			Linking: NewEditLinks(d.Readers, d.Writers, d.Index, d.Now),
		},
		Properties: Properties{Frontmatter: NewWriteFrontmatter(d.Readers, d.Writers, d.Index, d.Now)},
		Place: Placing{
			Move:       mover,
			Rename:     NewRename(moveDeps, mover, d.Admission),
			Remove:     remove,
			RemoveNote: NewRemoveNote(remove, d.Readers),
			EmptyTrash: NewEmptyTrash(d.Writers, d.Trash, d.Admission),
		},
		Connect: Connecting{
			SaveConnection:       NewSaveConnection(connecting),
			DeleteConnectionSide: NewDeleteConnectionSide(connecting),
			DeleteConnection:     NewDeleteConnection(connecting),
			Join:                 NewJoin(connecting),
			CreateInRelation:     NewCreateInRelation(create, d.Names),
		},
	}
}
