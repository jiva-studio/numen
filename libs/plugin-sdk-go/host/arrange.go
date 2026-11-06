package host

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

// NoteRename is the outcome of renaming a note: its path and title, whether its filename carries
// the title, and the move of its file, nil when the file kept its path.
type NoteRename struct {
	Path        string
	Title       string
	IsFilenamed bool
	Move        *domain.MoveResult
}

// NoteMove is the outcome of moving one note. Move.From is always set; Move.To and Move.IsMoved
// only when the note moved. Err is why it did not: ErrNotExist, ErrNotANote, ErrAlreadyExists,
// ErrStale, domain.ErrInvalidName or another error.
type NoteMove struct {
	Move domain.MoveResult
	Err  error
}

// NoteRemoval is the outcome of removing one note: where its file is in the trash and the notes whose
// links now resolve to nothing. Err is why it was not removed: ErrNotExist, ErrNotANote for a folder
// or a file that is not a note, or another error.
type NoteRemoval struct {
	Path        string
	TrashPath   string
	BrokenLinks []string
	Err         error
}

// VaultProblems is what the checks of a vault found, and the names of the checks the answer covers.
type VaultProblems struct {
	Checks []string
	Found  []domain.VaultProblem
}

// NoteArranger renames, moves and removes the notes of a vault, with the links that stop resolving
// repaired by the host, and runs the checks of the vault. The changes need vault:write and the checks
// vault:read.
type NoteArranger interface {
	// RenameNote gives the note another title and moves its file to the filename of the title.
	RenameNote(ctx context.Context, path, title string) (NoteRename, error)
	// MoveNotes moves each note to the folder, keeping its filename. An empty folder is the vault root.
	// The answer has one entry per path, in order; a note that fails does not stop the others.
	MoveNotes(ctx context.Context, paths []string, folder string) ([]NoteMove, error)
	// RemoveNotes moves each note to the vault's trash. The answer has one entry per path, in order.
	RemoveNotes(ctx context.Context, paths []string) ([]NoteRemoval, error)
	// GetProblems runs the checks named (parse, frontmatter, ambiguous, dangling), or every check that
	// is not opt-in when none is named. dangling is opt-in.
	GetProblems(ctx context.Context, checks []string) (VaultProblems, error)
}

// LinkAddition is one link to write into the note at From, which the caller read as Baseline.
type LinkAddition struct {
	From     string
	Baseline domain.FileStat
	Link     domain.Link
}

// LinkAdded is the outcome of one addition: the note after the write, or why the link was not
// written. A stale baseline is ErrStale.
type LinkAdded struct {
	Stat domain.FileStat
	Err  error
}

// LinkEditor edits the frontmatter links of the notes of a vault. A baseline the note moved past is
// ErrStale. The edits need vault:write.
type LinkEditor interface {
	// AddLinks writes the links, grouped by note: the links of one note are one write against one
	// baseline, and succeed or fail together. The answer has one entry per addition, in order.
	AddLinks(ctx context.Context, additions []LinkAddition) ([]LinkAdded, error)
	// UpdateLinks changes the role, type, label and note of the links the note at from has to the
	// target, keeping the target. An empty field is kept.
	UpdateLinks(ctx context.Context, from string, to domain.Address, change domain.Link, baseline domain.FileStat) (domain.FileStat, error)
	// RemoveLinks removes the frontmatter links the note at from has to the target, of every role, or
	// only those of the role when one is given. The note at the other end is unchanged.
	RemoveLinks(ctx context.Context, from string, to domain.Address, role domain.LinkRole, baseline domain.FileStat) (domain.FileStat, error)
}
